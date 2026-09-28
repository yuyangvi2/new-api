package vodaigc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	synchronousPollInterval = 2 * time.Second
	synchronousPollAttempts = 180
)

var synchronousImageModels = []string{
	"gpt-image-2",
	"gpt-image-2.5-sunburst",
	"gpt-image-2.5-flare",
}

// ImageAdaptor exposes Tencent VOD's asynchronous OG image models through the
// synchronous OpenAI Images API. OpenAI's adaptor is embedded only to satisfy
// non-image interface methods; channel path filtering prevents those methods
// from being selected for this channel type.
type ImageAdaptor struct {
	openai.Adaptor
	apiKey          string
	settings        dto.VODAIGCSettings
	responseFormat  string
	requestQuality  string
	requestSize     string
	acceptedCharged bool
}

func (a *ImageAdaptor) Init(info *relaycommon.RelayInfo) {
	a.apiKey = info.ApiKey
	a.settings = defaultSettings()
	if info.ChannelOtherSettings.VODAIGC == nil {
		return
	}
	configured := *info.ChannelOtherSettings.VODAIGC
	if configured.SubAppID > 0 {
		a.settings.SubAppID = configured.SubAppID
	}
	if configured.InputRegion != "" {
		a.settings.InputRegion = configured.InputRegion
	}
}

func (a *ImageAdaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	endpoint, err := endpointURL(info.ChannelBaseUrl)
	if err != nil {
		return "", err
	}
	return endpoint.String(), nil
}

func (a *ImageAdaptor) SetupRequestHeader(c *gin.Context, header *http.Header, info *relaycommon.RelayInfo) error {
	payloadValue, ok := c.Get(ctxPayloadKey)
	if !ok {
		return errors.New("Tencent VOD request payload is missing")
	}
	payload, ok := payloadValue.([]byte)
	if !ok {
		return errors.New("Tencent VOD request payload is invalid")
	}
	secretID, secretKey, err := splitKey(a.apiKey)
	if err != nil {
		return err
	}
	endpoint, err := endpointURL(info.ChannelBaseUrl)
	if err != nil {
		return err
	}
	for key, value := range signTC3(secretID, secretKey, createAction, endpoint.Host, endpoint.EscapedPath(), payload, time.Now().Unix()) {
		header.Set(key, value)
	}
	return nil
}

func (a *ImageAdaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	if request.N != nil && *request.N != 1 {
		return nil, errors.New("Tencent VOD OG supports exactly one output image per request")
	}
	if request.Stream != nil && *request.Stream {
		return nil, errors.New("Tencent VOD OG does not support streaming image responses")
	}

	responseFormat := strings.ToLower(strings.TrimSpace(request.ResponseFormat))
	if responseFormat == "" {
		responseFormat = "b64_json"
	}
	if responseFormat != "url" && responseFormat != "b64_json" {
		return nil, fmt.Errorf("unsupported response_format %q; use url or b64_json", request.ResponseFormat)
	}

	modelName := strings.TrimSpace(info.OriginModelName)
	if modelName == "" {
		modelName = strings.TrimSpace(request.Model)
	}
	modelVersion, normalizedQuality, err := ogModelVersion(modelName, request.Quality)
	if err != nil {
		return nil, err
	}
	extInfo, err := buildOGExtInfo(request.Size, request.Background)
	if err != nil {
		return nil, err
	}
	if err := validateOpenAIImageCompatibility(request); err != nil {
		return nil, err
	}

	fileInfos, err := synchronousInputFileInfos(c, request)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.Prompt) == "" && len(fileInfos) == 0 {
		return nil, errors.New("prompt is required when no reference image is provided")
	}
	if info.RelayMode == relayconstant.RelayModeImagesEdits && len(fileInfos) == 0 {
		return nil, errors.New("image is required for image edits")
	}
	if info.RelayMode != relayconstant.RelayModeImagesEdits && len(fileInfos) > 0 {
		return nil, errors.New("reference images are only supported by the image edits endpoint")
	}

	negativePrompt, enhancePrompt, seed, tasksPriority, err := parseDocumentedExtraParameters(request.Extra)
	if err != nil {
		return nil, err
	}
	logoAdd := "Disabled"
	if request.Watermark != nil && *request.Watermark {
		logoAdd = "Enabled"
	}
	body := createRequest{
		SubAppID:       a.settings.SubAppID,
		ModelName:      "OG",
		ModelVersion:   modelVersion,
		FileInfos:      fileInfos,
		Prompt:         request.Prompt,
		NegativePrompt: negativePrompt,
		EnhancePrompt:  enhancePrompt,
		OutputConfig: outputConfig{
			StorageMode: "Temporary",
			LogoAdd:     logoAdd,
		},
		InputRegion:   a.settings.InputRegion,
		Seed:          seed,
		TasksPriority: tasksPriority,
		ExtInfo:       extInfo,
	}
	payload, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	c.Set(ctxPayloadKey, payload)
	a.responseFormat = responseFormat
	a.requestQuality = normalizedQuality
	a.requestSize = request.Size
	return bytes.NewBuffer(payload), nil
}

func (a *ImageAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *ImageAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (any, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(errors.New("empty Tencent VOD response"), types.ErrorCodeEmptyResponse, http.StatusBadGateway)
	}
	responseBody, err := io.ReadAll(resp.Body)
	service.CloseResponseBodyGracefully(resp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusBadGateway)
	}

	var submitted submitResponse
	if err := common.Unmarshal(responseBody, &submitted); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusBadGateway)
	}
	if submitted.Response.Error != nil {
		statusCode := taskErrorFromTencentAPI(submitted.Response.Error).StatusCode
		return nil, types.NewOpenAIError(
			fmt.Errorf("%s: %s", submitted.Response.Error.Code, submitted.Response.Error.Message),
			types.ErrorCodeBadResponse,
			statusCode,
		)
	}
	taskID := strings.TrimSpace(submitted.Response.TaskID)
	if taskID == "" {
		return nil, types.NewOpenAIError(errors.New("Tencent VOD returned an empty TaskId"), types.ErrorCodeBadResponseBody, http.StatusBadGateway)
	}

	taskAdaptor := &TaskAdaptor{apiKey: a.apiKey, settings: a.settings}
	var lastPollErr error
	for attempt := 0; attempt < synchronousPollAttempts; attempt++ {
		if attempt > 0 && !info.ChannelOtherSettings.DisableTaskPollingSleep {
			time.Sleep(synchronousPollInterval)
		}
		pollResp, fetchErr := taskAdaptor.FetchTask(info.ChannelBaseUrl, a.apiKey, map[string]any{
			"task_id":    taskID,
			"sub_app_id": a.settings.SubAppID,
		}, info.ChannelSetting.Proxy)
		if fetchErr != nil {
			lastPollErr = fetchErr
			continue
		}
		pollBody, readErr := io.ReadAll(io.LimitReader(pollResp.Body, 2*1024*1024))
		service.CloseResponseBodyGracefully(pollResp)
		if readErr != nil {
			lastPollErr = readErr
			continue
		}
		if pollResp.StatusCode < http.StatusOK || pollResp.StatusCode >= http.StatusMultipleChoices {
			lastPollErr = fmt.Errorf("Tencent VOD polling returned HTTP %d", pollResp.StatusCode)
			continue
		}
		taskInfo, parseErr := taskAdaptor.ParseTaskResult(pollBody)
		if parseErr != nil {
			lastPollErr = parseErr
			continue
		}
		switch taskInfo.Status {
		case model.TaskStatusSubmitted, model.TaskStatusInProgress:
			continue
		case model.TaskStatusFailure:
			return nil, types.NewOpenAIError(
				fmt.Errorf("Tencent VOD image task failed: %s", taskInfo.Reason),
				types.ErrorCodeBadResponse,
				http.StatusBadGateway,
				types.ErrOptionWithSkipRetry(),
			)
		case model.TaskStatusSuccess:
			imageData := dto.ImageData{}
			if a.responseFormat == "url" {
				imageData.Url = taskInfo.Url
			} else {
				base64Data, _, downloadErr := service.GetBase64Data(c, types.NewURLFileSource(taskInfo.Url), "fetching Tencent VOD generated image")
				if downloadErr != nil {
					return nil, a.acceptedTaskError(c, info, fmt.Errorf("download Tencent VOD generated image: %w", downloadErr))
				}
				imageData.B64Json = base64Data
			}
			result := dto.ImageResponse{
				Created: time.Now().Unix(),
				Data:    []dto.ImageData{imageData},
			}
			resultBody, marshalErr := common.Marshal(result)
			if marshalErr != nil {
				return nil, a.acceptedTaskError(c, info, marshalErr)
			}
			resp.StatusCode = http.StatusOK
			resp.Header.Set("Content-Type", "application/json")
			service.IOCopyBytesGracefully(c, resp, resultBody)
			return a.billingUsage(info), nil
		default:
			lastPollErr = fmt.Errorf("Tencent VOD returned unknown task status %q", taskInfo.Status)
		}
	}
	if lastPollErr == nil {
		lastPollErr = errors.New("Tencent VOD image task polling timed out")
	} else {
		lastPollErr = fmt.Errorf("Tencent VOD image task polling timed out: %w", lastPollErr)
	}
	return nil, a.acceptedTaskError(c, info, lastPollErr)
}

func (a *ImageAdaptor) acceptedTaskError(c *gin.Context, info *relaycommon.RelayInfo, err error) *types.NewAPIError {
	if !a.acceptedCharged {
		a.acceptedCharged = true
		service.PostTextConsumeQuota(c, info, a.billingUsage(info), []string{
			"腾讯 VOD 任务已提交但同步响应失败",
		})
	}
	return types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusGatewayTimeout, types.ErrOptionWithSkipRetry())
}

func (a *ImageAdaptor) billingUsage(info *relaycommon.RelayInfo) *dto.Usage {
	if info.TieredBillingSnapshot != nil {
		promptTokens := info.TieredBillingSnapshot.EstimatedPromptTokens
		completionTokens := info.TieredBillingSnapshot.EstimatedCompletionTokens
		return &dto.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
			UsageSource:      "tencent_vod_estimate",
		}
	}
	if info.PriceData.UsePrice {
		return &dto.Usage{PromptTokens: 1, TotalTokens: 1, UsageSource: "tencent_vod_estimate"}
	}
	promptTokens := info.GetEstimatePromptTokens()
	if promptTokens < common.PreConsumedQuota {
		promptTokens = common.PreConsumedQuota
	}
	promptTokens += (&dto.ImageRequest{}).GetTokenCountMeta().MaxTokens
	return &dto.Usage{
		PromptTokens: promptTokens,
		TotalTokens:  promptTokens,
		UsageSource:  "tencent_vod_estimate",
	}
}

func (a *ImageAdaptor) GetModelList() []string {
	return append([]string(nil), synchronousImageModels...)
}

func (a *ImageAdaptor) GetChannelName() string {
	return "tencent-vod-openai-image"
}

func ogModelVersion(modelName string, quality string) (string, string, error) {
	model := strings.ToLower(strings.TrimSpace(modelName))
	normalizedQuality := strings.ToLower(strings.TrimSpace(quality))
	if normalizedQuality == "" || normalizedQuality == "auto" || normalizedQuality == "standard" {
		normalizedQuality = "medium"
	}

	switch model {
	case "gpt-image-2":
		if normalizedQuality != "low" && normalizedQuality != "medium" && normalizedQuality != "high" {
			return "", "", fmt.Errorf("quality %q is not supported by gpt-image-2; use low, medium, or high", quality)
		}
		return "image2_" + normalizedQuality, normalizedQuality, nil
	case "gpt-image-2.5-sunburst":
		if !isImage25Quality(normalizedQuality) {
			return "", "", fmt.Errorf("quality %q is not supported by gpt-image-2.5-sunburst", quality)
		}
		return "image2.5_sunburst_" + normalizedQuality, normalizedQuality, nil
	case "gpt-image-2.5-flare":
		if !isImage25Quality(normalizedQuality) {
			return "", "", fmt.Errorf("quality %q is not supported by gpt-image-2.5-flare", quality)
		}
		return "image2.5_flare_" + normalizedQuality, normalizedQuality, nil
	default:
		return "", "", fmt.Errorf("unsupported Tencent VOD OG model %q", modelName)
	}
}

func isImage25Quality(quality string) bool {
	switch quality {
	case "low", "medium", "high", "xhigh", "max":
		return true
	default:
		return false
	}
}

func buildOGExtInfo(size string, rawBackground []byte) (string, error) {
	size = strings.ToLower(strings.TrimSpace(size))
	if size == "auto" {
		size = ""
	}
	if size != "" {
		widthText, heightText, found := strings.Cut(size, "x")
		if !found || strings.Contains(heightText, "x") {
			return "", fmt.Errorf("invalid size %q; expected WIDTHxHEIGHT", size)
		}
		width, widthErr := strconv.Atoi(widthText)
		height, heightErr := strconv.Atoi(heightText)
		pixels := int64(width) * int64(height)
		if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 || width%16 != 0 || height%16 != 0 || pixels < 655360 || pixels > 8294400 {
			return "", fmt.Errorf("invalid Tencent VOD OG size %q: width and height must be divisible by 16 and total pixels must be between 655360 and 8294400", size)
		}
	}

	background := ""
	if hasJSONValue(rawBackground) {
		if err := common.Unmarshal(rawBackground, &background); err != nil {
			return "", errors.New("background must be a string")
		}
		background = strings.ToLower(strings.TrimSpace(background))
		switch background {
		case "", "auto", "opaque":
			background = ""
		case "transparent":
		default:
			return "", fmt.Errorf("unsupported background %q", background)
		}
	}
	if size == "" && background == "" {
		return "", nil
	}

	additional, err := common.Marshal(struct {
		Size       string `json:"size,omitempty"`
		Background string `json:"background,omitempty"`
	}{Size: size, Background: background})
	if err != nil {
		return "", err
	}
	extInfo, err := common.Marshal(struct {
		AdditionalParameters string `json:"AdditionalParameters"`
	}{AdditionalParameters: string(additional)})
	if err != nil {
		return "", err
	}
	return string(extInfo), nil
}

func validateOpenAIImageCompatibility(request dto.ImageRequest) error {
	unsupported := []struct {
		name  string
		value []byte
	}{
		{name: "style", value: request.Style},
		{name: "user", value: request.User},
		{name: "extra_fields", value: request.ExtraFields},
		{name: "moderation", value: request.Moderation},
		{name: "output_compression", value: request.OutputCompression},
		{name: "partial_images", value: request.PartialImages},
		{name: "input_fidelity", value: request.InputFidelity},
		{name: "mask", value: request.Mask},
	}
	for _, field := range unsupported {
		if hasJSONValue(field.value) {
			return fmt.Errorf("Tencent VOD OG does not support parameter %s", field.name)
		}
	}
	if hasJSONValue(request.OutputFormat) {
		var outputFormat string
		if err := common.Unmarshal(request.OutputFormat, &outputFormat); err != nil || !strings.EqualFold(strings.TrimSpace(outputFormat), "png") {
			return errors.New("Tencent VOD OG only supports output_format=png")
		}
	}
	return nil
}

func hasJSONValue(value []byte) bool {
	trimmed := strings.TrimSpace(string(value))
	return trimmed != "" && trimmed != "null" && trimmed != `""`
}

func parseDocumentedExtraParameters(extra map[string]json.RawMessage) (string, string, *int, *int, error) {
	var negativePrompt string
	var enhancePrompt string
	var seed *int
	var tasksPriority *int
	for key, value := range extra {
		switch key {
		case "negative_prompt":
			if err := common.Unmarshal(value, &negativePrompt); err != nil {
				return "", "", nil, nil, errors.New("negative_prompt must be a string")
			}
		case "enhance_prompt":
			var enabled bool
			if err := common.Unmarshal(value, &enabled); err == nil {
				if enabled {
					enhancePrompt = "Enabled"
				} else {
					enhancePrompt = "Disabled"
				}
				continue
			}
			if err := common.Unmarshal(value, &enhancePrompt); err != nil {
				return "", "", nil, nil, errors.New("enhance_prompt must be a boolean or Enabled/Disabled")
			}
			if !strings.EqualFold(enhancePrompt, "Enabled") && !strings.EqualFold(enhancePrompt, "Disabled") {
				return "", "", nil, nil, errors.New("enhance_prompt must be Enabled or Disabled")
			}
			if strings.EqualFold(enhancePrompt, "Enabled") {
				enhancePrompt = "Enabled"
			} else {
				enhancePrompt = "Disabled"
			}
		case "seed":
			var parsed int
			if err := common.Unmarshal(value, &parsed); err != nil {
				return "", "", nil, nil, errors.New("seed must be an integer")
			}
			seed = &parsed
		case "tasks_priority":
			var parsed int
			if err := common.Unmarshal(value, &parsed); err != nil || parsed < -10 || parsed > 10 {
				return "", "", nil, nil, errors.New("tasks_priority must be an integer between -10 and 10")
			}
			tasksPriority = &parsed
		default:
			return "", "", nil, nil, fmt.Errorf("Tencent VOD OG does not support parameter %s", key)
		}
	}
	return negativePrompt, enhancePrompt, seed, tasksPriority, nil
}

func synchronousInputFileInfos(c *gin.Context, request dto.ImageRequest) ([]inputFileInfo, error) {
	values := make([]string, 0)
	for _, raw := range [][]byte{request.Image, request.Images} {
		if !hasJSONValue(raw) {
			continue
		}
		var single string
		if err := common.Unmarshal(raw, &single); err == nil {
			values = append(values, single)
			continue
		}
		var multiple []string
		if err := common.Unmarshal(raw, &multiple); err != nil {
			return nil, errors.New("image must be a string or an array of strings")
		}
		values = append(values, multiple...)
	}
	fileInfos, err := buildInputFileInfos(values)
	if err != nil {
		return nil, err
	}

	if c == nil || c.Request == nil || c.Request.MultipartForm == nil {
		return fileInfos, nil
	}
	form := c.Request.MultipartForm
	if maskFiles := form.File["mask"]; len(maskFiles) > 0 {
		return nil, errors.New("Tencent VOD OG does not support parameter mask")
	}
	keys := make([]string, 0)
	for key := range form.File {
		if key == "image" || key == "image[]" || strings.HasPrefix(key, "image[") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	totalBytes := 0
	for _, key := range keys {
		for _, fileHeader := range form.File[key] {
			file, openErr := fileHeader.Open()
			if openErr != nil {
				return nil, fmt.Errorf("open reference image: %w", openErr)
			}
			remaining := maxBase64Bytes - totalBytes
			data, readErr := io.ReadAll(io.LimitReader(file, int64(remaining+1)))
			_ = file.Close()
			if readErr != nil {
				return nil, fmt.Errorf("read reference image: %w", readErr)
			}
			if len(data) > remaining {
				return nil, errors.New("reference images exceed the 7 MB Tencent VOD limit")
			}
			contentType := http.DetectContentType(data)
			if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
				return nil, fmt.Errorf("unsupported reference image type: %s", contentType)
			}
			totalBytes += len(data)
			fileInfos = append(fileInfos, inputFileInfo{Type: "Base64", Base64: base64.StdEncoding.EncodeToString(data)})
		}
	}
	return fileInfos, nil
}
