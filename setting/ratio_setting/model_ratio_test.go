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

func TestGetModelPriceUnitMarksXaiVideoAsSecond(t *testing.T) {
	assert.Equal(t, "second", GetModelPriceUnit("grok-imagine-video"))
	assert.Equal(t, "second", GetModelPriceUnit("grok-imagine-video-1.5"))
	assert.Equal(t, "request", GetModelPriceUnit("suno_music"))
}

func TestGrokImagineImage20HasDefaultImagePricing(t *testing.T) {
	originalPrice := ModelPrice2JSONString()
	originalImageRatio := ImageRatio2JSONString()
	InitRatioSettings()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(originalPrice))
		require.NoError(t, UpdateImageRatioByJSONString(originalImageRatio))
	})

	price, ok := GetModelPrice("grok-imagine-image-2.0", false)

	require.True(t, ok)
	assert.Equal(t, 0.05/0.6, price)

	ratio, ok := GetImageRatio("grok-imagine-image-2.0")
	require.True(t, ok)
	assert.Equal(t, 1.0, ratio)
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
