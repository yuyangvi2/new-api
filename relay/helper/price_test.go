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

func TestModelPriceHelperPerCallUsesSeedanceBillingAlias(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(saved))
	})

	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"dreamina-seedance-2-0-260128":0.42}`))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "seedance",
		UserGroup:       "default",
		UsingGroup:      "default",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "seedance",
		},
	}

	priceData, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 0.42, priceData.ModelPrice)
	require.Equal(t, int(0.42*common.QuotaPerUnit), priceData.Quota)
}

func TestModelPriceHelperPerCallUsesDoubaoSeedanceBillingAlias(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(saved))
	})

	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"doubao-seedance-2-0-fast-260128":0.37}`))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "doubao-seedance-2-0-fast",
		UserGroup:       "default",
		UsingGroup:      "default",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "doubao-seedance-2-0-fast",
		},
	}

	priceData, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 0.37, priceData.ModelPrice)
	require.Equal(t, int(0.37*common.QuotaPerUnit), priceData.Quota)
}

func TestModelPriceHelperPerCallUsesSeedance20PrefixPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(saved))
	})

	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"seedance2.0_":0.93}`))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "seedance2.0_direct",
		UserGroup:       "default",
		UsingGroup:      "default",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "seedance2.0_direct",
		},
	}

	priceData, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 0.93, priceData.ModelPrice)
	require.Equal(t, int(0.93*common.QuotaPerUnit), priceData.Quota)
}
