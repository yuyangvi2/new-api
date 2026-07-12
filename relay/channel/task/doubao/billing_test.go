package doubao

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateSeedanceQuotaForDefaultDirectStandard(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	originalExchangeRate := operation_setting.USDExchangeRate
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
		operation_setting.USDExchangeRate = originalExchangeRate
	})
	common.QuotaPerUnit = 500000
	operation_setting.USDExchangeRate = 7

	quota, tokens, pricePerMillionCNY, ok := EstimateSeedanceQuotaForRequest(
		"seedance2.0_direct",
		relaycommon.TaskSubmitReq{
			Duration: 5,
			Size:     "16:9",
			Metadata: map[string]interface{}{"resolution": "720p"},
		},
		false,
		1,
	)

	require.True(t, ok)
	assert.Equal(t, 108000, tokens)
	assert.Equal(t, 50.0, pricePerMillionCNY)
	assert.Equal(t, 385714, quota)
	assert.InDelta(t, 5.4, float64(quota)/common.QuotaPerUnit*operation_setting.USDExchangeRate, 0.00001)
}

func TestEstimateSeedanceQuotaForMiniVideoInputUsesVideoPrice(t *testing.T) {
	quota, tokens, pricePerMillionCNY, ok := EstimateSeedanceQuotaForRequest(
		"Seedance_2.0_mini",
		relaycommon.TaskSubmitReq{
			Duration: 5,
			Size:     "16:9",
			Metadata: map[string]interface{}{
				"resolution":           "1080p",
				"input_video_duration": 2.5,
			},
		},
		true,
		1,
	)

	require.True(t, ok)
	assert.Equal(t, 364500, tokens)
	assert.Equal(t, 14.0, pricePerMillionCNY)
	assert.Greater(t, quota, 0)
}

func TestSeedancePricePerMillionCNYUsesCorrectFastVideoPrice(t *testing.T) {
	pricePerMillionCNY, ok := SeedancePricePerMillionCNY("seedance2.0_fast_direct", "720p", true)

	require.True(t, ok)
	assert.Equal(t, 20.0, pricePerMillionCNY)
}

func TestSeedancePricePerMillionCNYSupportsAPIZFastAlias(t *testing.T) {
	pricePerMillionCNY, ok := SeedancePricePerMillionCNY("seedance_2.0_fast", "720p", false)

	require.True(t, ok)
	assert.Equal(t, 40.0, pricePerMillionCNY)
}
