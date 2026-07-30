package seedancem

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
	createTaskPath = "/contents/generations/tasks"
	mappingPath    = "/mapping/query"
)

type MediaURL struct {
	URL string `json:"url,omitempty"`
}

type ContentItem struct {
	Type     string    `json:"type,omitempty"`
	Text     string    `json:"text,omitempty"`
	ImageURL *MediaURL `json:"image_url,omitempty"`
	VideoURL *MediaURL `json:"video_url,omitempty"`
	AudioURL *MediaURL `json:"audio_url,omitempty"`
	Role     string    `json:"role,omitempty"`
}

type requestPayload struct {
	Model                 string        `json:"model"`
	Content               []ContentItem `json:"content,omitempty"`
	CallbackURL           string        `json:"callback_url,omitempty"`
	ReturnLastFrame       *bool         `json:"return_last_frame,omitempty"`
	ServiceTier           string        `json:"service_tier,omitempty"`
	ExecutionExpiresAfter *int          `json:"execution_expires_after,omitempty"`
	GenerateAudio         *bool         `json:"generate_audio,omitempty"`
	Draft                 *bool         `json:"draft,omitempty"`
	Tools                 []toolSpec    `json:"tools,omitempty"`
	SafetyIdentifier      string        `json:"safety_identifier,omitempty"`
	Resolution            string        `json:"resolution,omitempty"`
	Ratio                 string        `json:"ratio,omitempty"`
	Duration              *int          `json:"duration,omitempty"`
	Frames                *int          `json:"frames,omitempty"`
	Seed                  *int64        `json:"seed,omitempty"`
	CameraFixed           *bool         `json:"camera_fixed,omitempty"`
	Watermark             *bool         `json:"watermark,omitempty"`
}

type toolSpec struct {
	Type string `json:"type,omitempty"`
}

type submitResponse struct {
	ID string `json:"id"`
}

type responseTask struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Status  string `json:"status"`
	Content struct {
		VideoURL     string `json:"video_url"`
		LastFrameURL string `json:"last_frame_url"`
	} `json:"content"`
	Seed            int64  `json:"seed"`
	Resolution      string `json:"resolution"`
	Duration        int    `json:"duration"`
	Ratio           string `json:"ratio"`
	FramesPerSecond int    `json:"framespersecond"`
	ServiceTier     string `json:"service_tier"`
	Usage           struct {
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
		ToolUsage        struct {
			WebSearch int `json:"web_search"`
		} `json:"tool_usage"`
	} `json:"usage"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

type mappingResponse struct {
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
}

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
	settings    *dto.SeedanceMSettings
	hasVideo    bool
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = baseURL(info.ChannelBaseUrl)
	a.apiKey = strings.TrimSpace(info.ApiKey)
	if info.ChannelOtherSettings.SeedanceM != nil {
		settings := *info.ChannelOtherSettings.SeedanceM
		a.settings = &settings
	}
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	req, err := parseSeedanceMTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if err := validateSeedanceMRequest(req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	info.Action = constant.TaskActionGenerate
	c.Set("task_request", req)
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return a.baseURL + createTaskPath, nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	setBearerHeaders(req, a.apiKey, a.settings)
	if a.hasVideo {
		req.Header.Set("Input-Has-Video", "true")
	}
	if a.settings != nil && a.settings.EnableVideoEncrypt {
		if strings.TrimSpace(a.settings.PublicKeyPEM) == "" || strings.TrimSpace(a.settings.PrivateKeyPEM) == "" {
			return fmt.Errorf("seedance_m public_key_pem and private_key_pem are required when video encryption is enabled")
		}
		req.Header.Set("Enable-TOS-Content-Result-Encryption", "true")
		req.Header.Set("X-Encryption-Algorithm", "RSA_OAEP_4096_AES_256")
		req.Header.Set("PK", base64PublicKey(a.settings.PublicKeyPEM))
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	payload, err := a.convertToRequestPayload(&req)
	if err != nil {
		return nil, errors.Wrap(err, "convert request payload failed")
	}
	modelName := firstNonEmpty(info.UpstreamModelName, info.OriginModelName, req.Model)
	endpoint, err := a.resolveEndpoint(modelName, info.ChannelSetting.Proxy)
	if err != nil {
		return nil, err
	}
	payload.Model = endpoint
	info.UpstreamModelName = endpoint
	a.hasVideo = payloadHasVideo(payload)

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
	billingModel := seedanceMBillingModel(firstNonEmpty(info.OriginModelName, req.Model, info.UpstreamModelName))
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
	return map[string]float64{"seedance_m_estimated_price": ratio}
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var r submitResponse
	if err := common.Unmarshal(responseBody, &r); err != nil {
		return "", nil, service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	if strings.TrimSpace(r.ID) == "" {
		return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
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
	taskID := strings.TrimSpace(asString(body["task_id"]))
	if taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	uri := baseURL(baseUrl) + createTaskPath + "/" + url.PathEscape(taskID)
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	setBearerHeaders(req, strings.TrimSpace(key), a.settings)
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var res responseTask
	if err := common.Unmarshal(respBody, &res); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}
	info := &relaycommon.TaskInfo{
		Code:             0,
		TaskID:           res.ID,
		CompletionTokens: res.Usage.CompletionTokens,
		TotalTokens:      res.Usage.TotalTokens,
	}
	switch strings.ToLower(strings.TrimSpace(res.Status)) {
	case "queued", "pending":
		info.Status = model.TaskStatusQueued
		info.Progress = taskcommon.ProgressQueued
	case "running", "processing":
		info.Status = model.TaskStatusInProgress
		info.Progress = taskcommon.ProgressInProgress
	case "succeeded", "success", "completed":
		info.Status = model.TaskStatusSuccess
		info.Progress = taskcommon.ProgressComplete
		info.Url = res.Content.VideoURL
	case "failed", "cancelled", "canceled", "expired":
		info.Status = model.TaskStatusFailure
		info.Progress = taskcommon.ProgressComplete
		info.Reason = firstNonEmpty(res.Error.Message, res.Error.Code, res.Status)
	default:
		info.Status = model.TaskStatusInProgress
		info.Progress = taskcommon.ProgressInProgress
	}
	return info, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var r responseTask
	if err := common.Unmarshal(originTask.Data, &r); err != nil {
		return nil, errors.Wrap(err, "unmarshal seedance-m task data failed")
	}
	ov := dto.NewOpenAIVideo()
	ov.ID = originTask.TaskID
	ov.TaskID = originTask.TaskID
	ov.Status = originTask.Status.ToVideoStatus()
	ov.SetProgressStr(originTask.Progress)
	ov.CreatedAt = originTask.CreatedAt
	ov.CompletedAt = originTask.FinishTime
	if ov.CompletedAt == 0 {
		ov.CompletedAt = originTask.UpdatedAt
	}
	ov.Model = originTask.Properties.OriginModelName
	ov.SetMetadata("url", firstNonEmpty(r.Content.VideoURL, originTask.GetResultURL()))
	if r.Content.LastFrameURL != "" {
		ov.SetMetadata("last_frame_url", r.Content.LastFrameURL)
	}
	if r.Seed != 0 {
		ov.SetMetadata("seed", r.Seed)
	}
	if r.Resolution != "" {
		ov.SetMetadata("resolution", r.Resolution)
	}
	if r.Duration != 0 {
		ov.SetMetadata("duration", r.Duration)
	}
	if r.Ratio != "" {
		ov.SetMetadata("ratio", r.Ratio)
	}
	if r.FramesPerSecond != 0 {
		ov.SetMetadata("framespersecond", r.FramesPerSecond)
	}
	if originTask.Status == model.TaskStatusFailure {
		ov.Error = &dto.OpenAIVideoError{
			Message: firstNonEmpty(r.Error.Message, originTask.FailReason),
			Code:    firstNonEmpty(r.Error.Code, "task_failed"),
		}
	}
	return common.Marshal(ov)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq) (*requestPayload, error) {
	payload := &requestPayload{
		Model:   req.Model,
		Content: make([]ContentItem, 0, len(req.Images)+2),
	}
	for _, imageURL := range req.Images {
		if strings.TrimSpace(imageURL) == "" {
			continue
		}
		payload.Content = append(payload.Content, ContentItem{
			Type:     "image_url",
			ImageURL: &MediaURL{URL: strings.TrimSpace(imageURL)},
			Role:     "first_frame",
		})
	}
	if err := taskcommon.UnmarshalMetadata(req.Metadata, payload); err != nil {
		return nil, err
	}
	if req.Duration != 0 {
		payload.Duration = &req.Duration
	} else if sec, err := strconv.Atoi(strings.TrimSpace(req.Seconds)); err == nil && sec > 0 {
		payload.Duration = &sec
	}
	if strings.TrimSpace(req.Prompt) != "" && !contentHasText(payload.Content) {
		payload.Content = append([]ContentItem{{Type: "text", Text: req.Prompt}}, payload.Content...)
	}
	if len(payload.Content) == 0 {
		return nil, fmt.Errorf("content is required")
	}
	return payload, nil
}

func (a *TaskAdaptor) resolveEndpoint(modelName, proxy string) (string, error) {
	modelName = firstNonEmpty(modelName, "doubao-seedance-2.0")
	endpoint, err := queryModelMapping(a.baseURL, a.apiKey, modelName, proxy, a.settings)
	if err == nil && strings.TrimSpace(endpoint) != "" {
		return strings.TrimSpace(endpoint), nil
	}
	return modelName, nil
}

func validateSeedanceMRequest(req relaycommon.TaskSubmitReq) error {
	if req.Model != "" && !isSupportedModel(req.Model) {
		return fmt.Errorf("unsupported Seedance model: %s", req.Model)
	}
	if strings.TrimSpace(req.Prompt) == "" && !metadataHasContent(req.Metadata) {
		return fmt.Errorf("prompt or content is required")
	}
	if duration, ok := requestDuration(req); ok {
		if duration != -1 && (duration < 4 || duration > 15) {
			return fmt.Errorf("duration must be -1 or an integer between 4 and 15")
		}
	}
	if resolution := requestResolution(req); resolution != "" {
		switch resolution {
		case "480p", "720p", "1080p":
		default:
			return fmt.Errorf("resolution must be 480p, 720p, or 1080p")
		}
	}
	if expiry, ok := intFromAny(req.Metadata["execution_expires_after"]); ok && (expiry < 3600 || expiry > 259200) {
		return fmt.Errorf("execution_expires_after must be between 3600 and 259200")
	}
	if seed, ok := int64FromAny(req.Metadata["seed"]); ok && (seed < -1 || seed > 4294967295) {
		return fmt.Errorf("seed must be -1 or between 0 and 4294967295")
	}
	if frames, ok := intFromAny(req.Metadata["frames"]); ok && frames < 0 {
		return fmt.Errorf("frames must be non-negative")
	}
	return nil
}

func parseSeedanceMTaskRequest(c *gin.Context) (relaycommon.TaskSubmitReq, error) {
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return req, err
	}
	if len(req.Images) == 0 && strings.TrimSpace(req.Image) != "" {
		req.Images = []string{strings.TrimSpace(req.Image)}
	}
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}

	var raw map[string]interface{}
	if err := common.UnmarshalBodyReusable(c, &raw); err == nil {
		for _, key := range seedanceMOfficialRequestFields {
			if value, ok := raw[key]; ok {
				if _, exists := req.Metadata[key]; !exists {
					req.Metadata[key] = value
				}
			}
		}
		if strings.TrimSpace(req.Prompt) == "" {
			req.Prompt = firstTextFromContent(raw["content"])
		}
	}
	return req, nil
}

var seedanceMOfficialRequestFields = []string{
	"content",
	"callback_url",
	"return_last_frame",
	"service_tier",
	"execution_expires_after",
	"generate_audio",
	"draft",
	"tools",
	"safety_identifier",
	"resolution",
	"ratio",
	"frames",
	"seed",
	"camera_fixed",
	"watermark",
}

func metadataHasContent(metadata map[string]interface{}) bool {
	if len(metadata) == 0 {
		return false
	}
	content, ok := metadata["content"].([]interface{})
	if !ok {
		return false
	}
	return len(content) > 0
}

func firstTextFromContent(value interface{}) string {
	content, ok := value.([]interface{})
	if !ok {
		return ""
	}
	for _, item := range content {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if text, ok := m["text"].(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func queryModelMapping(base, apiKey, modelName, proxy string, settings *dto.SeedanceMSettings) (string, error) {
	body, err := common.Marshal(map[string]string{"model": modelName})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL(base)+mappingPath, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	setBearerHeaders(req, apiKey, settings)
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mapping query status %d", resp.StatusCode)
	}
	var out mappingResponse
	if err := common.DecodeJson(resp.Body, &out); err != nil {
		return "", err
	}
	return firstNonEmpty(out.Endpoint, out.Model), nil
}
