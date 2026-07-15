package vclm

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestBuildRequestBodyDropsUnsupportedFrontendParamsForVidu(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Prompt:   "make it move",
		Model:    "viduq2-pro",
		Images:   []string{"https://example.com/image.png"},
		Duration: 5,
		Metadata: map[string]interface{}{
			"MovementAmplitude": "auto",
			"Resolution":        "720p",
			"resolution":        "1080p",
			"Payload":           "client-payload",
		},
	})

	adaptor := &TaskAdaptor{}
	body, err := adaptor.BuildRequestBody(c, &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{Action: "SubmitImageToVideoViduJob"},
		ChannelMeta:   &relaycommon.ChannelMeta{UpstreamModelName: "viduq2-pro"},
	})
	require.NoError(t, err)

	payload, err := io.ReadAll(body)
	require.NoError(t, err)
	var got map[string]interface{}
	require.NoError(t, common.Unmarshal(payload, &got))

	assert.NotContains(t, got, "MovementAmplitude")
	assert.NotContains(t, got, "Resolution")
	assert.NotContains(t, got, "resolution")
	assert.Equal(t, "client-payload", got["Payload"])
}

func TestBuildRequestBodyDropsUnsupportedFrontendParamsForKling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Prompt:   "make it move",
		Model:    "kling-v1-6",
		Image:    "https://example.com/image.png",
		Duration: 5,
		Metadata: map[string]interface{}{
			"MovementAmplitude": "auto",
			"Resolution":        "720p",
			"NegativePrompt":    "low quality",
			"CfgScale":          0.5,
		},
	})

	adaptor := &TaskAdaptor{}
	body, err := adaptor.BuildRequestBody(c, &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{Action: "SubmitImageToVideoJob"},
		ChannelMeta:   &relaycommon.ChannelMeta{UpstreamModelName: "kling-v1-6"},
	})
	require.NoError(t, err)

	payload, err := io.ReadAll(body)
	require.NoError(t, err)
	var got map[string]interface{}
	require.NoError(t, common.Unmarshal(payload, &got))

	assert.NotContains(t, got, "MovementAmplitude")
	assert.NotContains(t, got, "Resolution")
	assert.Equal(t, "low quality", got["NegativePrompt"])
	assert.Equal(t, 0.5, got["CfgScale"])
}

func TestAdjustBillingOnCompleteUsesFloatingFinalUnitDeduction(t *testing.T) {
	originalModelRatio := ratio_setting.ModelRatio2JSONString()
	originalGroupRatio := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(originalModelRatio))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatio))
	})

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"kling-v2-6":0.14}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"test-group":1}`))

	adaptor := &TaskAdaptor{}
	result, err := adaptor.ParseTaskResult([]byte(`{
		"Response": {
			"Status": "DONE",
			"ResultVideoUrl": "https://example.com/video.mp4",
			"FinalUnitDeduction": "1.5"
		}
	}`))
	require.NoError(t, err)

	assert.Equal(t, 1.5, result.BillingUnits)
	assert.Zero(t, result.TotalTokens)
	assert.Zero(t, result.CompletionTokens)

	actualQuota := adaptor.AdjustBillingOnComplete(&model.Task{
		Group: "test-group",
		Properties: model.Properties{
			OriginModelName: "kling-v2-6",
		},
	}, result)

	assert.Equal(t, common.QuotaFromFloat(1.5*0.14*common.QuotaPerUnit), actualQuota)
}
