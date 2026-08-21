package ratio_setting

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveDeepSeekPricingByShanghaiTime(t *testing.T) {
	location, err := time.LoadLocation(deepSeekPricingTimezone)
	require.NoError(t, err)

	offPeak, configured, err := ResolveModelPricing("deepseek-chat", time.Date(2026, 8, 21, 0, 30, 0, 0, location))
	require.NoError(t, err)
	require.True(t, configured)
	require.Equal(t, "off_peak", offPeak.PeriodName)
	require.InDelta(t, 0.0675, offPeak.ModelRatio, 0.000001)
	require.InDelta(t, 0.035/0.135, offPeak.CacheRatio, 0.000001)
	require.InDelta(t, 0.55/0.135, offPeak.CompletionRatio, 0.000001)

	peak, configured, err := ResolveModelPricing("deepseek-chat", time.Date(2026, 8, 21, 8, 30, 0, 0, location))
	require.NoError(t, err)
	require.True(t, configured)
	require.Equal(t, "peak", peak.PeriodName)
	require.InDelta(t, 0.135, peak.ModelRatio, 0.000001)
}

func TestResolveModelPricingSupportsCrossMidnightPeriod(t *testing.T) {
	original := ModelPricing2JSONString()
	t.Cleanup(func() { require.NoError(t, UpdateModelPricingByJSONString(original)) })
	require.NoError(t, UpdateModelPricingByJSONString(`{
		"test-model": {
			"timezone": "UTC",
			"periods": [{"name":"night","start":"22:00","end":"02:00","input_price":2,"cache_hit_price":1,"output_price":4}]
		}
	}`))

	snapshot, configured, err := ResolveModelPricing("test-model", time.Date(2026, 8, 21, 23, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.True(t, configured)
	require.Equal(t, "night", snapshot.PeriodName)

	snapshot, configured, err = ResolveModelPricing("test-model", time.Date(2026, 8, 21, 1, 59, 0, 0, time.UTC))
	require.NoError(t, err)
	require.True(t, configured)
	require.Equal(t, "night", snapshot.PeriodName)

	_, configured, err = ResolveModelPricing("test-model", time.Date(2026, 8, 21, 2, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.False(t, configured)
}
