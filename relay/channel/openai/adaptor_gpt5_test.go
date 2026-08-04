package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIRequestRejectsGPT5UnsupportedChatFields(t *testing.T) {
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.5"}}
	request := &dto.GeneralOpenAIRequest{
		Model:    "gpt-5.5",
		LogProbs: common.GetPointer(true),
	}

	_, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "logprobs is not supported")
}

func TestConvertOpenAIRequestAppliesGPT5SmallBudgetCompatibility(t *testing.T) {
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5.5"}}
	request := &dto.GeneralOpenAIRequest{
		Model:               "gpt-5.5",
		MaxCompletionTokens: common.GetPointer(uint(8)),
	}

	got, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)

	require.NoError(t, err)
	converted := got.(*dto.GeneralOpenAIRequest)
	assert.Equal(t, "none", converted.ReasoningEffort)
}

func TestConvertOpenAIResponsesRequestStripsGPT5UnsupportedSamplingParams(t *testing.T) {
	temperature := 0.0
	topP := 1.0
	request := dto.OpenAIResponsesRequest{
		Model:       "gpt-5.5",
		Temperature: &temperature,
		TopP:        &topP,
	}

	got, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, nil, request)

	require.NoError(t, err)
	converted := got.(dto.OpenAIResponsesRequest)
	assert.Nil(t, converted.Temperature)
	assert.Nil(t, converted.TopP)
}
