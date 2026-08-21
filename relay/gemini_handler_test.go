package relay

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
)

func TestApplyGeminiNoThinkingPricingRunsBeforePriceLookup(t *testing.T) {
	originalPrices := ratio_setting.ModelPrice2JSONString()
	originalEnabled := model_setting.GetGeminiSettings().ThinkingAdapterEnabled
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(originalPrices))
		model_setting.GetGeminiSettings().ThinkingAdapterEnabled = originalEnabled
	})
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"gemini-2.5-flash":0.05,"gemini-2.5-flash-nothinking":0.01}`))
	model_setting.GetGeminiSettings().ThinkingAdapterEnabled = true

	info := &relaycommon.RelayInfo{
		OriginModelName: "gemini-2.5-flash",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-2.5-flash",
		},
		Request: &dto.GeminiChatRequest{
			GenerationConfig: dto.GeminiChatGenerationConfig{
				ThinkingConfig: &dto.GeminiThinkingConfig{ThinkingBudget: common.GetPointer(0)},
			},
		},
	}
	ApplyGeminiNoThinkingPricing(info)

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	priceData, err := helper.ModelPriceHelper(ctx, info, 1, &types.TokenCountMeta{})
	require.NoError(t, err)
	assert.Equal(t, "gemini-2.5-flash-nothinking", info.OriginModelName)
	assert.Equal(t, "gemini-2.5-flash-nothinking", info.UpstreamModelName)
	assert.True(t, priceData.UsePrice)
	assert.Equal(t, 0.01, priceData.ModelPrice)
}
