package toapis

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	toapisBaseURL = "https://toapis.com"
	epVideos      = "/v1/videos/generations"
)

type submitResponse struct {
	ID       string `json:"id"`
	Object   string `json:"object"`
	Model    string `json:"model"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Error    struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type roleMedia struct {
	URL  string `json:"url"`
	Role string `json:"role"`
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
			fmt.Errorf("unsupported ToAPIs model: %s", peek.Model),
			"invalid_model",
			http.StatusBadRequest,
		)
	}
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return baseURL(a.baseURL) + epVideos, nil
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
	payload, videoSR, err := buildSubmitPayload(&req, info.OriginModelName, info.UpstreamModelName, common.VideoSuperResolutionEnabled)
	if err != nil {
		return nil, err
	}
	if videoSR != nil && info.TaskRelayInfo != nil {
		info.TaskRelayInfo.VideoSuperResolution = videoSR
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
	billingModel, ok := seedanceBillingModel(firstNonEmpty(info.OriginModelName, req.Model, info.UpstreamModelName))
	if !ok {
		return nil
	}
	req = normalizedBillingRequest(req)
	quota, _, _, ok := doubaotask.EstimateSeedanceQuotaForRequest(
		billingModel,
		req,
		hasVideoInput(req),
		info.PriceData.GroupRatioInfo.GroupRatio,
	)
	if !ok || info.PriceData.Quota <= 0 {
		return nil
	}
	ratio := float64(quota) / float64(info.PriceData.Quota)
	if ratio == 1.0 {
		return nil
	}
	return map[string]float64{"seedance_estimated_price": ratio}
}

func normalizedBillingRequest(req relaycommon.TaskSubmitReq) relaycommon.TaskSubmitReq {
	if strings.TrimSpace(req.Size) == "" {
		req.Size = firstNonEmpty(asString(req.Metadata["aspect_ratio"]), asString(req.Metadata["ratio"]))
	}
	return req
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
	if r.Error.Message != "" {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("%s", r.Error.Message), firstNonEmpty(r.Error.Code, "task_failed"), http.StatusBadRequest)
		return
	}
	if r.ID == "" {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return r.ID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	uri := baseURL(baseUrl) + epVideos + "/" + url.PathEscape(strings.TrimSpace(taskID))
	req, err := http.NewRequest(http.MethodGet, uri, nil)
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
	var data map[string]any
	if err := common.Unmarshal(respBody, &data); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}
	if errMap, ok := data["error"].(map[string]any); ok {
		message := strings.TrimSpace(asString(errMap["message"]))
		if message != "" {
			return relaycommon.FailTaskInfo(message), nil
		}
	}

	status := strings.ToLower(strings.TrimSpace(asString(data["status"])))
	info := &relaycommon.TaskInfo{
		TaskID: firstNonEmpty(asString(data["id"]), asString(data["task_id"])),
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
			asString(lookupNested(data, "error", "message")),
			asString(data["message"]),
			"generation failed",
		)
	case status == "queued":
		info.Status = model.TaskStatusQueued
	case status == "" || isInProgressStatus(status):
		info.Status = model.TaskStatusInProgress
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

func buildSubmitPayload(req *relaycommon.TaskSubmitReq, originModel, upstreamModel string, videoSREnabled bool) (map[string]any, *relaycommon.TaskVideoSuperResolution, error) {
	modelName := toapisModel(firstNonEmpty(upstreamModel, originModel, req.Model))
	payload := map[string]any{}
	for k, v := range req.Metadata {
		if isToAPISPassthroughKey(k) {
			payload[k] = v
		}
	}
	payload["model"] = modelName
	if req.Prompt != "" {
		payload["prompt"] = req.Prompt
	}
	if req.Duration > 0 {
		payload["duration"] = req.Duration
	} else if sec, err := strconv.Atoi(strings.TrimSpace(req.Seconds)); err == nil && sec > 0 {
		payload["duration"] = sec
	}
	applySize(payload, req.Size)
	applyMetadataAliases(payload, req.Metadata)
	videoSR := applyVideoSuperResolutionRewrite(req, payload, videoSREnabled)

	if _, has := payload["video_with_roles"]; !has {
		videos, err := roleMediaFromURLList(req.Metadata, "reference_video", 3, "video_with_roles",
			"referenceVideoUrls", "reference_video_urls", "video_files")
		if err != nil {
			return nil, nil, err
		}
		if len(videos) > 0 {
			payload["video_with_roles"] = videos
		}
	}
	if _, has := payload["audio_with_roles"]; !has {
		audios, err := roleMediaFromURLList(req.Metadata, "reference_audio", 3, "audio_with_roles",
			"referenceAudioUrls", "reference_audio_urls", "audio_files")
		if err != nil {
			return nil, nil, err
		}
		if len(audios) > 0 {
			payload["audio_with_roles"] = audios
		}
	}
	if _, hasRoles := payload["image_with_roles"]; !hasRoles && payload["image_urls"] == nil {
		images, err := buildImageRoles(req, payload, modelName)
		if err != nil {
			return nil, nil, err
		}
		if len(images) > 0 {
			payload["image_with_roles"] = images
		}
	}
	if modelName == "seedance-2-mini" {
		delete(payload, "generate_audio")
	}
	if err := validateRoleMediaPayload(payload, modelName); err != nil {
		return nil, nil, err
	}
	return payload, videoSR, nil
}

func applyMetadataAliases(payload map[string]any, metadata map[string]any) {
	if _, has := payload["aspect_ratio"]; !has {
		if ratio := strings.TrimSpace(asString(metadata["ratio"])); ratio != "" {
			payload["aspect_ratio"] = ratio
		}
	}
	if _, has := payload["client_business_id"]; !has {
		if id := strings.TrimSpace(asString(lookupNested(metadata, "metadata", "client_business_id"))); id != "" {
			payload["client_business_id"] = id
		}
	}
}

func applySize(payload map[string]any, size string) {
	size = strings.TrimSpace(size)
	if size == "" {
		return
	}
	if strings.Contains(size, ":") {
		payload["aspect_ratio"] = size
		return
	}
	if strings.Contains(size, "x") {
		parts := strings.Split(size, "x")
		if len(parts) == 2 {
			w, _ := strconv.Atoi(parts[0])
			h, _ := strconv.Atoi(parts[1])
			if w > 0 && h > 0 {
				payload["aspect_ratio"] = reduceRatio(w, h)
			}
		}
		return
	}
	payload["resolution"] = size
}

func buildImageRoles(req *relaycommon.TaskSubmitReq, payload map[string]any, modelName string) ([]roleMedia, error) {
	referenceImages, err := referenceImages(req)
	if err != nil {
		return nil, err
	}
	images := make([]roleMedia, 0, len(referenceImages)+1)
	hasReferenceMode := len(referenceImages) > 0 || hasRoleMedia(payload, "video_with_roles") || hasRoleMedia(payload, "audio_with_roles")
	if img := strings.TrimSpace(req.Image); img != "" {
		role := "first_frame"
		if modelName == "seedance-2-mini" || hasReferenceMode {
			role = "reference_image"
		}
		if err := validateMediaURL(img, "image_with_roles", 1, true, true); err != nil {
			return nil, err
		}
		images = append(images, roleMedia{URL: img, Role: role})
	}
	for i, img := range referenceImages {
		if err := validateMediaURL(img, "image_with_roles", i+1, true, true); err != nil {
			return nil, err
		}
		images = append(images, roleMedia{URL: img, Role: "reference_image"})
	}
	if len(images) > 9 {
		return nil, fmt.Errorf("image_with_roles supports up to 9 reference images")
	}
	return images, nil
}

func referenceImages(req *relaycommon.TaskSubmitReq) ([]string, error) {
	if len(req.Images) > 0 {
		return normalizeStringList(req.Images), nil
	}
	for _, key := range []string{"referenceImages", "reference_images", "image_urls"} {
		if values, err := stringListFromAny(req.Metadata[key]); err == nil {
			return values, nil
		}
	}
	return nil, nil
}

func roleMediaFromURLList(metadata map[string]any, role string, limit int, field string, keys ...string) ([]roleMedia, error) {
	var values []string
	for _, key := range keys {
		next, err := stringListFromAny(metadata[key])
		if err != nil || len(next) == 0 {
			continue
		}
		values = next
		break
	}
	if len(values) == 0 {
		return nil, nil
	}
	if len(values) > limit {
		return nil, fmt.Errorf("%s supports up to %d URLs", field, limit)
	}
	items := make([]roleMedia, 0, len(values))
	allowData := role == "reference_audio"
	for i, value := range values {
		if err := validateMediaURL(value, field, i+1, false, allowData); err != nil {
			return nil, err
		}
		items = append(items, roleMedia{URL: value, Role: role})
	}
	return items, nil
}

func validateRoleMediaPayload(payload map[string]any, modelName string) error {
	imageRoles := countRoleMedia(payload["image_with_roles"])
	videoRoles := countRoleMedia(payload["video_with_roles"])
	audioRoles := countRoleMedia(payload["audio_with_roles"])
	if imageRoles["first_frame"] > 1 {
		return fmt.Errorf("image_with_roles supports at most one first_frame")
	}
	if imageRoles["last_frame"] > 1 {
		return fmt.Errorf("image_with_roles supports at most one last_frame")
	}
	if imageRoles["reference_image"] > 9 {
		return fmt.Errorf("image_with_roles supports up to 9 reference_image items")
	}
	if videoRoles["reference_video"] > 3 {
		return fmt.Errorf("video_with_roles supports up to 3 reference_video items")
	}
	if audioRoles["reference_audio"] > 3 {
		return fmt.Errorf("audio_with_roles supports up to 3 reference_audio items")
	}
	if audioRoles["reference_audio"] > 0 &&
		imageRoles["first_frame"]+imageRoles["last_frame"]+imageRoles["reference_image"] == 0 &&
		lenStringList(payload["image_urls"]) == 0 &&
		videoRoles["reference_video"] == 0 {
		return fmt.Errorf("audio_with_roles cannot be used alone; provide image_with_roles or video_with_roles")
	}
	if imageRoles["reference_image"] > 0 && (imageRoles["first_frame"] > 0 || imageRoles["last_frame"] > 0) {
		return fmt.Errorf("first_frame/last_frame mode cannot be mixed with reference_image mode")
	}
	if modelName == "seedance-2-mini" && (imageRoles["first_frame"] > 0 || imageRoles["last_frame"] > 0) {
		return fmt.Errorf("seedance-2-mini only supports reference_image")
	}
	if (imageRoles["first_frame"] > 0 || imageRoles["last_frame"] > 0) &&
		(videoRoles["reference_video"] > 0 || audioRoles["reference_audio"] > 0) {
		return fmt.Errorf("first_frame/last_frame mode cannot be mixed with reference video or audio")
	}
	if lenStringList(payload["image_urls"]) > 0 && imageRoles["first_frame"]+imageRoles["last_frame"]+imageRoles["reference_image"] > 0 {
		return fmt.Errorf("image_urls cannot be used with image_with_roles")
	}
	return nil
}

func hasRoleMedia(payload map[string]any, key string) bool {
	roles := countRoleMedia(payload[key])
	for _, count := range roles {
		if count > 0 {
			return true
		}
	}
	return false
}

func lenStringList(value any) int {
	values, err := stringListFromAny(value)
	if err != nil {
		return 0
	}
	return len(values)
}

func countRoleMedia(value any) map[string]int {
	counts := map[string]int{}
	switch items := value.(type) {
	case []roleMedia:
		for _, item := range items {
			counts[item.Role]++
		}
	case []map[string]string:
		for _, item := range items {
			counts[item["role"]]++
		}
	case []any:
		for _, raw := range items {
			if item, ok := raw.(map[string]any); ok {
				counts[asString(item["role"])]++
			}
		}
	}
	return counts
}

func stringListFromAny(v any) ([]string, error) {
	switch values := v.(type) {
	case nil:
		return nil, fmt.Errorf("not an array")
	case []string:
		return normalizeStringList(values), nil
	case []any:
		out := make([]string, 0, len(values))
		for _, item := range values {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("non-string item")
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("not an array")
	}
}

func normalizeStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func validateMediaURL(value, field string, index int, allowImageData bool, allowAudioData bool) error {
	value = strings.TrimSpace(value)
	if taskcommon.IsHTTPURL(value) || strings.HasPrefix(strings.ToLower(value), "asset://") {
		return nil
	}
	lower := strings.ToLower(value)
	if allowImageData && strings.HasPrefix(lower, "data:image/") {
		return nil
	}
	if allowAudioData && strings.HasPrefix(lower, "data:audio/") {
		return nil
	}
	return fmt.Errorf("%s %d must be an HTTP/HTTPS URL, data URI, or asset:// URI", field, index)
}

func hasVideoInput(req relaycommon.TaskSubmitReq) bool {
	if strings.TrimSpace(req.InputReference) != "" {
		return true
	}
	if countRoleMedia(req.Metadata["video_with_roles"])["reference_video"] > 0 {
		return true
	}
	for _, key := range []string{"referenceVideoUrls", "reference_video_urls", "video_files"} {
		if values, err := stringListFromAny(req.Metadata[key]); err == nil && len(values) > 0 {
			return true
		}
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
	if strings.TrimSpace(modelName) == "" {
		return true
	}
	_, ok := toapisModelName(modelName)
	return ok
}

func toapisModel(modelName string) string {
	if model, ok := toapisModelName(modelName); ok {
		return model
	}
	return strings.TrimSpace(modelName)
}

func toapisModelName(modelName string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case "seedance-2", "seedance", "seedance2.0_direct", "seedance2.0_vision", "seedance_2.0", "doubao-seedance-2-0-260128", "dreamina-seedance-2-0-260128":
		return "seedance-2", true
	case "seedance-2-fast", "seedance-fast", "seedance2.0_fast_direct", "seedance2.0_fast_vision", "seedance_2.0_fast", "ark/seedance-2.0", "doubao-seedance-2-0-fast-260128", "dreamina-seedance-2-0-fast-260128":
		return "seedance-2-fast", true
	case "seedance-2-mini", "seedance_2.0_mini", "seedance2.0_mini":
		return "seedance-2-mini", true
	default:
		return "", false
	}
}

func seedanceBillingModel(modelName string) (string, bool) {
	switch toapisModel(modelName) {
	case "seedance-2":
		return "seedance2.0_direct", true
	case "seedance-2-fast":
		return "seedance2.0_fast_direct", true
	case "seedance-2-mini":
		return "Seedance_2.0_mini", true
	default:
		if doubaotask.IsSeedanceModel(modelName) {
			return modelName, true
		}
		return "", false
	}
}

func isToAPISPassthroughKey(key string) bool {
	switch key {
	case "client_business_id", "duration", "aspect_ratio", "image_urls", "image_with_roles",
		"video_with_roles", "audio_with_roles", "resolution", "generate_audio", "seed",
		"callback_url", "trace_id":
		return true
	default:
		return false
	}
}

func baseURL(raw string) string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		base = toapisBaseURL
	}
	for _, suffix := range []string{epVideos, "/v1/videos", "/v1"} {
		base = strings.TrimSuffix(base, suffix)
	}
	return base
}

func setHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

func reduceRatio(width, height int) string {
	g := gcd(width, height)
	if g <= 0 {
		return "16:9"
	}
	return fmt.Sprintf("%d:%d", width/g, height/g)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func lookupNested(data map[string]any, keys ...string) any {
	var current any = data
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}
	return current
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return ""
	}
}

func numericInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func normalizeProgress(value any) string {
	if value == nil {
		return ""
	}
	if s := strings.TrimSpace(asString(value)); s != "" {
		if strings.HasSuffix(s, "%") {
			return s
		}
		if _, err := strconv.ParseFloat(s, 64); err == nil {
			return s + "%"
		}
	}
	return ""
}

func extractResultURL(data map[string]any) string {
	if u := asString(data["video_url"]); u != "" {
		return u
	}
	result, _ := data["result"].(map[string]any)
	if u := asString(result["url"]); u != "" {
		return u
	}
	if items, ok := result["data"].([]any); ok {
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if u := asString(item["url"]); u != "" {
				return u
			}
		}
	}
	output, _ := data["output"].(map[string]any)
	if u := asString(output["video_url"]); u != "" {
		return u
	}
	return asString(output["url"])
}

func isSuccessStatus(status string) bool {
	switch status {
	case "completed", "success", "succeeded", "complete", "done", "finished":
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
	case "queued", "in_progress", "inprogress", "pending", "processing", "running", "generating", "submitted", "waiting":
		return true
	}
	return false
}
