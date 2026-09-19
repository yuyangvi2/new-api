// Package vodaigc implements Tencent Cloud VOD asynchronous AIGC image tasks.
package vodaigc

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	vodHost         = "vod.tencentcloudapi.com"
	vodService      = "vod"
	vodVersion      = "2018-07-17"
	createAction    = "CreateAigcImageTask"
	describeAction  = "DescribeTaskDetail"
	defaultSubAppID = int64(1480226150)

	ctxPayloadKey          = "vod_aigc_payload"
	defaultInputRegion     = "Mainland"
	defaultModelName       = "GG"
	maxBase64Bytes         = 7 * 1024 * 1024
	upstreamModelDelimiter = ":"
)

var defaultModels = []string{
	"gemini-2.5-flash-image",
	"gemini-3-pro-image",
	"gemini-3.1-flash-image",
	"gemini-3.1-flash-lite-image",
	"vidu-q2",
}

var allowedAspectRatios = map[string]map[string]bool{
	"GG:2.5": {
		"1:1": true, "2:3": true, "3:2": true, "3:4": true, "4:3": true,
		"4:5": true, "5:4": true, "9:16": true, "16:9": true, "21:9": true,
	},
	"GG:3.0": {
		"1:1": true, "2:3": true, "3:2": true, "3:4": true, "4:3": true,
		"4:5": true, "5:4": true, "9:16": true, "16:9": true, "21:9": true,
	},
	"GG:3.1": {
		"1:1": true, "1:4": true, "1:8": true, "2:3": true, "3:2": true,
		"3:4": true, "4:1": true, "4:3": true, "4:5": true, "5:4": true,
		"8:1": true, "9:16": true, "16:9": true, "21:9": true,
	},
	"GG:3.1-lite": {
		"1:1": true, "1:4": true, "1:8": true, "2:3": true, "3:2": true,
		"3:4": true, "4:1": true, "4:3": true, "4:5": true, "5:4": true,
		"8:1": true, "9:16": true, "16:9": true, "21:9": true,
	},
	"Vidu:q2": {
		"1:1": true, "2:3": true, "3:2": true, "3:4": true,
		"4:3": true, "9:16": true, "16:9": true, "21:9": true,
	},
}

var allowedResolutions = map[string]map[string]string{
	"GG:2.5":      {"1k": "1K", "2k": "2K", "4k": "4K"},
	"GG:3.0":      {"1k": "1K", "2k": "2K", "4k": "4K"},
	"GG:3.1":      {"720p": "720P", "1k": "1K", "2k": "2K", "4k": "4K"},
	"GG:3.1-lite": {"720p": "720P", "1k": "1K", "2k": "2K", "4k": "4K"},
	"Vidu:q2":     {"1080p": "1080p", "2k": "2K", "4k": "4K"},
}

type inputFileInfo struct {
	Type   string `json:"Type"`
	URL    string `json:"Url,omitempty"`
	Base64 string `json:"Base64,omitempty"`
}

type outputConfig struct {
	StorageMode string `json:"StorageMode"`
	Resolution  string `json:"Resolution,omitempty"`
	AspectRatio string `json:"AspectRatio,omitempty"`
	LogoAdd     string `json:"LogoAdd"`
}

type createRequest struct {
	SubAppID     int64           `json:"SubAppId"`
	ModelName    string          `json:"ModelName"`
	ModelVersion string          `json:"ModelVersion"`
	FileInfos    []inputFileInfo `json:"FileInfos,omitempty"`
	Prompt       string          `json:"Prompt,omitempty"`
	OutputConfig outputConfig    `json:"OutputConfig"`
	InputRegion  string          `json:"InputRegion"`
}

type tcError struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

type submitResponse struct {
	Response struct {
		TaskID    string   `json:"TaskId"`
		RequestID string   `json:"RequestId"`
		Error     *tcError `json:"Error,omitempty"`
	} `json:"Response"`
}

type outputFileInfo struct {
	FileURL string `json:"FileUrl"`
}

type aigcImageTask struct {
	TaskID     string `json:"TaskId"`
	Status     string `json:"Status"`
	ErrCode    int    `json:"ErrCode"`
	ErrCodeExt string `json:"ErrCodeExt"`
	Message    string `json:"Message"`
	Progress   int    `json:"Progress"`
	Output     struct {
		FileInfos []outputFileInfo `json:"FileInfos"`
	} `json:"Output"`
}

type describeResponse struct {
	Response struct {
		Status        string         `json:"Status"`
		ErrCode       int            `json:"ErrCode"`
		Message       string         `json:"Message"`
		AigcImageTask *aigcImageTask `json:"AigcImageTask"`
		Error         *tcError       `json:"Error,omitempty"`
	} `json:"Response"`
}

type TaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey   string
	settings dto.VODAIGCSettings
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.apiKey = info.ApiKey
	a.settings = defaultSettings()
	if info.ChannelOtherSettings.VODAIGC != nil {
		configured := *info.ChannelOtherSettings.VODAIGC
		if configured.SubAppID > 0 {
			a.settings.SubAppID = configured.SubAppID
		}
		if configured.InputRegion != "" {
			a.settings.InputRegion = configured.InputRegion
		}
		if configured.DefaultModelName != "" {
			a.settings.DefaultModelName = configured.DefaultModelName
		}
	}
	if info.TaskRelayInfo != nil {
		info.TaskRelayInfo.SubAppID = a.settings.SubAppID
	}
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	var peek relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &peek); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	mappedModel, err := resolveMappedModel(peek.Model, c.GetString("model_mapping"))
	if err != nil {
		return service.TaskErrorWrapperLocal(
			err, "invalid_model_mapping", http.StatusBadRequest)
	}
	upstreamModel, err := parseUpstreamModel(mappedModel, a.effectiveSettings().DefaultModelName)
	if err != nil {
		return service.TaskErrorWrapperLocal(
			err,
			"invalid_model_mapping", http.StatusBadRequest)
	}

	if taskErr := relaycommon.ValidateImageTaskRequest(c, info, createAction); taskErr != nil {
		return taskErr
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	maxImages := referenceImageLimit(upstreamModel)
	if len(req.Images) > maxImages {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("model %s supports at most %d reference images", peek.Model, maxImages),
			"invalid_image_count", http.StatusBadRequest)
	}
	if _, err := buildInputFileInfos(req.Images); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_reference_image", http.StatusBadRequest)
	}
	if err := validateOutputConfig(upstreamModel, req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_output_config", http.StatusBadRequest)
	}
	return nil
}

type upstreamModel struct {
	Name    string
	Version string
}

func (m upstreamModel) String() string {
	return m.Name + upstreamModelDelimiter + m.Version
}

func defaultSettings() dto.VODAIGCSettings {
	return dto.VODAIGCSettings{
		SubAppID:         defaultSubAppID,
		InputRegion:      defaultInputRegion,
		DefaultModelName: defaultModelName,
	}
}

func (a *TaskAdaptor) effectiveSettings() dto.VODAIGCSettings {
	settings := a.settings
	defaults := defaultSettings()
	if settings.SubAppID <= 0 {
		settings.SubAppID = defaults.SubAppID
	}
	if settings.InputRegion == "" {
		settings.InputRegion = defaults.InputRegion
	}
	if settings.DefaultModelName == "" {
		settings.DefaultModelName = defaults.DefaultModelName
	}
	return settings
}

func resolveMappedModel(modelName string, mappingJSON string) (string, error) {
	return common.ResolveStrictModelMapping(modelName, mappingJSON)
}

func parseUpstreamModel(value string, fallbackModelName string) (upstreamModel, error) {
	name, version, err := constant.ParseVODAIGCUpstreamModel(value, fallbackModelName)
	if err != nil {
		return upstreamModel{}, err
	}
	return upstreamModel{Name: name, Version: version}, nil
}

func referenceImageLimit(model upstreamModel) int {
	limits := map[string]int{
		"GG:2.5": 3, "GG:3.0": 14, "GG:3.1": 14, "GG:3.1-lite": 14,
		"Vidu:q2": 7, "Kling:2.1": 4, "Kling:3.0": 1,
		"Kling:3.0-Omni": 10, "Kling:O1": 10, "Hunyuan:3.0": 3,
	}
	if limit, ok := limits[model.String()]; ok {
		return limit
	}
	return 0
}

func validateOutputConfig(model upstreamModel, req relaycommon.TaskSubmitReq) error {
	aspectRatio := strings.TrimSpace(req.AspectRatio)
	if aspectRatio == "" {
		aspectRatio = strings.TrimSpace(req.Size)
	}
	allowedRatios, validatesRatios := allowedAspectRatios[model.String()]
	if aspectRatio != "" {
		if !validatesRatios || !allowedRatios[aspectRatio] {
			return fmt.Errorf("unsupported aspect_ratio %q for %s", aspectRatio, model.String())
		}
	}

	resolution := strings.ToLower(strings.TrimSpace(req.Resolution))
	if resolution == "" {
		return nil
	}
	allowed, validatesResolution := allowedResolutions[model.String()]
	if !validatesResolution || allowed[resolution] == "" {
		return fmt.Errorf("unsupported resolution %q for %s", req.Resolution, model.String())
	}
	return nil
}

func normalizeResolution(model upstreamModel, value string) string {
	resolution := strings.TrimSpace(value)
	if resolution != "" {
		if allowed := allowedResolutions[model.String()]; allowed != nil {
			if canonical := allowed[strings.ToLower(resolution)]; canonical != "" {
				return canonical
			}
		}
		return resolution
	}
	if model.Name == "GG" && allowedResolutions[model.String()] != nil {
		return "1K"
	}
	if model.String() == "Vidu:q2" {
		return "1080p"
	}
	return ""
}

func endpointURL(baseURL string) (*url.URL, error) {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://" + vodHost
	}
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/")
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid Tencent VOD base URL: %s", baseURL)
	}
	parsed.Host = strings.ToLower(parsed.Host)
	return parsed, nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	endpoint, err := endpointURL(info.ChannelBaseUrl)
	if err != nil {
		return "", err
	}
	return endpoint.String(), nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	settings := a.effectiveSettings()
	upstreamModel, err := parseUpstreamModel(info.UpstreamModelName, settings.DefaultModelName)
	if err != nil {
		return nil, err
	}

	fileInfos, err := buildInputFileInfos(req.Images)
	if err != nil {
		return nil, err
	}

	aspectRatio := strings.TrimSpace(req.AspectRatio)
	if aspectRatio == "" {
		aspectRatio = strings.TrimSpace(req.Size)
	}
	resolution := normalizeResolution(upstreamModel, req.Resolution)
	body := createRequest{
		SubAppID:     settings.SubAppID,
		ModelName:    upstreamModel.Name,
		ModelVersion: upstreamModel.Version,
		FileInfos:    fileInfos,
		Prompt:       req.Prompt,
		OutputConfig: outputConfig{
			StorageMode: "Temporary",
			Resolution:  resolution,
			AspectRatio: aspectRatio,
			LogoAdd:     "Disabled",
		},
		InputRegion: settings.InputRegion,
	}
	payload, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	c.Set(ctxPayloadKey, payload)
	return bytes.NewReader(payload), nil
}

func buildInputFileInfos(images []string) ([]inputFileInfo, error) {
	fileInfos := make([]inputFileInfo, 0, len(images))
	totalBase64Bytes := 0
	for _, image := range images {
		image = strings.TrimSpace(image)
		if image == "" {
			return nil, errors.New("reference image cannot be empty")
		}
		if taskcommon.IsHTTPURL(image) {
			fileInfos = append(fileInfos, inputFileInfo{Type: "Url", URL: image})
			continue
		}

		raw := image
		if comma := strings.Index(raw, ","); strings.HasPrefix(strings.ToLower(raw), "data:image/") && comma >= 0 {
			if !strings.Contains(strings.ToLower(raw[:comma]), ";base64") {
				return nil, errors.New("reference image data URI must use base64 encoding")
			}
			raw = raw[comma+1:]
		}
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			decoded, err = base64.RawStdEncoding.DecodeString(raw)
		}
		if err != nil {
			return nil, fmt.Errorf("reference image must be an http(s) URL or valid base64: %w", err)
		}
		contentType := http.DetectContentType(decoded)
		if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
			return nil, fmt.Errorf("unsupported reference image type: %s", contentType)
		}
		totalBase64Bytes += len(decoded)
		if totalBase64Bytes > maxBase64Bytes {
			return nil, errors.New("base64 reference images exceed the 7 MB Tencent VOD limit")
		}
		fileInfos = append(fileInfos, inputFileInfo{
			Type:   "Base64",
			Base64: base64.StdEncoding.EncodeToString(decoded),
		})
	}
	return fileInfos, nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	payload, _ := c.Get(ctxPayloadKey)
	body, _ := payload.([]byte)
	secretID, secretKey, err := splitKey(a.apiKey)
	if err != nil {
		return err
	}
	for key, value := range signTC3(secretID, secretKey, createAction, req.URL.Host, req.URL.EscapedPath(), body, time.Now().Unix()) {
		req.Header.Set(key, value)
	}
	return nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	if resp == nil || resp.Body == nil {
		return "", nil, service.TaskErrorWrapper(errors.New("empty upstream response"), "empty_response", http.StatusBadGateway)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	var result submitResponse
	if err := common.Unmarshal(responseBody, &result); err != nil {
		return "", nil, service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
	}
	if result.Response.Error != nil {
		return "", nil, taskErrorFromTencentAPI(result.Response.Error)
	}
	if result.Response.TaskID == "" {
		return "", nil, service.TaskErrorWrapper(errors.New("empty TaskId"), "invalid_response", http.StatusBadGateway)
	}
	c.JSON(http.StatusOK, dto.ImageTaskSubmitResponse{
		ID:        info.PublicTaskID,
		TaskID:    info.PublicTaskID,
		Object:    "image_generation_task",
		Model:     info.OriginModelName,
		Status:    "queued",
		CreatedAt: time.Now().Unix(),
	})
	return result.Response.TaskID, responseBody, nil
}

func taskErrorFromTencentAPI(upstreamError *tcError) *dto.TaskError {
	statusCode := http.StatusBadGateway
	localError := false
	code := strings.TrimSpace(upstreamError.Code)

	switch {
	case strings.HasPrefix(code, "InvalidParameter"):
		statusCode = http.StatusBadRequest
		localError = true
	case strings.HasPrefix(code, "AuthFailure"), code == "UnauthorizedOperation":
		statusCode = http.StatusUnauthorized
	case strings.HasPrefix(code, "RequestLimitExceeded"):
		statusCode = http.StatusTooManyRequests
	case code == "ResourceInsufficient":
		statusCode = http.StatusServiceUnavailable
	}

	taskErr := service.TaskErrorWrapper(
		fmt.Errorf("%s: %s", code, upstreamError.Message),
		"task_failed", statusCode,
	)
	taskErr.LocalError = localError
	return taskErr
}

func (a *TaskAdaptor) FetchTask(baseURL string, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, errors.New("invalid task_id")
	}
	secretID, secretKey, err := splitKey(key)
	if err != nil {
		return nil, err
	}
	configuredSubAppID := defaultSubAppID
	if value, ok := body["sub_app_id"].(int64); ok && value > 0 {
		configuredSubAppID = value
	}
	payload, err := common.Marshal(map[string]any{
		"TaskId":   taskID,
		"SubAppId": configuredSubAppID,
	})
	if err != nil {
		return nil, err
	}
	endpoint, err := endpointURL(baseURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	for header, value := range signTC3(secretID, secretKey, describeAction, req.URL.Host, req.URL.EscapedPath(), payload, time.Now().Unix()) {
		req.Header.Set(header, value)
	}
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var result describeResponse
	if err := common.Unmarshal(respBody, &result); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}
	if result.Response.Error != nil {
		return nil, fmt.Errorf("%s: %s", result.Response.Error.Code, result.Response.Error.Message)
	}

	task := result.Response.AigcImageTask
	status := result.Response.Status
	if task != nil && task.Status != "" {
		status = task.Status
	}
	info := &relaycommon.TaskInfo{}
	if task != nil {
		info.TaskID = task.TaskID
		progress := task.Progress
		if progress < 0 {
			progress = 0
		} else if progress > 100 {
			progress = 100
		}
		info.Progress = strconv.Itoa(progress) + "%"
	}

	switch status {
	case "WAITING":
		info.Status = model.TaskStatusSubmitted
	case "PROCESSING":
		info.Status = model.TaskStatusInProgress
	case "FINISH":
		if result.Response.ErrCode != 0 {
			info.Status = model.TaskStatusFailure
			info.Reason = strings.TrimSpace(fmt.Sprintf("%d %s", result.Response.ErrCode, result.Response.Message))
			return info, nil
		}
		if task == nil {
			return nil, errors.New("AigcImageTask missing from finished response")
		}
		if task.ErrCode != 0 || task.ErrCodeExt != "" {
			info.Status = model.TaskStatusFailure
			info.Reason = taskFailureReason(task)
			return info, nil
		}
		for _, fileInfo := range task.Output.FileInfos {
			if fileURL := strings.TrimSpace(fileInfo.FileURL); fileURL != "" {
				info.Url = fileURL
				break
			}
		}
		if info.Url == "" {
			info.Status = model.TaskStatusFailure
			info.Reason = "finished task returned no output image URL"
			return info, nil
		}
		info.Status = model.TaskStatusSuccess
		info.CompletionTokens = 1
		info.TotalTokens = 1
	case "ABORTED":
		info.Status = model.TaskStatusFailure
		if task != nil {
			info.Reason = taskFailureReason(task)
		}
		if info.Reason == "" {
			info.Reason = result.Response.Message
		}
	default:
		return nil, fmt.Errorf("unknown task status: %s", status)
	}
	return info, nil
}

func taskFailureReason(task *aigcImageTask) string {
	parts := make([]string, 0, 3)
	if task.ErrCode != 0 {
		parts = append(parts, strconv.Itoa(task.ErrCode))
	}
	if strings.TrimSpace(task.ErrCodeExt) != "" {
		parts = append(parts, strings.TrimSpace(task.ErrCodeExt))
	}
	if strings.TrimSpace(task.Message) != "" {
		parts = append(parts, strings.TrimSpace(task.Message))
	}
	return strings.Join(parts, " ")
}

func (a *TaskAdaptor) GetModelList() []string {
	models := append([]string(nil), defaultModels...)
	sort.Strings(models)
	return models
}

func (a *TaskAdaptor) GetChannelName() string {
	return "tencent-vod-image"
}

func splitKey(apiKey string) (secretID, secretKey string, err error) {
	parts := strings.Split(apiKey, "|")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", errors.New("invalid api_key, required format is SecretId|SecretKey")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func hmacSHA256(data, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

func signTC3(secretID, secretKey, action string, host string, canonicalURI string, payload []byte, timestamp int64) map[string]string {
	const algorithm = "TC3-HMAC-SHA256"
	const contentType = "application/json; charset=utf-8"

	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-tc-action:%s\n",
		contentType, host, strings.ToLower(action))
	signedHeaders := "content-type;host;x-tc-action"
	canonicalRequest := strings.Join([]string{
		"POST", canonicalURI, "", canonicalHeaders, signedHeaders, sha256Hex(payload),
	}, "\n")

	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, vodService)
	stringToSign := strings.Join([]string{
		algorithm,
		strconv.FormatInt(timestamp, 10),
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	secretDate := hmacSHA256([]byte(date), []byte("TC3"+secretKey))
	secretService := hmacSHA256([]byte(vodService), secretDate)
	secretSigning := hmacSHA256([]byte("tc3_request"), secretService)
	signature := hex.EncodeToString(hmacSHA256([]byte(stringToSign), secretSigning))
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, secretID, credentialScope, signedHeaders, signature)

	return map[string]string{
		"Authorization":  authorization,
		"Content-Type":   contentType,
		"Host":           host,
		"X-TC-Action":    action,
		"X-TC-Timestamp": strconv.FormatInt(timestamp, 10),
		"X-TC-Version":   vodVersion,
	}
}
