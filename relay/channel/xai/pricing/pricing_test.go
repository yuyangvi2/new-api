package pricing

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrokImagineImageRequestPriceByModelAndResolution(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		resolution  string
		outputs     int
		inputImages int
		want        float64
	}{
		{name: "standard 1k text to image", model: "grok-imagine-image", resolution: "1k", outputs: 1, want: 0.02},
		{name: "standard 2k same output price", model: "grok-imagine", resolution: "2k", outputs: 2, want: 0.04},
		{name: "standard input image charged once", model: "grok-imagine-image", resolution: "1k", outputs: 2, inputImages: 1, want: 0.042},
		{name: "quality 1k", model: "grok-imagine-image-quality", resolution: "1k", outputs: 1, want: 0.05},
		{name: "quality 2k", model: "grok-imagine-image-quality", resolution: "2k", outputs: 1, want: 0.07},
		{name: "quality input image charged once", model: "grok-imagine-image-quality", resolution: "2k", outputs: 2, inputImages: 1, want: 0.15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GrokImagineImageRequestPrice(tt.model, tt.resolution, tt.outputs, tt.inputImages)
			require.True(t, ok)
			assert.InDelta(t, tt.want, got, 0.000001)
		})
	}
}

func TestGrokImagineVideoPriceByResolutionAndDuration(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		resolution  string
		duration    int
		inputImages int
		want        float64
	}{
		{name: "legacy video 720p per second", model: "grok-imagine-video", resolution: "720p", duration: 8, want: 0.56},
		{name: "legacy video 480p per second", model: "grok-imagine-video", resolution: "480p", duration: 8, want: 0.40},
		{name: "1.5 480p image to video", model: "grok-imagine-video-1.5", resolution: "480p", duration: 8, inputImages: 1, want: 0.65},
		{name: "1.5 720p image to video", model: "grok-imagine-video-1.5", resolution: "720p", duration: 8, inputImages: 1, want: 1.13},
		{name: "1.5 1080p image to video", model: "grok-imagine-video-1.5", resolution: "1080p", duration: 8, inputImages: 1, want: 2.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GrokImagineVideoPrice(tt.model, tt.resolution, tt.duration, tt.inputImages)
			require.True(t, ok)
			assert.InDelta(t, tt.want, got, 0.000001)
		})
	}
}

func TestGrokImagineTaskPriceUsesRequestFields(t *testing.T) {
	got, ok := GrokImagineTaskPrice(relaycommon.TaskSubmitReq{
		Model:    "grok-imagine-video-1.5",
		Image:    "https://example.com/input.jpg",
		Images:   []string{"https://example.com/input.jpg"},
		Duration: 6,
		Metadata: map[string]any{"resolution": "1080p"},
	}, "grok-imagine-video-1.5")

	require.True(t, ok)
	assert.InDelta(t, 1.51, got, 0.000001)
}

func TestCountImageRequestInputs(t *testing.T) {
	request := dto.ImageRequest{
		Image:  []byte(`"https://example.com/input.png"`),
		Images: []byte(`["https://example.com/a.png","https://example.com/b.png"]`),
	}

	assert.Equal(t, 3, CountImageRequestInputs(request))
}
