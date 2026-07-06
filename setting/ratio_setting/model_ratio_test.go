package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetModelPriceUsesViduQFamilyFallback(t *testing.T) {
	original := ModelPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(original))
	})
	require.NoError(t, UpdateModelPriceByJSONString(`{"viduq":0.12}`))

	price, ok := GetModelPrice("viduq2-pro", false)

	require.True(t, ok)
	assert.Equal(t, 0.12, price)
}

func TestGetModelPriceExactViduQModelOverridesFamilyFallback(t *testing.T) {
	original := ModelPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(original))
	})
	require.NoError(t, UpdateModelPriceByJSONString(`{"viduq":0.12,"viduq2-pro":0.2}`))

	price, ok := GetModelPrice("viduq2-pro", false)

	require.True(t, ok)
	assert.Equal(t, 0.2, price)
}

func TestGetModelRatioUsesViduQFamilyFallback(t *testing.T) {
	original := ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelRatioByJSONString(original))
	})
	require.NoError(t, UpdateModelRatioByJSONString(`{"viduq":3.5}`))

	ratio, ok, matchName := GetModelRatio("viduq2-pro")

	require.True(t, ok)
	assert.Equal(t, 3.5, ratio)
	assert.Equal(t, "viduq", matchName)
}
