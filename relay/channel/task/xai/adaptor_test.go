package xai

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateBillingUsesGrokImagineVideo15ResolutionPrice(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() { common.QuotaPerUnit = originalQuotaPerUnit })
	common.QuotaPerUnit = 1_000_000

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("task_request", relaycommon.TaskSubmitReq{
		Model:    "grok-imagine-video-1.5",
		Image:    "https://example.com/input.jpg",
		Duration: 8,
		Metadata: map[string]any{"resolution": "720p"},
	})

	ratios := (&TaskAdaptor{}).EstimateBilling(ctx, &relaycommon.RelayInfo{
		OriginModelName: "grok-imagine-video-1.5",
		PriceData: types.PriceData{
			Quota:          80_000,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	})

	got, ok := ratios["xai_imagine_price"]
	require.True(t, ok)
	assert.InDelta(t, 1.13/0.08, got, 0.000001)
}

func TestValidateRequestRejectsVideo15WithoutImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(`{
		"model":"grok-imagine-video-1.5",
		"prompt":"animate it",
		"duration":8
	}`))
	req.Header.Set("Content-Type", "application/json")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	info := &relaycommon.RelayInfo{OriginModelName: "grok-imagine-video-1.5", TaskRelayInfo: &relaycommon.TaskRelayInfo{}}

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)

	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "image is required")
}

func TestValidateRequestPreservesTopLevelResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(`{
		"model":"grok-imagine-video-1.5",
		"prompt":"animate it",
		"image":"https://example.com/input.jpg",
		"duration":6,
		"resolution":"480p",
		"aspect_ratio":"16:9"
	}`))
	req.Header.Set("Content-Type", "application/json")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	info := &relaycommon.RelayInfo{OriginModelName: "grok-imagine-video-1.5", TaskRelayInfo: &relaycommon.TaskRelayInfo{}}

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.Nil(t, taskErr)
	storedReq, err := relaycommon.GetTaskRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, "480p", storedReq.Metadata["resolution"])
	assert.Equal(t, "16:9", storedReq.Metadata["aspect_ratio"])
}

func TestParseTaskResultExtractsURLAndCostTicks(t *testing.T) {
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

func TestAdjustBillingOnCompleteUsesUpstreamFinalUSDCost(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 500000

	task := &model.Task{
		Properties: model.Properties{OriginModelName: "grok-imagine-video-1.5"},
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				ModelPrice:      0.08 / 0.6,
				GroupRatio:      1,
				OriginModelName: "grok-imagine-video-1.5",
			},
		},
	}

	quota, clamp := (&TaskAdaptor{}).AdjustBillingOnCompleteWithClamp(task, &relaycommon.TaskInfo{BillingUnits: 2.01})

	require.Nil(t, clamp)
	assert.Equal(t, 1004999, quota)
}

func TestAdjustBillingOnCompleteAppliesGroupRatioToUpstreamFinalUSDCost(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 500000

	task := &model.Task{
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				GroupRatio: 0.6,
			},
		},
	}

	quota, clamp := (&TaskAdaptor{}).AdjustBillingOnCompleteWithClamp(task, &relaycommon.TaskInfo{BillingUnits: 2.0})

	require.Nil(t, clamp)
	assert.Equal(t, 600000, quota)
}
