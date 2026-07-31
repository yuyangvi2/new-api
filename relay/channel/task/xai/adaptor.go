package xai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	xaipricing "github.com/QuantumNous/new-api/relay/channel/xai/pricing"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
)

type submitResponse struct {
	RequestID string `json:"request_id"`
	ID        string `json:"id,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
}

type videoInfo struct {
	Duration          int    `json:"duration,omitempty"`
	RespectModeration bool   `json:"respect_moderation,omitempty"`
	URL               string `json:"url,omitempty"`
}

type responseTask struct {
	ID        string         `json:"id,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
	Object    string         `json:"object,omitempty"`
	Model     string         `json:"model,omitempty"`
	Status    string         `json:"status,omitempty"`
	Progress  int            `json:"progress,omitempty"`
	CreatedAt int64          `json:"created_at,omitempty"`
	Usage     map[string]any `json:"usage,omitempty"`
	Video     *videoInfo     `json:"video,omitempty"`
	Error     *struct {
		Message string `json:"message"`
		Type    string `json:"type,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

type TaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey  string
	baseURL string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.baseURL = strings.TrimRight(info.ChannelBaseUrl, "/")
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	taskErr := relaycommon.ValidateMultipartDirect(c, info)
	if taskErr != nil {
		return taskErr
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	mergeXAITopLevelMetadata(c, &req)
	c.Set("task_request", req)
	if strings.EqualFold(info.OriginModelName, "grok-imagine-video-1.5") && strings.TrimSpace(req.Image) == "" && len(req.Images) == 0 {
		return service.TaskErrorWrapperLocal(fmt.Errorf("image is required for grok-imagine-video-1.5"), "invalid_request", http.StatusBadRequest)
	}
	return nil
}

func mergeXAITopLevelMetadata(c *gin.Context, req *relaycommon.TaskSubmitReq) {
	if req == nil {
		return
	}
	var raw map[string]any
	if err := common.UnmarshalBodyReusable(c, &raw); err != nil {
		return
	}
	if req.Metadata == nil {
		req.Metadata = make(map[string]any)
	}
	for _, key := range []string{"resolution", "aspect_ratio"} {
		if value, ok := raw[key]; ok {
			req.Metadata[key] = value
		}
	}
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	price, ok := xaipricing.GrokImagineTaskPrice(req, info.OriginModelName)
	if !ok || info.PriceData.Quota <= 0 {
		return nil
	}
	groupRatio := info.PriceData.GroupRatioInfo.GroupRatio
	if groupRatio < 0 {
		groupRatio = 0
	}
	ratio := price * float64(common.QuotaPerUnit) * groupRatio / float64(info.PriceData.Quota)
	if ratio == 1.0 {
		return nil
	}
	return map[string]float64{"xai_imagine_price": ratio}
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return a.baseURL + "/v1/videos/generations", nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"model":  info.UpstreamModelName,
		"prompt": req.Prompt,
	}
	if req.Duration > 0 {
		body["duration"] = req.Duration
	}
	if req.Seconds != "" {
		if seconds, err := strconv.Atoi(req.Seconds); err == nil && seconds > 0 {
			body["duration"] = seconds
		}
	}
	if req.Image != "" {
		body["image"] = map[string]string{"url": req.Image}
	} else if len(req.Images) > 0 {
		if strings.EqualFold(info.OriginModelName, "grok-imagine-video-1.5") {
			body["image"] = map[string]string{"url": req.Images[0]}
		} else {
			body["images"] = req.Images
		}
	}
	for key, value := range req.Metadata {
		if key == "size" || key == "model" || key == "prompt" {
			continue
		}
		body[key] = value
	}
	marshal, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(marshal), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var upstream submitResponse
	if err := common.Unmarshal(responseBody, &upstream); err != nil {
		return "", nil, service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	upstreamID := upstream.RequestID
	if upstreamID == "" {
		upstreamID = upstream.ID
	}
	if upstreamID == "" {
		upstreamID = upstream.TaskID
	}
	if upstreamID == "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
	}

	clientResp := responseTask{
		ID:        info.PublicTaskID,
		RequestID: info.PublicTaskID,
		Object:    "video",
		Model:     info.OriginModelName,
		Status:    "queued",
		Progress:  0,
	}
	c.JSON(http.StatusOK, clientResp)
	return upstreamID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	uri := strings.TrimRight(baseUrl, "/") + "/v1/videos/" + taskID
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}
	taskResult := relaycommon.TaskInfo{Code: 0}
	switch strings.ToLower(resTask.Status) {
	case "queued", "pending":
		taskResult.Status = model.TaskStatusQueued
	case "processing", "in_progress", "running":
		taskResult.Status = model.TaskStatusInProgress
	case "done", "completed", "succeeded", "success":
		taskResult.Status = model.TaskStatusSuccess
	case "failed", "cancelled", "canceled":
		taskResult.Status = model.TaskStatusFailure
		if resTask.Error != nil && resTask.Error.Message != "" {
			taskResult.Reason = resTask.Error.Message
		} else {
			taskResult.Reason = "task failed"
		}
	}
	if resTask.Progress > 0 && resTask.Progress < 100 {
		taskResult.Progress = fmt.Sprintf("%d%%", resTask.Progress)
	}
	if resTask.Video != nil && strings.TrimSpace(resTask.Video.URL) != "" {
		taskResult.Url = strings.TrimSpace(resTask.Video.URL)
	}
	if costUSD, ok := xaiUsageCostUSD(resTask.Usage); ok {
		taskResult.BillingUnits = costUSD
	}
	return &taskResult, nil
}

func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int {
	quota, _ := a.AdjustBillingOnCompleteWithClamp(task, taskResult)
	return quota
}

func (a *TaskAdaptor) AdjustBillingOnCompleteWithClamp(task *model.Task, taskResult *relaycommon.TaskInfo) (int, *common.QuotaClamp) {
	if task == nil || taskResult == nil || taskResult.BillingUnits <= 0 {
		return 0, nil
	}
	bc := task.PrivateData.BillingContext
	if bc == nil || bc.GroupRatio <= 0 {
		return 0, nil
	}
	return common.QuotaFromFloatChecked(taskResult.BillingUnits * bc.GroupRatio * common.QuotaPerUnit)
}

func xaiUsageCostUSD(usage map[string]any) (float64, bool) {
	if len(usage) == 0 {
		return 0, false
	}
	if ticks, ok := xaiNumericField(usage["cost_in_usd_ticks"]); ok && ticks > 0 {
		return ticks / 10_000_000_000.0, true
	}
	if cost, ok := xaiNumericField(usage["cost_in_usd"]); ok && cost > 0 {
		return cost, true
	}
	return 0, false
}

func xaiNumericField(value any) (float64, bool) {
	var f float64
	switch v := value.(type) {
	case float64:
		f = v
	case float32:
		f = float64(v)
	case int:
		f = float64(v)
	case int64:
		f = float64(v)
	case json.Number:
		parsed, err := strconv.ParseFloat(v.String(), 64)
		if err != nil {
			return 0, false
		}
		f = parsed
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, false
		}
		f = parsed
	default:
		return 0, false
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	data := task.Data
	var err error
	if data, err = sjson.SetBytes(data, "id", task.TaskID); err != nil {
		return nil, errors.Wrap(err, "set id failed")
	}
	if data, err = sjson.SetBytes(data, "request_id", task.TaskID); err != nil {
		return nil, errors.Wrap(err, "set request_id failed")
	}
	if data, err = sjson.SetBytes(data, "object", "video"); err != nil {
		return nil, errors.Wrap(err, "set object failed")
	}
	status := strings.ToLower(string(task.Status))
	if task.Status == model.TaskStatusSuccess {
		status = "completed"
	} else if task.Status == model.TaskStatusFailure {
		status = "failed"
	}
	if data, err = sjson.SetBytes(data, "status", status); err != nil {
		return nil, errors.Wrap(err, "set status failed")
	}
	if task.Status == model.TaskStatusSuccess {
		if data, err = sjson.SetBytes(data, "video.url", taskcommon.BuildProxyURL(task.TaskID)); err != nil {
			return nil, errors.Wrap(err, "set video url failed")
		}
	}
	return data, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}
