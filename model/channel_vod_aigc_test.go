package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVODAIGCChannelRequiresMappingForEveryModel(t *testing.T) {
	mapping := `{"gemini-3-pro-image":"GG:3.0"}`
	channel := Channel{
		Type:         constant.ChannelTypeVODAIGC,
		Models:       "gemini-3-pro-image,vidu-q2",
		ModelMapping: &mapping,
	}

	err := channel.ValidateSettings()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "vidu-q2")
}

func TestVODAIGCChannelAcceptsCompleteModelMapping(t *testing.T) {
	mapping := `{"gemini-3-pro-image":"GG:3.0","vidu-q2":"Vidu:q2"}`
	channel := Channel{
		Type:         constant.ChannelTypeVODAIGC,
		Models:       "gemini-3-pro-image,vidu-q2",
		ModelMapping: &mapping,
	}

	require.NoError(t, channel.ValidateSettings())
}

func TestVODAIGCChannelRejectsInvalidMappingTarget(t *testing.T) {
	tests := []string{
		`{"gemini-3-pro-image":"Unknown:1"}`,
		`{"gemini-3-pro-image":"GG:"}`,
	}

	for _, mapping := range tests {
		channel := Channel{
			Type:         constant.ChannelTypeVODAIGC,
			Models:       "gemini-3-pro-image",
			ModelMapping: &mapping,
		}

		err := channel.ValidateSettings()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "gemini-3-pro-image")
	}
}
