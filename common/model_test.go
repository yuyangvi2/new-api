package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGrokImagineModelClassification(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		wantImage bool
		wantVideo bool
	}{
		{
			name:      "base imagine routes as image only",
			model:     "grok-imagine",
			wantImage: true,
		},
		{
			name:      "image quality routes as image only",
			model:     "grok-imagine-image-quality",
			wantImage: true,
		},
		{
			name:      "video routes as OpenAI video only",
			model:     "grok-imagine-video-1.5",
			wantVideo: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantImage, IsImageGenerationModel(tt.model))
			assert.Equal(t, tt.wantVideo, IsOpenAIVideoGenerationModel(tt.model))
		})
	}
}
