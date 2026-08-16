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
	assert.Equal(t, 46.0, pricePerMillionCNY)
	assert.Equal(t, 354857, quota)
	assert.InDelta(t, 4.968, float64(quota)/common.QuotaPerUnit*operation_setting.USDExchangeRate, 0.00001)
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

func TestSeedancePricePerMillionCNYUsesOfficialFastVideoInputPrice(t *testing.T) {
	pricePerMillionCNY, ok := SeedancePricePerMillionCNY("seedance2.0_fast_direct", "720p", true)

	require.True(t, ok)
	assert.Equal(t, 22.0, pricePerMillionCNY)
}

func TestSeedancePricePerMillionCNYSupportsFastAlias(t *testing.T) {
	pricePerMillionCNY, ok := SeedancePricePerMillionCNY("seedance_2.0_fast", "720p", false)

	require.True(t, ok)
	assert.Equal(t, 37.0, pricePerMillionCNY)
}

func TestSeedancePricePerMillionCNYSupportsOfficialStandardAlias(t *testing.T) {
	pricePerMillionCNY, ok := SeedancePricePerMillionCNY("doubao-seedance-2.0", "1080p", true)

	require.True(t, ok)
	assert.Equal(t, 31.0, pricePerMillionCNY)
	assert.True(t, IsSeedanceModel("doubao-seedance-2.0"))
}

func TestSeedancePricePerMillionCNYSupportsOfficialMiniAndFastAliases(t *testing.T) {
	tests := []struct {
		model         string
		hasVideoInput bool
		want          float64
	}{
		{model: "doubao-seedance-2-0-fast", hasVideoInput: false, want: 37.0},
		{model: "doubao-seedance-2-0-fast", hasVideoInput: true, want: 22.0},
		{model: "doubao-seedance-2-0-mini", hasVideoInput: false, want: 23.0},
		{model: "doubao-seedance-2-0-mini", hasVideoInput: true, want: 14.0},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			pricePerMillionCNY, ok := SeedancePricePerMillionCNY(tt.model, "720p", tt.hasVideoInput)

			require.True(t, ok)
			assert.Equal(t, tt.want, pricePerMillionCNY)
			assert.True(t, IsSeedanceModel(tt.model))
		})
	}
}

func TestEstimateSeedanceQuotaClampsUntrustedDurationAndRatio(t *testing.T) {
	quota, tokens, _, ok := EstimateSeedanceQuotaForRequest(
		"seedance2.0_direct",
		relaycommon.TaskSubmitReq{
			Duration: relaycommon.MaxTaskDurationSeconds + 1,
			Size:     "999999999:1",
			Metadata: map[string]interface{}{
				"input_video_duration": float64(relaycommon.MaxTaskDurationSeconds + 1),
				"resolution":           "720p",
			},
		},
		false,
		1,
	)

	require.True(t, ok)
	assert.Greater(t, quota, 0)
	assert.Greater(t, tokens, 0)
}
