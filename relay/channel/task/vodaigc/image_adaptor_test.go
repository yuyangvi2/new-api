package vodaigc

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, describeAction, r.Header.Get("X-TC-Action"))
		_, _ = w.Write([]byte(`{"Response":{"Status":"FINISH","AigcImageTask":{"TaskId":"task-1","Status":"FINISH","ErrCode":0,"Progress":100,"Output":{"FileInfos":[{"FileUrl":"https://example.com/result.png"}]}}}}`))
	}))
	defer server.Close()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBuffer(nil))
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:         "secret-id|secret-key",
			ChannelBaseUrl: server.URL,
			ChannelOtherSettings: dto.ChannelOtherSettings{
				DisableTaskPollingSleep: true,
			},
		},
	}
	adaptor := &ImageAdaptor{
		apiKey:         "secret-id|secret-key",
		settings:       dto.VODAIGCSettings{SubAppID: 123},
		responseFormat: "url",
	}
	submitHTTPResponse := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(`{"Response":{"TaskId":"task-1","RequestId":"request-1"}}`)),
	}

	usage, apiErr := adaptor.DoResponse(c, submitHTTPResponse, info)
	require.Nil(t, apiErr)
	require.IsType(t, &dto.Usage{}, usage)
	var imageResponse dto.ImageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &imageResponse))
	require.Len(t, imageResponse.Data, 1)
	assert.Equal(t, "https://example.com/result.png", imageResponse.Data[0].Url)
}
