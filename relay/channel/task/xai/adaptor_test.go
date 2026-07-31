package xai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRequestRequiresImageForGrokImagineVideo15(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/videos/generations",
		strings.NewReader(`{"model":"grok-imagine-video-1.5","prompt":"animate this"}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	})

	require.NotNil(t, taskErr)
	assert.Equal(t, "missing_image", taskErr.Code)
	assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	assert.True(t, taskErr.LocalError)
}

func TestEstimateBillingUsesDurationResolutionAndInputImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/videos/generations",
		strings.NewReader(`{"model":"grok-imagine-video-1.5","prompt":"animate","image":"https://example.com/a.png","duration":10,"metadata":{"resolution":"1080p"}}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{UpstreamModelName: "grok-imagine-video-1.5"},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}

	require.Nil(t, (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info))
	ratios := (&TaskAdaptor{}).EstimateBilling(c, info)

	require.Contains(t, ratios, "xai_imagine_price")
	assert.InDelta(t, 31.375, ratios["xai_imagine_price"], 0.000001)
}

func TestBuildRequestBodyPreservesTopLevelResolutionAndAspectRatio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/videos/generations",
		strings.NewReader(`{"model":"grok-imagine-video-1.5","prompt":"animate","image":"https://example.com/a.png","duration":5,"resolution":"1080p","aspect_ratio":"16:9"}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{UpstreamModelName: "grok-imagine-video-1.5"},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
	adaptor := &TaskAdaptor{}

	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
	ratios := adaptor.EstimateBilling(c, info)
	bodyReader, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	body, err := io.ReadAll(bodyReader)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	assert.Equal(t, "1080p", payload["resolution"])
	assert.Equal(t, "16:9", payload["aspect_ratio"])
	assert.InDelta(t, 15.75, ratios["xai_imagine_price"], 0.000001)
}

func TestParseTaskResultExtractsCostTicks(t *testing.T) {
	body := []byte(`{
		"id":"task_upstream",
		"status":"completed",
		"usage":{"cost_in_usd_ticks":25100000000},
		"video":{"duration":10,"url":"https://example.com/video.mp4"}
	}`)

	taskResult, err := (&TaskAdaptor{}).ParseTaskResult(body)

	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), taskResult.Status)
	assert.Equal(t, "https://example.com/video.mp4", taskResult.Url)
	assert.InDelta(t, 2.51, taskResult.BillingUnits, 0.0000001)
}

func TestAdjustBillingOnCompleteScalesUpstreamUsageByConfiguredPrice(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 500000

	task := &model.Task{
		Properties: model.Properties{OriginModelName: "grok-imagine-video"},
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				ModelPrice:      0.05 / 0.6,
				GroupRatio:      0.6,
				OriginModelName: "grok-imagine-video",
			},
		},
	}

	quota, clamp := (&TaskAdaptor{}).AdjustBillingOnCompleteWithClamp(task, &relaycommon.TaskInfo{BillingUnits: 0.28})

	require.Nil(t, clamp)
	assert.Equal(t, 140000, quota)
}
