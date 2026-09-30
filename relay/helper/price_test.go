package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyTencentTokenHubSearchPreConsume(t *testing.T) {
	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		operation_setting.RebuildToolPriceIndex()
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"tool_price_setting.prices": `{"tencent_web_search_lite":0.98,"tencent_web_search_standard":1.68}`,
	}))
	operation_setting.RebuildToolPriceIndex()

	enabled := true
	tests := []struct {
		name      string
		baseURL   string
		options   *dto.WebSearchOptions
		initial   int
		wantQuota int
		wantFree  bool
	}{
		{
			name:      "lite search reserves one possible call",
			baseURL:   "https://tokenhub.tencentmaas.com",
			options:   &dto.WebSearchOptions{Enable: &enabled, SearchSource: "lite"},
			initial:   100,
			wantQuota: 590,
		},
		{
			name:      "missing search source defaults to standard",
			baseURL:   "https://tokenhub.tencentmaas.com/v1",
			options:   &dto.WebSearchOptions{Enable: &enabled},
			initial:   100,
			wantQuota: 940,
		},
		{
			name:      "other upstream is not charged",
			baseURL:   "https://api.openai.com",
			options:   &dto.WebSearchOptions{Enable: &enabled, SearchSource: "lite"},
			initial:   100,
			wantQuota: 100,
			wantFree:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				OriginModelName: "deepseek-v4-flash",
				Request: &dto.GeneralOpenAIRequest{
					WebSearchOptions: tc.options,
				},
				ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: tc.baseURL},
			}
			priceData := types.PriceData{
				FreeModel:         true,
				QuotaToPreConsume: tc.initial,
				GroupRatioInfo:    types.GroupRatioInfo{GroupRatio: 1},
			}

			applyTencentTokenHubSearchPreConsume(info, &priceData)

			assert.Equal(t, tc.wantQuota, priceData.QuotaToPreConsume)
			assert.Equal(t, tc.wantFree, priceData.FreeModel)
		})
	}
}

func TestModelPriceHelperTieredUsesPreloadedRequestInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode":    `{"tiered-test-model":"tiered_expr"}`,
		"billing_setting.billing_expr":    `{"tiered-test-model":"param(\"stream\") == true ? tier(\"stream\", p * 3) : tier(\"base\", p * 2)"}`,
		"group_ratio_setting.group_ratio": `{"default":1}`,
	}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/channel/test/1", nil)
	req.Body = nil
	req.ContentLength = 0
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "tiered-test-model",
		UserGroup:       "default",
		UsingGroup:      "default",
		RequestHeaders:  map[string]string{"Content-Type": "application/json"},
		BillingRequestInput: &billingexpr.RequestInput{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    []byte(`{"stream":true}`),
		},
	}

	priceData, err := ModelPriceHelper(ctx, info, 1000, &types.TokenCountMeta{})
	require.NoError(t, err)
	require.Equal(t, 1500, priceData.QuotaToPreConsume)
	require.NotNil(t, info.TieredBillingSnapshot)
	require.Equal(t, "stream", info.TieredBillingSnapshot.EstimatedTier)
	require.Equal(t, billing_setting.BillingModeTieredExpr, info.TieredBillingSnapshot.BillingMode)
	require.Equal(t, common.QuotaPerUnit, info.TieredBillingSnapshot.QuotaPerUnit)
}

func TestModelPriceHelperTieredPreConsumeMaxTokensFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode":    `{"tiered-fallback-model":"tiered_expr"}`,
		"billing_setting.billing_expr":    `{"tiered-fallback-model":"tier(\"base\", p * 3 + c * 15)"}`,
		"group_ratio_setting.group_ratio": `{"default":1,"free":0}`,
	}))

	const promptTokens = 1000

	cases := []struct {
		name      string
		group     string
		maxTokens int
		expected  int
	}{
		{
			name:      "non-free group falls back to 8192 completion tokens",
			group:     "default",
			maxTokens: 0,
			expected:  62940,
		},
		{
			name:      "explicit max_tokens is used verbatim",
			group:     "default",
			maxTokens: 100,
			expected:  2250,
		},
		{
			name:      "free group stays zero without fallback",
			group:     "free",
			maxTokens: 0,
			expected:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			req.Header.Set("Content-Type", "application/json")
			ctx.Request = req
			ctx.Set("group", tc.group)

			info := &relaycommon.RelayInfo{
				OriginModelName: "tiered-fallback-model",
				UserGroup:       tc.group,
				UsingGroup:      tc.group,
				RequestHeaders:  map[string]string{"Content-Type": "application/json"},
				BillingRequestInput: &billingexpr.RequestInput{
					Headers: map[string]string{"Content-Type": "application/json"},
					Body:    []byte(`{}`),
				},
			}

			priceData, err := ModelPriceHelper(ctx, info, promptTokens, &types.TokenCountMeta{MaxTokens: tc.maxTokens})
			require.NoError(t, err)
			require.Equal(t, tc.expected, priceData.QuotaToPreConsume)
		})
	}
}

func TestModelPriceHelperPerCallFallsBackToDefaultModelRatio(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalModelRatio := ratio_setting.ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(originalModelRatio))
	})
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "doubao-seedance-2-0-260128",
		UserGroup:       "default",
		UsingGroup:      "default",
	}

	priceData, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	require.False(t, priceData.UsePrice)
	require.InDelta(t, 46.0/14.0, priceData.ModelRatio, 0.000001)
	require.True(t, HasModelBillingConfig("doubao-seedance-2-0-260128"))
}

func TestModelPriceHelperPerCallFallsBackToDefaultVipeakSeedanceRatio(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalModelRatio := ratio_setting.ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(originalModelRatio))
	})
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("group", "default")

	tests := []struct {
		model string
		ratio float64
	}{
		{model: "doubao-seedance-2.0", ratio: 46.0 / 14.0},
		{model: "dreamina-seedance-2-0-260128", ratio: 46.0 / 14.0},
		{model: "dreamina-seedance-2-0-fast-260128", ratio: 37.0 / 14.0},
		{model: "seedance", ratio: 46.0 / 14.0},
		{model: "seedance-fast", ratio: 37.0 / 14.0},
	}

	for _, tt := range tests {
		info := &relaycommon.RelayInfo{
			OriginModelName: tt.model,
			UserGroup:       "default",
			UsingGroup:      "default",
		}

		priceData, err := ModelPriceHelperPerCall(ctx, info)
		require.NoError(t, err)
		require.False(t, priceData.UsePrice)
		require.InDelta(t, tt.ratio, priceData.ModelRatio, 0.000001)
		require.True(t, HasModelBillingConfig(tt.model))
	}
}
