package doubao

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedanceOfficialParseTaskResultSuccess(t *testing.T) {
	result, err := (&SeedanceOfficialTaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_123",
		"status":"succeeded",
		"content":{"video_url":"https://example.com/video.mp4"},
		"usage":{"completion_tokens":12,"total_tokens":34}
	}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "task_123", result.TaskID)
	assert.Equal(t, model.TaskStatusSuccess, result.Status)
	assert.Equal(t, taskcommon.ProgressComplete, result.Progress)
	assert.Equal(t, "https://example.com/video.mp4", result.Url)
	assert.Equal(t, 12, result.CompletionTokens)
	assert.Equal(t, 34, result.TotalTokens)
}

func TestSeedanceOfficialParseTaskResultFailure(t *testing.T) {
	result, err := (&SeedanceOfficialTaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_123",
		"status":"failed",
		"error":{"code":"InvalidParameter","message":"bad request"}
	}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, model.TaskStatusFailure, result.Status)
	assert.Equal(t, taskcommon.ProgressComplete, result.Progress)
	assert.Equal(t, "bad request", result.Reason)
}

func TestSeedanceOfficialParseTaskResultErrorWithoutStatus(t *testing.T) {
	result, err := (&SeedanceOfficialTaskAdaptor{}).ParseTaskResult([]byte(`{
		"ResponseMetadata":{"Error":{"Code":"AccessDenied","Message":"invalid api key"}}
	}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, model.TaskStatusFailure, result.Status)
	assert.Equal(t, taskcommon.ProgressComplete, result.Progress)
	assert.Equal(t, "invalid api key", result.Reason)
}

func TestSeedanceOfficialParseTaskResultMissingStatusWithoutError(t *testing.T) {
	result, err := (&SeedanceOfficialTaskAdaptor{}).ParseTaskResult([]byte(`{"id":"task_123"}`))

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestSeedanceOfficialParseTaskResultUnknownStatusFails(t *testing.T) {
	result, err := (&SeedanceOfficialTaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_123",
		"status":"paused"
	}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, model.TaskStatusFailure, result.Status)
	assert.Equal(t, taskcommon.ProgressComplete, result.Progress)
	assert.Equal(t, "unknown status: paused", result.Reason)
}

func TestSeedanceOfficialValidateRequestBounds(t *testing.T) {
	hugeDuration := dto.IntValue(3601)
	hugeFrames := dto.IntValue(86401)

	tests := []struct {
		name string
		req  *requestPayload
	}{
		{
			name: "duration too large",
			req: &requestPayload{
				Model:    "doubao-seedance-2-0-260128",
				Content:  []ContentItem{{Type: "text", Text: "make a video"}},
				Duration: &hugeDuration,
			},
		},
		{
			name: "frames too large",
			req: &requestPayload{
				Model:   "doubao-seedance-2-0-260128",
				Content: []ContentItem{{Type: "text", Text: "make a video"}},
				Frames:  &hugeFrames,
			},
		},
		{
			name: "unsupported ratio",
			req: &requestPayload{
				Model:   "doubao-seedance-2-0-260128",
				Content: []ContentItem{{Type: "text", Text: "make a video"}},
				Ratio:   "999999999:1",
			},
		},
		{
			name: "unsupported resolution",
			req: &requestPayload{
				Model:      "doubao-seedance-2-0-260128",
				Content:    []ContentItem{{Type: "text", Text: "make a video"}},
				Resolution: "999999999p",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, validateSeedanceOfficialRequest(tt.req))
		})
	}
}

func TestSeedanceOfficialValidateRequestAllowsAutoDuration(t *testing.T) {
	autoDuration := dto.IntValue(-1)
	req := &requestPayload{
		Model:      "doubao-seedance-2-0-260128",
		Content:    []ContentItem{{Type: "text", Text: "make a video"}},
		Duration:   &autoDuration,
		Ratio:      "16:9",
		Resolution: "720p",
	}

	assert.NoError(t, validateSeedanceOfficialRequest(req))
}

func TestSeedanceOfficialEstimateBillingDistinguishesVideoInput(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	originalExchangeRate := operation_setting.USDExchangeRate
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
		operation_setting.USDExchangeRate = originalExchangeRate
	})
	common.QuotaPerUnit = 1_000_000
	operation_setting.USDExchangeRate = 1

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	duration := dto.IntValue(5)
	videoReq := requestPayload{
		Model:      "doubao-seedance-2-0-fast",
		Content:    []ContentItem{{Type: "text", Text: "make a video"}, {Type: "video_url", VideoURL: &MediaURL{URL: "https://example.com/ref.mp4"}}},
		Duration:   &duration,
		Ratio:      "16:9",
		Resolution: "720p",
	}
	ctx.Set(seedanceOfficialTaskContextKey, videoReq)

	info := &relaycommon.RelayInfo{
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		OriginModelName: "doubao-seedance-2-0-fast",
		PriceData: types.PriceData{
			Quota:          3_996_000,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	}
	ratios := (&SeedanceOfficialTaskAdaptor{}).EstimateBilling(ctx, info)

	require.Contains(t, ratios, "seedance_official_estimated_price")
	assert.InDelta(t, 22.0/37.0, ratios["seedance_official_estimated_price"], 0.000001)
	require.NotNil(t, info.TaskRelayInfo.UsageBilling)
	assert.Equal(t, relaycommon.TaskPricingSourceSeedanceOfficialUsage, info.TaskRelayInfo.UsageBilling.PricingSource)
	assert.Equal(t, 22.0, info.TaskRelayInfo.UsageBilling.PricePerMillionCNY)
	assert.Equal(t, "720p", info.TaskRelayInfo.UsageBilling.Resolution)
	require.NotNil(t, info.TaskRelayInfo.UsageBilling.HasVideoInput)
	assert.True(t, *info.TaskRelayInfo.UsageBilling.HasVideoInput)
}

func TestSeedanceOfficialAdjustBillingOnCompleteUsesUpstreamUsageTokens(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 500000

	hasVideoInput := false
	task := &model.Task{
		Quota: 34094,
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-2-0-mini",
		},
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				GroupRatio:      0.98,
				OriginModelName: "doubao-seedance-2-0-mini",
				UsageBilling: &relaycommon.TaskUsageBillingContext{
					PricingSource:      relaycommon.TaskPricingSourceSeedanceOfficialUsage,
					PricePerMillionCNY: 23.0,
					USDExchangeRate:    7.14,
					Resolution:         "480p",
					HasVideoInput:      &hasVideoInput,
				},
			},
		},
		Data: []byte(`{"resolution":"480p","usage":{"total_tokens":38800}}`),
	}

	quota, clamp := (&SeedanceOfficialTaskAdaptor{}).AdjustBillingOnCompleteWithClamp(task, &relaycommon.TaskInfo{
		TotalTokens: 38800,
	})

	require.Nil(t, clamp)
	assert.Equal(t, 61243, quota)
}

func TestSeedanceOfficialAdjustBillingOnCompleteUsesVideoInputPrice(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})
	common.QuotaPerUnit = 500000

	hasVideoInput := true
	task := &model.Task{
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-2-0-mini",
		},
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				GroupRatio:      0.98,
				OriginModelName: "doubao-seedance-2-0-mini",
				UsageBilling: &relaycommon.TaskUsageBillingContext{
					PricingSource:      relaycommon.TaskPricingSourceSeedanceOfficialUsage,
					PricePerMillionCNY: 14.0,
					USDExchangeRate:    7.14,
					Resolution:         "480p",
					HasVideoInput:      &hasVideoInput,
				},
			},
		},
		Data: []byte(`{"resolution":"480p"}`),
	}

	quota, clamp := (&SeedanceOfficialTaskAdaptor{}).AdjustBillingOnCompleteWithClamp(task, &relaycommon.TaskInfo{
		TotalTokens: 58000,
	})

	require.Nil(t, clamp)
	assert.Equal(t, 55725, quota)
}

func TestSeedanceOfficialAdjustBillingOnCompleteFallsBackToPrechargedQuotaWithoutUsageBillingContext(t *testing.T) {
	task := &model.Task{Quota: 12345}

	quota, clamp := (&SeedanceOfficialTaskAdaptor{}).AdjustBillingOnCompleteWithClamp(task, &relaycommon.TaskInfo{
		TotalTokens: 38800,
	})

	require.Nil(t, clamp)
	assert.Equal(t, 0, quota)
	assert.Equal(t, task.Quota, (&SeedanceOfficialTaskAdaptor{}).AdjustBillingOnComplete(task, &relaycommon.TaskInfo{}))
}
