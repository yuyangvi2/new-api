package doubao

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/tidwall/gjson"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
)

const seedanceOfficialTaskContextKey = "seedance_official_task_request"

type SeedanceOfficialTaskAdaptor struct {
	taskcommon.BaseBilling
}

func (a *SeedanceOfficialTaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := a.taskRequest(c)
	if err != nil {
		return nil
	}
	billingReq := seedanceOfficialBillingRequest(req)
	modelName := firstNonEmptyString(info.OriginModelName, req.Model)
	hasVideoInput := seedanceOfficialHasVideoInput(req)
	quota, _, pricePerMillionCNY, ok := EstimateSeedanceQuotaForRequest(
		modelName,
		billingReq,
		hasVideoInput,
		info.PriceData.GroupRatioInfo.GroupRatio,
	)
	if ok && info.TaskRelayInfo != nil {
		hasVideoInputSnapshot := hasVideoInput
		resolution := stringFromAny(billingReq.Metadata["resolution"])
		if resolution == "" {
			resolution = "720p"
		}
		info.TaskRelayInfo.UsageBilling = &relaycommon.TaskUsageBillingContext{
			PricingSource:      relaycommon.TaskPricingSourceSeedanceOfficialUsage,
			PricePerMillionCNY: pricePerMillionCNY,
			USDExchangeRate:    seedanceUSDExchangeRate(),
			Resolution:         resolution,
			HasVideoInput:      &hasVideoInputSnapshot,
		}
	}
	if !ok || info.PriceData.Quota <= 0 {
		return nil
	}
	ratio := float64(quota) / float64(info.PriceData.Quota)
	if ratio == 1.0 {
		return nil
	}
	return map[string]float64{"seedance_official_estimated_price": ratio}
}

func (*SeedanceOfficialTaskAdaptor) AdjustBillingOnComplete(task *model.Task, _ *relaycommon.TaskInfo) int {
	if task == nil || task.Quota <= 0 {
		return 0
	}
	return task.Quota
}

func (*SeedanceOfficialTaskAdaptor) AdjustBillingOnCompleteWithClamp(task *model.Task, taskResult *relaycommon.TaskInfo) (int, *common.QuotaClamp) {
	if task == nil || taskResult == nil || taskResult.TotalTokens <= 0 {
		return 0, nil
	}
	billingContext := task.PrivateData.BillingContext
	if billingContext == nil || billingContext.UsageBilling == nil {
		return 0, nil
	}
	usageBilling := billingContext.UsageBilling
	if usageBilling.PricingSource != relaycommon.TaskPricingSourceSeedanceOfficialUsage || usageBilling.HasVideoInput == nil {
		return 0, nil
	}

	resolution := strings.TrimSpace(gjson.GetBytes(task.Data, "resolution").String())
	if resolution == "" {
		resolution = usageBilling.Resolution
	}
	pricePerMillionCNY := usageBilling.PricePerMillionCNY
	if resolution != "" && !strings.EqualFold(resolution, usageBilling.Resolution) {
		if price, ok := SeedancePricePerMillionCNY(TaskModelName(task), resolution, *usageBilling.HasVideoInput); ok {
			pricePerMillionCNY = price
		}
	}
	actualQuota, clamp, ok := EstimateSeedanceQuotaFromUsageTokens(
		TaskModelName(task),
		resolution,
		taskResult.TotalTokens,
		*usageBilling.HasVideoInput,
		billingContext.GroupRatio,
		pricePerMillionCNY,
		usageBilling.USDExchangeRate,
	)
	if !ok {
		return 0, nil
	}
	return actualQuota, clamp
}

func (a *SeedanceOfficialTaskAdaptor) GetModelList() []string { return ModelList }

func (a *SeedanceOfficialTaskAdaptor) GetChannelName() string { return "seedance-official" }

func (a *SeedanceOfficialTaskAdaptor) Init(*relaycommon.RelayInfo) {}

func (a *SeedanceOfficialTaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	req, err := parseSeedanceOfficialRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if err := validateSeedanceOfficialRequest(req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	info.Action = constant.TaskActionGenerate
	c.Set(seedanceOfficialTaskContextKey, *req)
	return nil
}

func (a *SeedanceOfficialTaskAdaptor) requestURL(raw string) string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	switch {
	case strings.HasSuffix(base, "/contents/generations/tasks"):
		return base
	case strings.HasSuffix(base, "/v1/video/tasks"):
		return base
	case strings.HasSuffix(base, "/api/v3"):
		return base + "/contents/generations/tasks"
	default:
		return base + "/api/v3/contents/generations/tasks"
	}
}

func (a *SeedanceOfficialTaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return a.requestURL(info.ChannelBaseUrl), nil
}

func (a *SeedanceOfficialTaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(info.ApiKey))
	return nil
}

func (a *SeedanceOfficialTaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := a.taskRequest(c)
	if err != nil {
		return nil, err
	}
	if info.UpstreamModelName != "" {
		req.Model = info.UpstreamModelName
	}
	data, err := common.Marshal(req)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *SeedanceOfficialTaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *SeedanceOfficialTaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	var payload struct {
		ID     string `json:"id"`
		TaskID string `json:"task_id"`
	}
	if err = common.Unmarshal(body, &payload); err != nil {
		return "", nil, service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", body), "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	taskID := lo.Ternary(payload.ID != "", payload.ID, payload.TaskID)
	if taskID == "" {
		return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":               info.PublicTaskID,
		"upstream_task_id": taskID,
	})
	return taskID, body, nil
}

func (a *SeedanceOfficialTaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	req, err := http.NewRequest(http.MethodGet, a.requestURL(baseURL)+"/"+url.PathEscape(taskID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(key))

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *SeedanceOfficialTaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var response SeedanceOfficialResponse
	if err := common.Unmarshal(respBody, &response); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := &relaycommon.TaskInfo{
		Code:   0,
		TaskID: response.ID,
	}
	if response.Usage != nil {
		taskResult.CompletionTokens = response.Usage.CompletionTokens
		taskResult.TotalTokens = response.Usage.TotalTokens
	}
	errorReason := ""
	if response.Error != nil {
		errorReason = firstNonEmptyString(response.Error.Message, response.Error.Code)
	}
	if response.ResponseMetadata != nil && response.ResponseMetadata.Error != nil {
		errorReason = firstNonEmptyString(response.ResponseMetadata.Error.Message, response.ResponseMetadata.Error.Code, errorReason)
	}
	status := strings.ToLower(strings.TrimSpace(response.Status))
	if status == "" {
		if errorReason == "" {
			return nil, errors.Errorf("seedance official response missing status: %s", string(respBody))
		}
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = taskcommon.ProgressComplete
		taskResult.Reason = errorReason
		return taskResult, nil
	}
	switch status {
	case "pending", "queued":
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = taskcommon.ProgressQueued
	case "processing", "running":
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = taskcommon.ProgressInProgress
	case "succeeded", "success", "completed":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = taskcommon.ProgressComplete
		if response.Content != nil {
			taskResult.Url = response.Content.VideoURL
		}
	case "failed", "cancelled", "canceled", "expired":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = taskcommon.ProgressComplete
		taskResult.Reason = firstNonEmptyString(errorReason, response.Status)
	default:
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = taskcommon.ProgressComplete
		taskResult.Reason = firstNonEmptyString(errorReason, fmt.Sprintf("unknown status: %s", response.Status))
	}
	return taskResult, nil
}

func (a *SeedanceOfficialTaskAdaptor) taskRequest(c *gin.Context) (*requestPayload, error) {
	v, exists := c.Get(seedanceOfficialTaskContextKey)
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req, ok := v.(requestPayload)
	if !ok {
		return nil, fmt.Errorf("invalid task request type")
	}
	return &req, nil
}

func (a *SeedanceOfficialTaskAdaptor) ConvertToSeedanceVideo(originTask *model.Task) ([]byte, error) {
	var status string
	switch originTask.Status {
	case model.TaskStatusQueued:
		status = "queued"
	case model.TaskStatusInProgress:
		status = "running"
	case model.TaskStatusSuccess:
		status = "succeeded"
	case model.TaskStatusFailure:
		status = "failed"
	default:
		status = gjson.GetBytes(originTask.Data, "status").String()
	}

	response := SeedanceOfficialResponse{}
	if len(originTask.Data) > 0 {
		_ = common.Unmarshal(originTask.Data, &response)
	}
	response.ID = originTask.TaskID
	response.Model = firstNonEmptyString(TaskModelName(originTask), response.Model)
	response.Status = status
	if originTask.Status == model.TaskStatusSuccess {
		videoURL := originTask.PrivateData.ResultURL
		if response.Content != nil {
			videoURL = firstNonEmptyString(videoURL, response.Content.VideoURL)
		}
		if videoURL != "" {
			response.Content = &SeedanceOfficialResponseContent{VideoURL: videoURL}
		}
	} else {
		response.Content = nil
	}
	if response.Usage != nil && response.Usage.CompletionTokens == 0 && response.Usage.TotalTokens == 0 {
		response.Usage = nil
	}
	if response.ResponseMetadata != nil && response.ResponseMetadata.Error == nil {
		response.ResponseMetadata = nil
	}
	if response.CreatedAt == 0 {
		response.CreatedAt = originTask.CreatedAt
	}
	if response.UpdatedAt == 0 {
		response.UpdatedAt = originTask.UpdatedAt
	}
	if originTask.Status == model.TaskStatusFailure && response.Error == nil && strings.TrimSpace(originTask.FailReason) != "" {
		response.Error = &SeedanceOfficialResponseError{Message: originTask.FailReason}
	}
	return common.Marshal(response)
}

type (
	SeedanceOfficialResponse struct {
		ID                    string                            `json:"id"`
		Model                 string                            `json:"model"`
		Status                string                            `json:"status"`
		Content               *SeedanceOfficialResponseContent  `json:"content,omitempty"`
		Usage                 *SeedanceOfficialResponseUsage    `json:"usage,omitempty"`
		Error                 *SeedanceOfficialResponseError    `json:"error"`
		ResponseMetadata      *SeedanceOfficialResponseMetadata `json:"ResponseMetadata,omitempty"`
		CreatedAt             int64                             `json:"created_at"`
		UpdatedAt             int64                             `json:"updated_at"`
		Seed                  *int                              `json:"seed,omitempty"`
		Resolution            *string                           `json:"resolution,omitempty"`
		Ratio                 *string                           `json:"ratio,omitempty"`
		Duration              *int                              `json:"duration,omitempty"`
		FramesPerSecond       *int                              `json:"framespersecond,omitempty"`
		GenerateAudio         *bool                             `json:"generate_audio,omitempty"`
		Tools                 json.RawMessage                   `json:"tools,omitempty"`
		SafetyIdentifier      *string                           `json:"safety_identifier,omitempty"`
		Draft                 *bool                             `json:"draft,omitempty"`
		DraftTaskID           *string                           `json:"draft_task_id,omitempty"`
		ExecutionExpiresAfter *int                              `json:"execution_expires_after,omitempty"`
	}
	SeedanceOfficialResponseContent struct {
		VideoURL string `json:"video_url"`
	}
	SeedanceOfficialResponseUsage struct {
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	}
	SeedanceOfficialResponseError struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	SeedanceOfficialResponseMetadata struct {
		Error *SeedanceOfficialResponseError `json:"Error,omitempty"`
	}
)

func parseSeedanceOfficialRequest(c *gin.Context) (*requestPayload, error) {
	var req requestPayload
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return nil, err
	}
	if len(req.Content) > 0 {
		return &req, nil
	}

	var generic relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &generic); err != nil {
		return nil, err
	}
	converted, err := (&TaskAdaptor{}).convertToRequestPayload(&generic)
	if err != nil {
		return nil, err
	}
	return converted, nil
}

func validateSeedanceOfficialRequest(req *requestPayload) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return fmt.Errorf("model is required")
	}
	if len(req.Content) == 0 {
		return fmt.Errorf("content is required")
	}
	if duration, ok := seedanceOfficialDuration(req); ok {
		if duration != -1 && (duration < 0 || duration > relaycommon.MaxTaskDurationSeconds) {
			return fmt.Errorf("duration must be -1 or between 0 and %d", relaycommon.MaxTaskDurationSeconds)
		}
	}
	if req.Frames != nil {
		frames := int(*req.Frames)
		maxFrames := relaycommon.MaxTaskDurationSeconds * 24
		if frames < 0 || frames > maxFrames {
			return fmt.Errorf("frames must be between 0 and %d", maxFrames)
		}
	}
	if resolution := strings.ToLower(strings.TrimSpace(req.Resolution)); resolution != "" {
		switch resolution {
		case "480p", "720p", "1080p", "4k":
		default:
			return fmt.Errorf("resolution must be 480p, 720p, 1080p, or 4k")
		}
		if strings.EqualFold(strings.TrimSpace(req.Model), "doubao-seedance-2-5") && resolution == "4k" {
			return fmt.Errorf("doubao-seedance-2-5 resolution must be 480p, 720p, or 1080p")
		}
	}
	if ratio := strings.ToLower(strings.TrimSpace(req.Ratio)); ratio != "" {
		switch ratio {
		case "16:9", "9:16", "1:1", "4:3", "3:4", "21:9", "9:21", "3:2", "2:3":
		default:
			return fmt.Errorf("ratio is unsupported")
		}
	}
	if len(req.Tools) > 0 {
		switch common.GetJsonType(req.Tools) {
		case "object", "array":
		default:
			return fmt.Errorf("tools must be an object or array")
		}
	}
	return nil
}

func seedanceOfficialBillingRequest(req *requestPayload) relaycommon.TaskSubmitReq {
	metadata := map[string]interface{}{}
	if req.Resolution != "" {
		metadata["resolution"] = req.Resolution
	}
	if req.Ratio != "" {
		metadata["ratio"] = req.Ratio
	}
	duration, _ := seedanceOfficialDuration(req)
	if duration <= 0 {
		duration = 5
	}
	return relaycommon.TaskSubmitReq{
		Prompt:   seedanceOfficialPrompt(req.Content),
		Model:    req.Model,
		Duration: duration,
		Size:     firstNonEmptyString(req.Ratio, req.Resolution),
		Metadata: metadata,
	}
}

func seedanceOfficialDuration(req *requestPayload) (int, bool) {
	if req == nil {
		return 0, false
	}
	if req.Duration != nil {
		return int(*req.Duration), true
	}
	if req.Frames != nil {
		return int(math.Ceil(float64(int(*req.Frames)) / 24.0)), true
	}
	return 0, false
}

func seedanceOfficialPrompt(content []ContentItem) string {
	for _, item := range content {
		if item.Type == "text" && strings.TrimSpace(item.Text) != "" {
			return strings.TrimSpace(item.Text)
		}
	}
	return ""
}

func seedanceOfficialHasVideoInput(req *requestPayload) bool {
	if req == nil {
		return false
	}
	for _, item := range req.Content {
		if item.Type == "video_url" || item.VideoURL != nil {
			return true
		}
	}
	return false
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func TaskModelName(task *model.Task) string {
	if task == nil {
		return ""
	}
	if bc := task.PrivateData.BillingContext; bc != nil && bc.OriginModelName != "" {
		return bc.OriginModelName
	}
	return task.Properties.OriginModelName
}
