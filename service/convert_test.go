package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiToOpenAIRequestStopSequencesAreBounded(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-4o-mini",
		},
	}

	request, err := GeminiToOpenAIRequest(&dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{{
			Role:  "user",
			Parts: []dto.GeminiPart{{Text: "hi"}},
		}},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			StopSequences: []string{"A"},
		},
	}, info)

	require.NoError(t, err)
	assert.Equal(t, []string{"A"}, request.Stop)

	request, err = GeminiToOpenAIRequest(&dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{{
			Role:  "user",
			Parts: []dto.GeminiPart{{Text: "hi"}},
		}},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			StopSequences: []string{"A", "B", "C", "D", "E"},
		},
	}, info)

	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "C", "D"}, request.Stop)
}
