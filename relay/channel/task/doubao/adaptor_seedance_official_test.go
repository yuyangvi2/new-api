package doubao

import (
	"io"
	"net/http/httptest"
	"strings"
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
	"github.com/tidwall/gjson"
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

func TestSeedanceOfficialRequestURL(t *testing.T) {
	adaptor := &SeedanceOfficialTaskAdaptor{}

	assert.Equal(
		t,
		"https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks",
		adaptor.requestURL("https://ark.cn-beijing.volces.com"),
	)
	assert.Equal(
		t,
		"https://gateway.example.com/volcengine/api/v3/contents/generations/tasks",
		adaptor.requestURL("https://gateway.example.com/volcengine"),
	)
	assert.Equal(
		t,
		"https://gateway.example.com/api/v3/contents/generations/tasks",
		adaptor.requestURL("https://gateway.example.com/api/v3/contents/generations/tasks"),
	)
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

func TestSeedanceOfficialValidateRequestRejectsSeedance25Unsupported4K(t *testing.T) {
	req := &requestPayload{
		Model:      "doubao-seedance-2-5",
		Content:    []ContentItem{{Type: "text", Text: "make a video"}},
		Resolution: "4k",
	}

	assert.Error(t, validateSeedanceOfficialRequest(req))
}

func TestSeedanceOfficialDirectRequestPreservesDataURIImageURL(t *testing.T) {
	dataURI := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB"
	body := `{
		"model":"doubao-seedance-2.0",
		"content":[
			{"type":"text","text":"animate this"},
			{"type":"image_url","image_url":{"url":"` + dataURI + `"}}
		],
		"resolution":"480p",
		"ratio":"1:1"
	}`
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/video/tasks", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	req, err := parseSeedanceOfficialRequest(ctx)

	require.NoError(t, err)
	require.Len(t, req.Content, 2)
	require.NotNil(t, req.Content[1].ImageURL)
	assert.Equal(t, dataURI, req.Content[1].ImageURL.URL)
	assert.NoError(t, validateSeedanceOfficialRequest(req))
}

func TestSeedanceOfficialGenericSingleImageDataURIBecomesImageURLContent(t *testing.T) {
	dataURI := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB"
	body := `{
		"model":"doubao-seedance-2.0",
		"prompt":"animate this",
		"image":"` + dataURI + `",
		"resolution":"480p",
		"aspect_ratio":"1:1",
		"duration":5
	}`
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/video/tasks", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	req, err := parseSeedanceOfficialRequest(ctx)

	require.NoError(t, err)
	require.Len(t, req.Content, 2)
	require.NotNil(t, req.Content[0].ImageURL)
	assert.Equal(t, dataURI, req.Content[0].ImageURL.URL)
	assert.Equal(t, "text", req.Content[1].Type)
}

func TestSeedanceOfficialBuildRequestBodyPreservesDataURIImageURL(t *testing.T) {
	dataURI := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB"
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set(seedanceOfficialTaskContextKey, requestPayload{
		Model: "doubao-seedance-2.0",
		Content: []ContentItem{
			{Type: "text", Text: "animate this"},
			{Type: "image_url", ImageURL: &MediaURL{URL: dataURI}},
		},
	})

	body, err := (&SeedanceOfficialTaskAdaptor{}).BuildRequestBody(ctx, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	})

	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var req requestPayload
	require.NoError(t, common.Unmarshal(data, &req))
	require.Len(t, req.Content, 2)
	require.NotNil(t, req.Content[1].ImageURL)
	assert.Equal(t, dataURI, req.Content[1].ImageURL.URL)
}

func TestSeedanceOfficialDirectRequestPreservesObjectTools(t *testing.T) {
	body := `{
		"model":"doubao-seedance-2.0",
		"content":[{"type":"text","text":"make a video"}],
		"tools":{"type":"web_search","config":{"max_results":3}}
	}`
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/video/tasks", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	req, err := parseSeedanceOfficialRequest(ctx)
	require.NoError(t, err)
	require.NoError(t, validateSeedanceOfficialRequest(req))
	ctx.Set(seedanceOfficialTaskContextKey, *req)

	out, err := (&SeedanceOfficialTaskAdaptor{}).BuildRequestBody(ctx, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	})
	require.NoError(t, err)
	data, err := io.ReadAll(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"web_search","config":{"max_results":3}}`, gjson.GetBytes(data, "tools").Raw)
}

func TestSeedanceOfficialDirectRequestPreservesArrayToolsWithExtraFields(t *testing.T) {
	body := `{
		"model":"doubao-seedance-2.0",
		"content":[{"type":"text","text":"make a video"}],
		"tools":[{"type":"web_search","config":{"max_results":3}}]
	}`
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/video/tasks", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	req, err := parseSeedanceOfficialRequest(ctx)
	require.NoError(t, err)
	require.NoError(t, validateSeedanceOfficialRequest(req))
	ctx.Set(seedanceOfficialTaskContextKey, *req)

	out, err := (&SeedanceOfficialTaskAdaptor{}).BuildRequestBody(ctx, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	})
	require.NoError(t, err)
	data, err := io.ReadAll(out)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"type":"web_search","config":{"max_results":3}}]`, gjson.GetBytes(data, "tools").Raw)
}

func TestSeedanceOfficialValidateRequestRejectsInvalidToolsType(t *testing.T) {
	body := `{
		"model":"doubao-seedance-2.0",
		"content":[{"type":"text","text":"make a video"}],
		"tools":"web_search"
	}`
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/video/tasks", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	req, err := parseSeedanceOfficialRequest(ctx)
	require.NoError(t, err)
	assert.Error(t, validateSeedanceOfficialRequest(req))
}

func TestSeedanceOfficialConvertToSeedanceVideoPreservesRunyuanTaskFields(t *testing.T) {
	task := &model.Task{
		TaskID:    "task_public",
		Status:    model.TaskStatusSuccess,
		CreatedAt: 1718049470,
		UpdatedAt: 1718049480,
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://example.com/public-video.mp4",
		},
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-2.0",
		},
		Data: []byte(`{
			"id":"cgt-upstream",
			"model":"doubao-seedance-2.0",
			"status":"succeeded",
			"error":null,
			"created_at":1718049470,
			"updated_at":1718049480,
			"content":{"video_url":"https://example.com/upstream-video.mp4"},
			"seed":0,
			"resolution":"720p",
			"ratio":"16:9",
			"duration":4,
			"framespersecond":24,
			"generate_audio":false,
			"tools":{},
			"safety_identifier":"",
			"draft":false,
			"draft_task_id":"",
			"execution_expires_after":3600,
			"usage":{"completion_tokens":35800,"total_tokens":35800}
		}`),
	}

	data, err := (&SeedanceOfficialTaskAdaptor{}).ConvertToSeedanceVideo(task)
	require.NoError(t, err)

	assert.Equal(t, "task_public", gjson.GetBytes(data, "id").String())
	assert.Equal(t, "succeeded", gjson.GetBytes(data, "status").String())
	assert.Equal(t, "https://example.com/public-video.mp4", gjson.GetBytes(data, "content.video_url").String())
	assert.Equal(t, int64(0), gjson.GetBytes(data, "seed").Int())
	assert.Equal(t, "720p", gjson.GetBytes(data, "resolution").String())
	assert.Equal(t, "16:9", gjson.GetBytes(data, "ratio").String())
	assert.Equal(t, int64(4), gjson.GetBytes(data, "duration").Int())
	assert.Equal(t, int64(24), gjson.GetBytes(data, "framespersecond").Int())
	assert.False(t, gjson.GetBytes(data, "generate_audio").Bool())
	assert.True(t, gjson.GetBytes(data, "generate_audio").Exists())
	assert.JSONEq(t, `{}`, gjson.GetBytes(data, "tools").Raw)
	assert.Equal(t, "", gjson.GetBytes(data, "safety_identifier").String())
	assert.True(t, gjson.GetBytes(data, "safety_identifier").Exists())
	assert.False(t, gjson.GetBytes(data, "draft").Bool())
	assert.True(t, gjson.GetBytes(data, "draft").Exists())
	assert.Equal(t, "", gjson.GetBytes(data, "draft_task_id").String())
	assert.True(t, gjson.GetBytes(data, "draft_task_id").Exists())
	assert.Equal(t, int64(3600), gjson.GetBytes(data, "execution_expires_after").Int())
	assert.True(t, gjson.GetBytes(data, "error").Exists())
	assert.Equal(t, "null", gjson.GetBytes(data, "error").Raw)
}

func TestSeedanceOfficialConvertToSeedanceVideoFailureOmitsEmptySuccessFields(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_public",
		Status:     model.TaskStatusFailure,
		FailReason: "sensitive content detected",
		CreatedAt:  100,
		UpdatedAt:  200,
		Data:       []byte(`{"id":"upstream_task","status":"failed"}`),
	}

	data, err := (&SeedanceOfficialTaskAdaptor{}).ConvertToSeedanceVideo(task)
	require.NoError(t, err)

	assert.Equal(t, "task_public", gjson.GetBytes(data, "id").String())
	assert.Equal(t, "failed", gjson.GetBytes(data, "status").String())
	assert.Equal(t, "sensitive content detected", gjson.GetBytes(data, "error.message").String())
	assert.False(t, gjson.GetBytes(data, "content").Exists())
	assert.False(t, gjson.GetBytes(data, "usage").Exists())
	assert.False(t, gjson.GetBytes(data, "ResponseMetadata").Exists())
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
	assert.InDelta(t, 2*22.0/37.0, ratios["seedance_official_estimated_price"], 0.000001)
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
