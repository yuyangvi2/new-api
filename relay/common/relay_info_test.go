package common

import (
	"testing"

	appconstant "github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestRelayInfoGetFinalRequestRelayFormatPrefersExplicitFinal(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		RequestConversionChain:  []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
		FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToConversionChain(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatClaude), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToRelayFormat(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatGemini,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatGemini), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatNilReceiver(t *testing.T) {
	var info *RelayInfo
	require.Equal(t, types.RelayFormat(""), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoShouldUseChatCompletionsForResponses(t *testing.T) {
	tests := []struct {
		name     string
		info     *RelayInfo
		expected bool
	}{
		{
			name: "responses mode enabled",
			info: &RelayInfo{
				RelayMode: relayconstant.RelayModeResponses,
				ChannelMeta: &ChannelMeta{
					ApiType: appconstant.APITypeOpenAI,
					ChannelSetting: dto.ChannelSettings{
						ResponsesViaChatCompletions: true,
					},
				},
			},
			expected: true,
		},
		{
			name: "unsupported adaptor stays native",
			info: &RelayInfo{
				RelayMode: relayconstant.RelayModeResponses,
				ChannelMeta: &ChannelMeta{
					ApiType: appconstant.APITypeDeepSeek,
					ChannelSetting: dto.ChannelSettings{
						ResponsesViaChatCompletions: true,
					},
				},
			},
			expected: false,
		},
		{
			name: "responses compact stays native",
			info: &RelayInfo{
				RelayMode: relayconstant.RelayModeResponsesCompact,
				ChannelMeta: &ChannelMeta{ChannelSetting: dto.ChannelSettings{
					ResponsesViaChatCompletions: true,
				}},
			},
			expected: false,
		},
		{
			name: "setting disabled",
			info: &RelayInfo{
				RelayMode:   relayconstant.RelayModeResponses,
				ChannelMeta: &ChannelMeta{},
			},
			expected: false,
		},
		{
			name:     "nil relay info",
			info:     nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.info.ShouldUseChatCompletionsForResponses())
		})
	}
}
