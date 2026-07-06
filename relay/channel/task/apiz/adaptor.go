package apiz

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	doubaotask "github.com/QuantumNous/new-api/relay/channel/task/doubao"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
)

const (
	apizBaseURL = "https://api.apiz.ai"
	seedanceID  = "ark/seedance-2.0"

	epCreate = "/api/v3/tasks/create"
	epQuery  = "/api/v3/tasks/query"
)

type submitPayload struct {
	Model   string         `json:"model"`
	Params  map[string]any `json:"params"`
	Channel any            `json:"channel,omitempty"`
}

type submitResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	} `json:"data"`
}

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	var peek relaycommon.TaskSubmitReq
	_ = common.UnmarshalBodyReusable(c, &peek)
	if !isSupportedModel(peek.Model) {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("unsupported apiz model: %s", peek.Model),
			"invalid_model",
			http.StatusBadRequest,
		)
	}
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return baseURL(a.baseURL) + epCreate, nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	setHeaders(req, a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	payload, err := buildSubmitPayload(&req, info.OriginModelName, info.UpstreamModelName)
	if err != nil {
		return nil, err
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	resolution := strings.TrimSpace(asString(req.Metadata["resolution"]))
	if resolution == "" {
		resolution = strings.TrimSpace(req.Size)
	}
	ratio, ok := doubaotask.GetVideoInputRatio(info.OriginModelName, resolution, hasVideoInput(req))
	if !ok || ratio == 1.0 {
		return nil
	}
	return map[string]float64{"video_input": ratio}
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}

	var r submitResponse
	if err = common.Unmarshal(responseBody, &r); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}
	if r.Code != 0 && r.Code != http.StatusOK {
		msg := strings.TrimSpace(r.Message)
		if msg == "" {
			msg = "submit failed"
		}
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("%s", msg), "task_failed", http.StatusBadRequest)
		return
	}
	if r.Data.TaskID == "" {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return r.Data.TaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	payload, err := common.Marshal(map[string]any{"task_id": taskID})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL(baseUrl)+epQuery, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	setHeaders(req, key)
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var envelope map[string]any
	if err := common.Unmarshal(respBody, &envelope); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	if code, hasCode := numericInt(envelope["code"]); hasCode && code != 0 && code != http.StatusOK {
		return relaycommon.FailTaskInfo(firstNonEmpty(asString(envelope["message"]), "query failed")), nil
	}

	data := envelope
	if inner, ok := envelope["data"].(map[string]any); ok {
		data = inner
	}

	status := strings.ToLower(strings.TrimSpace(firstNonEmpty(
		asString(data["status"]),
		asString(data["task_status"]),
		asString(data["state"]),
	)))
	info := &relaycommon.TaskInfo{
		TaskID: asString(data["task_id"]),
	}
	if progress := normalizeProgress(data["progress"]); progress != "" {
		info.Progress = progress
	}
	if total, ok := numericInt(firstNonNil(data["total_tokens"], lookupNested(data, "usage", "total_tokens"))); ok {
		info.TotalTokens = total
	}
	if completion, ok := numericInt(firstNonNil(data["completion_tokens"], lookupNested(data, "usage", "completion_tokens"))); ok {
		info.CompletionTokens = completion
	}

	switch {
	case isSuccessStatus(status):
		info.Status = model.TaskStatusSuccess
		info.Progress = taskcommon.ProgressComplete
		info.Url = extractResultURL(data)
	case isFailureStatus(status):
		info.Status = model.TaskStatusFailure
		info.Progress = taskcommon.ProgressComplete
		info.Reason = firstNonEmpty(
			asString(data["fail_reason"]),
			asString(data["error"]),
			asString(lookupNested(data, "result", "error")),
			asString(lookupNested(data, "output", "error")),
			asString(lookupNested(data, "result", "message")),
			asString(lookupNested(data, "output", "message")),
			asString(data["message"]),
			asString(envelope["message"]),
			"generation failed",
		)
	case status == "" || isInProgressStatus(status):
		info.Status = model.TaskStatusInProgress
		if status == "pending" || status == "queued" {
			info.Status = model.TaskStatusQueued
		}
	default:
		info.Status = model.TaskStatusInProgress
	}
	return info, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func buildSubmitPayload(req *relaycommon.TaskSubmitReq, originModel, upstreamModel string) (*submitPayload, error) {
	params := map[string]any{}
	for k, v := range req.Metadata {
		if k == "model" || k == "content" {
			continue
		}
		params[k] = v
	}

	if req.Prompt != "" {
		params["prompt"] = req.Prompt
	}
	if req.Duration > 0 {
		params["duration"] = req.Duration
	} else if sec, err := strconv.Atoi(strings.TrimSpace(req.Seconds)); err == nil && sec > 0 {
		params["duration"] = sec
	}
	applySize(params, req.Size)
	applyImages(params, collectImages(req))

	outerModel := seedanceID
	if strings.HasPrefix(upstreamModel, "ark/") {
		outerModel = upstreamModel
	}
	params["model"] = variantForModel(firstNonEmpty(originModel, upstreamModel, req.Model))

	return &submitPayload{
		Model:   outerModel,
		Params:  params,
		Channel: nil,
	}, nil
}

func applySize(params map[string]any, size string) {
	size = strings.TrimSpace(size)
	if size == "" {
		return
	}
	if strings.Contains(size, ":") {
		params["ratio"] = size
		return
	}
	if strings.Contains(size, "x") {
		parts := strings.Split(size, "x")
		if len(parts) == 2 {
			w, _ := strconv.Atoi(parts[0])
			h, _ := strconv.Atoi(parts[1])
			if w > 0 && h > 0 {
				params["ratio"] = reduceRatio(w, h)
			}
		}
		return
	}
	params["resolution"] = size
}

func applyImages(params map[string]any, images []string) {
	if len(images) == 0 {
		return
	}
	if len(images) == 1 {
		params["image_url"] = images[0]
		return
	}
	params["reference_images"] = images
}

func collectImages(req *relaycommon.TaskSubmitReq) []string {
	out := make([]string, 0, len(req.Images)+1)
	seen := map[string]struct{}{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	add(req.Image)
	for _, img := range req.Images {
		add(img)
	}
	return out
}

func hasVideoInput(req relaycommon.TaskSubmitReq) bool {
	if strings.TrimSpace(req.InputReference) != "" {
		return true
	}
	content, ok := req.Metadata["content"].([]interface{})
	if !ok {
		return false
	}
	for _, item := range content {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if itemMap["type"] == "video_url" {
			return true
		}
		if _, ok := itemMap["video_url"]; ok {
			return true
		}
	}
	return false
}

func isSupportedModel(modelName string) bool {
	if modelName == "" {
		return true
	}
	for _, m := range ModelList {
		if modelName == m {
			return true
		}
	}
	return false
}

func variantForModel(modelName string) string {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case "seedance2.0_direct", "seedance2.0_vision", "doubao-seedance-2-0-260128":
		return "seedance_2.0"
	case "seedance2.0_fast_direct", "seedance2.0_fast_vision", "doubao-seedance-2-0-fast-260128", seedanceID:
		return "seedance_2.0_fast"
	case "seedance2.0_mini":
		return "seedance_2.0"
	case "seedance2.0_fast_mini":
		return "seedance_2.0_fast"
	default:
		if strings.Contains(strings.ToLower(modelName), "fast") {
			return "seedance_2.0_fast"
		}
		return "seedance_2.0"
	}
}

func baseURL(raw string) string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		base = apizBaseURL
	}
	for _, suffix := range []string{epCreate, epQuery, "/api/v3/tasks", "/api/v3", "/api"} {
		base = strings.TrimSuffix(base, suffix)
	}
	return base
}

func setHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

func isSuccessStatus(status string) bool {
	switch status {
	case "success", "succeeded", "completed", "complete", "done", "finished":
		return true
	}
	return false
}

func isFailureStatus(status string) bool {
	switch status {
	case "failed", "failure", "error", "cancelled", "canceled", "rejected":
		return true
	}
	return false
}

func isInProgressStatus(status string) bool {
	switch status {
	case "pending", "queued", "queueing", "processing", "running", "in_progress", "inprogress", "generating", "submitted", "waiting":
		return true
	}
	return false
}

func normalizeProgress(v any) string {
	switch p := v.(type) {
	case string:
		p = strings.TrimSpace(p)
		if p == "" {
			return ""
		}
		if strings.HasSuffix(p, "%") {
			return p
		}
		return p + "%"
	case float64:
		if p < 0 {
			return ""
		}
		return fmt.Sprintf("%.0f%%", p)
	case int:
		if p < 0 {
			return ""
		}
		return fmt.Sprintf("%d%%", p)
	}
	return ""
}

func extractResultURL(data map[string]any) string {
	for _, key := range []string{"video_url", "videoUrl", "result_url", "resultUrl", "url", "download_url", "downloadUrl", "output_url", "outputUrl"} {
		if u := urlFromAny(data[key], true); u != "" {
			return u
		}
	}
	for _, key := range []string{"result", "output", "outputs", "results", "data", "response", "file", "files", "video", "videos"} {
		if u := urlFromAny(data[key], true); u != "" {
			return u
		}
	}
	if u := urlFromAny(data, false); u != "" {
		return u
	}
	return ""
}

func urlFromAny(v any, preferVideo bool) string {
	var fallback string
	var walk func(any)
	walk = func(x any) {
		if fallback != "" && (!preferVideo || isVideoURL(fallback)) {
			return
		}
		switch t := x.(type) {
		case string:
			if looksLikeURL(t) {
				if preferVideo && isVideoURL(t) {
					fallback = t
					return
				}
				if fallback == "" {
					fallback = t
				}
			}
		case []any:
			for _, item := range t {
				walk(item)
			}
		case map[string]any:
			for _, item := range t {
				walk(item)
			}
		}
	}
	walk(v)
	return fallback
}

func looksLikeURL(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func isVideoURL(s string) bool {
	lower := strings.ToLower(strings.Split(strings.TrimSpace(s), "?")[0])
	return strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".mov") || strings.HasSuffix(lower, ".webm") || strings.HasSuffix(lower, ".m3u8")
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func lookupNested(m map[string]any, keys ...string) any {
	var current any = m
	for _, key := range keys {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = obj[key]
	}
	return current
}

func numericInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case float64:
		return int(t), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(t))
		return i, err == nil
	}
	return 0, false
}

func reduceRatio(w, h int) string {
	g := gcd(w, h)
	return fmt.Sprintf("%d:%d", w/g, h/g)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	if a == 0 {
		return 1
	}
	return a
}
