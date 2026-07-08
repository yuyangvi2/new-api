package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

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
		"billing_setting.billing_mode": `{"tiered-test-model":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"tiered-test-model":"param(\"stream\") == true ? tier(\"stream\", p * 3) : tier(\"base\", p * 2)"}`,
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
