package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestInferPricingModelType(t *testing.T) {
	tests := []struct {
		name         string
		modelName    string
		tags         string
		channelTypes []int
		endpoints    []constant.EndpointType
		want         int
	}{
		{
			name:      "seedance model name is video",
			modelName: "seedance-2-fast",
			want:      pricingModelTypeVideo,
		},
		{
			name:         "toapis channel is video",
			modelName:    "custom-generation-model",
			channelTypes: []int{constant.ChannelTypeToAPIs},
			want:         pricingModelTypeVideo,
		},
		{
			name:      "image generation endpoint is image",
			modelName: "custom-image-model",
			endpoints: []constant.EndpointType{constant.EndpointTypeImageGeneration},
			want:      pricingModelTypeImage,
		},
		{
			name:      "audio model name is audio",
			modelName: "whisper-1",
			want:      pricingModelTypeAudio,
		},
		{
			name:      "wanx video name wins before image terms",
			modelName: "wanx2.1-t2v-plus",
			want:      pricingModelTypeVideo,
		},
		{
			name:      "plain chat model remains text",
			modelName: "gpt-4o-mini",
			want:      pricingModelTypeText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferPricingModelType(tt.modelName, tt.tags, tt.channelTypes, tt.endpoints, false, false)
			assert.Equal(t, tt.want, got)
		})
	}
}
