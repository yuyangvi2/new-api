package vodaigc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOGModelVersionMapping(t *testing.T) {
	tests := []struct {
		model   string
		quality string
		want    string
		wantErr bool
	}{
		{model: "gpt-image-2", quality: "low", want: "image2_low"},
		{model: "gpt-image-2", quality: "", want: "image2_medium"},
		{model: "gpt-image-2", quality: "auto", want: "image2_medium"},
		{model: "gpt-image-2", quality: "standard", want: "image2_medium"},
		{model: "gpt-image-2", quality: "high", want: "image2_high"},
		{model: "gpt-image-2", quality: "xhigh", wantErr: true},
		{model: "gpt-image-2.5-sunburst", quality: "medium", want: "image2.5_sunburst_medium"},
		{model: "gpt-image-2.5-sunburst", quality: "max", want: "image2.5_sunburst_max"},
		{model: "gpt-image-2.5-flare", quality: "xhigh", want: "image2.5_flare_xhigh"},
		{model: "gpt-image-2.5-flare", quality: "invalid", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.model+"/"+tt.quality, func(t *testing.T) {
			got, _, err := ogModelVersion(tt.model, tt.quality)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildOGExtInfo(t *testing.T) {
	background, err := common.Marshal("transparent")
	require.NoError(t, err)
	extInfo, err := buildOGExtInfo("1792x1024", background)
	require.NoError(t, err)

	var outer struct {
		AdditionalParameters string `json:"AdditionalParameters"`
	}
	require.NoError(t, common.UnmarshalJsonStr(extInfo, &outer))
	var additional map[string]string
	require.NoError(t, common.UnmarshalJsonStr(outer.AdditionalParameters, &additional))
	assert.Equal(t, "1792x1024", additional["size"])
	assert.Equal(t, "transparent", additional["background"])

	_, err = buildOGExtInfo("1023x1024", nil)
	require.Error(t, err)
	_, err = buildOGExtInfo("512x512", nil)
	require.Error(t, err)
}

func TestImageAdaptorConvertsOpenAIRequestToOG(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/images/generations", bytes.NewBuffer(nil))
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		OriginModelName: "gpt-image-2.5-flare",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:         "secret-id|secret-key",
			ChannelBaseUrl: "https://vod.tencentcloudapi.com",
			ChannelOtherSettings: dto.ChannelOtherSettings{VODAIGC: &dto.VODAIGCSettings{
				SubAppID:    123,
				InputRegion: "Oversea",
			}},
		},
	}
	adaptor := &ImageAdaptor{}
	adaptor.Init(info)
	background, err := common.Marshal("transparent")
	require.NoError(t, err)
	converted, err := adaptor.ConvertImageRequest(c, info, dto.ImageRequest{
		Model:          "gpt-image-2.5-flare",
		Prompt:         "test prompt",
		Quality:        "medium",
		Size:           "1024x1024",
		Background:     background,
		ResponseFormat: "url",
	})
	require.NoError(t, err)

	buffer, ok := converted.(*bytes.Buffer)
	require.True(t, ok)
	var request createRequest
	require.NoError(t, common.Unmarshal(buffer.Bytes(), &request))
	assert.Equal(t, int64(123), request.SubAppID)
	assert.Equal(t, "OG", request.ModelName)
	assert.Equal(t, "image2.5_flare_medium", request.ModelVersion)
	assert.Equal(t, "Oversea", request.InputRegion)
	assert.NotEmpty(t, request.ExtInfo)
}

func TestImageAdaptorRejectsUnsupportedOutputCount(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/images/generations", bytes.NewBuffer(nil))
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		OriginModelName: "gpt-image-2",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:         "secret-id|secret-key",
			ChannelBaseUrl: "https://vod.tencentcloudapi.com",
		},
	}
	adaptor := &ImageAdaptor{}
	adaptor.Init(info)
	n := uint(2)
	_, err := adaptor.ConvertImageRequest(c, info, dto.ImageRequest{Model: "gpt-image-2", Prompt: "test", N: &n})
	require.Error(t, err)
}

func TestImageAdaptorConvertsJSONReferenceImagesAfterAzureFallback(t *testing.T) {
	const pngBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB"
	tests := []struct {
		name   string
		model  string
		fields string
		want   []inputFileInfo
	}{
		{
			name:   "single URL",
			model:  "gpt-image-2",
			fields: `"image":"https://example.com/first.png"`,
			want:   []inputFileInfo{{Type: "Url", URL: "https://example.com/first.png"}},
		},
		{
			name:   "URL array",
			model:  "gpt-image-2",
			fields: `"images":["https://example.com/first.png","https://example.com/second.png"]`,
			want:   []inputFileInfo{{Type: "Url", URL: "https://example.com/first.png"}, {Type: "Url", URL: "https://example.com/second.png"}},
		},
		{
			name:   "image URL objects",
			model:  "gpt-image-2.5-sunburst",
			fields: `"images":[{"image_url":"https://example.com/first.png"},{"image_url":"https://example.com/second.png"}]`,
			want:   []inputFileInfo{{Type: "Url", URL: "https://example.com/first.png"}, {Type: "Url", URL: "https://example.com/second.png"}},
		},
		{
			name:   "single image URL object",
			model:  "gpt-image-2",
			fields: `"image":{"image_url":"https://example.com/first.png"}`,
			want:   []inputFileInfo{{Type: "Url", URL: "https://example.com/first.png"}},
		},
		{
			name:   "mixed URL and base64 references preserve order",
			model:  "gpt-image-2.5-flare",
			fields: `"image":"https://example.com/first.png","images":[{"image_url":"data:image/png;base64,` + pngBase64 + `"},"https://example.com/second.png"]`,
			want:   []inputFileInfo{{Type: "Url", URL: "https://example.com/first.png"}, {Type: "Base64", Base64: pngBase64}, {Type: "Url", URL: "https://example.com/second.png"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := `{"model":"` + test.model + `","prompt":"edit this image",` + test.fields + `}`
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")
			var request dto.ImageRequest
			require.NoError(t, common.UnmarshalJsonStr(body, &request))
			info := &relaycommon.RelayInfo{
				RelayMode:       relayconstant.RelayModeImagesEdits,
				OriginModelName: test.model,
				ChannelMeta:     &relaycommon.ChannelMeta{},
			}
			_, err := (&openai.Adaptor{}).ConvertImageRequest(c, info, request)
			require.NoError(t, err)
			retryRequest, err := common.DeepCopy(&request)
			require.NoError(t, err)
			adaptor := &ImageAdaptor{}
			adaptor.Init(info)
			converted, err := adaptor.ConvertImageRequest(c, info, *retryRequest)
			require.NoError(t, err)
			buffer, ok := converted.(*bytes.Buffer)
			require.True(t, ok)
			var vodRequest createRequest
			require.NoError(t, common.Unmarshal(buffer.Bytes(), &vodRequest))
			assert.Equal(t, "OG", vodRequest.ModelName)
			assert.Equal(t, test.want, vodRequest.FileInfos)
			assert.Equal(t, request.Image, retryRequest.Image)
			assert.Equal(t, request.Images, retryRequest.Images)
		})
	}
}

func TestImageAdaptorRejectsInvalidReferenceImagesWithoutRetry(t *testing.T) {
	tests := []struct {
		name   string
		images string
	}{
		{name: "file ID cannot be resolved by VOD", images: `[{"file_id":"file-123"}]`},
		{name: "missing image URL", images: `[{}]`},
		{name: "empty image URL", images: `[{"image_url":""}]`},
		{name: "invalid image URL type", images: `[{"image_url":123}]`},
		{name: "null reference", images: `[null]`},
		{name: "invalid reference type", images: `[123]`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", nil)
			info := &relaycommon.RelayInfo{
				RelayMode:       relayconstant.RelayModeImagesEdits,
				OriginModelName: "gpt-image-2",
				ChannelMeta:     &relaycommon.ChannelMeta{},
			}
			adaptor := &ImageAdaptor{}
			adaptor.Init(info)
			_, err := adaptor.ConvertImageRequest(c, info, dto.ImageRequest{Prompt: "edit", Images: []byte(test.images)})
			require.Error(t, err)
			apiErr := types.NewError(err, types.ErrorCodeConvertRequestFailed)
			assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
			assert.True(t, types.IsSkipRetryError(apiErr))
			assert.NotContains(t, c.Keys, ctxPayloadKey)
		})
	}
}

func TestImageAdaptorPreservesMultipartReferencesAfterAzureFallback(t *testing.T) {
	const pngBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB"
	imageBytes, err := base64.StdEncoding.DecodeString(pngBase64)
	require.NoError(t, err)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "edit this image"))
	part, err := writer.CreateFormFile("image[]", "reference.png")
	require.NoError(t, err)
	_, err = part.Write(imageBytes)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, c.Request.ParseMultipartForm(1024))
	t.Cleanup(func() { require.NoError(t, c.Request.MultipartForm.RemoveAll()) })
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImagesEdits,
		OriginModelName: "gpt-image-2",
		ChannelMeta:     &relaycommon.ChannelMeta{},
	}
	request := dto.ImageRequest{Model: "gpt-image-2", Prompt: "edit this image"}
	azureBody, err := (&openai.Adaptor{}).ConvertImageRequest(c, info, request)
	require.NoError(t, err)
	closer, ok := azureBody.(io.Closer)
	require.True(t, ok)
	t.Cleanup(func() { require.NoError(t, closer.Close()) })
	adaptor := &ImageAdaptor{}
	adaptor.Init(info)
	converted, err := adaptor.ConvertImageRequest(c, info, request)
	require.NoError(t, err)
	buffer, ok := converted.(*bytes.Buffer)
	require.True(t, ok)
	var vodRequest createRequest
	require.NoError(t, common.Unmarshal(buffer.Bytes(), &vodRequest))
	assert.Equal(t, []inputFileInfo{{Type: "Base64", Base64: pngBase64}}, vodRequest.FileInfos)
}

func TestImageAdaptorBoundsCombinedBase64AndMultipartReferences(t *testing.T) {
	const pngBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB"
	imageBytes, err := base64.StdEncoding.DecodeString(pngBase64)
	require.NoError(t, err)
	for _, totalBytes := range []int{maxBase64Bytes, maxBase64Bytes + 1} {
		t.Run(fmt.Sprintf("combined bytes %d", totalBytes), func(t *testing.T) {
			inlineBytes := make([]byte, totalBytes-len(imageBytes))
			copy(inlineBytes, imageBytes)
			inlineImage, err := common.Marshal("data:image/png;base64," + base64.StdEncoding.EncodeToString(inlineBytes))
			require.NoError(t, err)
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("image", "reference.png")
			require.NoError(t, err)
			_, err = part.Write(imageBytes)
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
			c.Request.Header.Set("Content-Type", writer.FormDataContentType())
			require.NoError(t, c.Request.ParseMultipartForm(1024))
			t.Cleanup(func() { require.NoError(t, c.Request.MultipartForm.RemoveAll()) })
			fileInfos, err := synchronousInputFileInfos(c, dto.ImageRequest{Image: inlineImage})
			if totalBytes > maxBase64Bytes {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "7 MB Tencent VOD limit")
				return
			}
			require.NoError(t, err)
			require.Len(t, fileInfos, 2)
			assert.Equal(t, "Base64", fileInfos[0].Type)
			assert.Equal(t, pngBase64, fileInfos[1].Base64)
		})
	}
}

func TestImageAdaptorBillingUsagePreservesPreConsumeInputs(t *testing.T) {
	adaptor := &ImageAdaptor{}
	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			EstimatedPromptTokens:     321,
			EstimatedCompletionTokens: 1584,
		},
	}
	usage := adaptor.billingUsage(info)
	assert.Equal(t, 321, usage.PromptTokens)
	assert.Equal(t, 1584, usage.CompletionTokens)
	assert.Equal(t, 1905, usage.TotalTokens)

	info.TieredBillingSnapshot = nil
	info.SetEstimatePromptTokens(12)
	usage = adaptor.billingUsage(info)
	assert.Equal(t, common.PreConsumedQuota+(&dto.ImageRequest{}).GetTokenCountMeta().MaxTokens, usage.PromptTokens)
	assert.Equal(t, usage.PromptTokens, usage.TotalTokens)
}

func TestImageAdaptorReturnsOpenAIImageResponseAfterTencentTaskCompletes(t *testing.T) {
	service.InitHttpClient()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, vodVersion, r.Header.Get("X-TC-Version"))
		assert.Contains(t, r.Header.Get("Authorization"), "TC3-HMAC-SHA256 Credential=secret-id/")
		w.Header().Set("Content-Type", "application/json")
		switch r.Header.Get("X-TC-Action") {
		case createAction:
			var request createRequest
			if !assert.NoError(t, common.DecodeJson(r.Body, &request)) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			assert.Equal(t, int64(123), request.SubAppID)
			assert.Equal(t, "OG", request.ModelName)
			assert.Equal(t, "image2.5_sunburst_medium", request.ModelVersion)
			assert.Equal(t, []inputFileInfo{{Type: "Url", URL: "https://example.com/reference.png"}}, request.FileInfos)
			_, _ = w.Write([]byte(`{"Response":{"TaskId":"task-1","RequestId":"request-1"}}`))
		case describeAction:
			var request map[string]any
			if !assert.NoError(t, common.DecodeJson(r.Body, &request)) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			assert.Equal(t, float64(123), request["SubAppId"])
			assert.Equal(t, "task-1", request["TaskId"])
			_, _ = w.Write([]byte(`{"Response":{"Status":"FINISH","AigcImageTask":{"TaskId":"task-1","Status":"FINISH","ErrCode":0,"Progress":100,"Output":{"FileInfos":[{"FileUrl":"https://example.com/result.png"}]}}}}`))
		default:
			assert.Fail(t, "unexpected Tencent VOD action", "%s", r.Header.Get("X-TC-Action"))
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewBuffer(nil))
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImagesEdits,
		OriginModelName: "gpt-image-2.5-sunburst",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:         "secret-id|secret-key",
			ChannelBaseUrl: server.URL,
			ChannelOtherSettings: dto.ChannelOtherSettings{
				DisableTaskPollingSleep: true,
				VODAIGC:                 &dto.VODAIGCSettings{SubAppID: 123},
			},
		},
	}
	adaptor := &ImageAdaptor{}
	adaptor.Init(info)
	converted, err := adaptor.ConvertImageRequest(c, info, dto.ImageRequest{
		Prompt:         "edit this image",
		ResponseFormat: "url",
		Images:         []byte(`[{"image_url":"https://example.com/reference.png"}]`),
	})
	require.NoError(t, err)
	requestBody, ok := converted.(io.Reader)
	require.True(t, ok)
	response, err := adaptor.DoRequest(c, info, requestBody)
	require.NoError(t, err)
	submitHTTPResponse, ok := response.(*http.Response)
	require.True(t, ok)

	usage, apiErr := adaptor.DoResponse(c, submitHTTPResponse, info)
	require.Nil(t, apiErr)
	require.IsType(t, &dto.Usage{}, usage)
	var imageResponse dto.ImageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &imageResponse))
	require.Len(t, imageResponse.Data, 1)
	assert.Equal(t, "https://example.com/result.png", imageResponse.Data[0].Url)
}
