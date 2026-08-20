package oairesponses

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponsesRequestToClaudeMessagesOmitsNewClaudeSamplingParams(t *testing.T) {
	temperature := 0.7
	topP := 0.9
	maxOutputTokens := uint(64)

	got, err := OpenAIResponsesRequestToClaudeMessages(nil, &dto.OpenAIResponsesRequest{
		Model:           "claude-opus-5",
		Input:           []byte(`[{"role":"user","content":"hello"}]`),
		MaxOutputTokens: &maxOutputTokens,
		Temperature:     &temperature,
		TopP:            &topP,
	})

	require.NoError(t, err)
	assert.Equal(t, "claude-opus-5", got.Model)
	assert.Nil(t, got.Temperature)
	assert.Nil(t, got.TopP)
	require.NotNil(t, got.MaxTokens)
	assert.Equal(t, maxOutputTokens, *got.MaxTokens)
}

func TestOpenAIResponsesRequestToClaudeMessagesKeepsOlderClaudeSamplingParams(t *testing.T) {
	temperature := 0.7
	topP := 0.9
	maxOutputTokens := uint(64)

	got, err := OpenAIResponsesRequestToClaudeMessages(nil, &dto.OpenAIResponsesRequest{
		Model:           "claude-sonnet-4-5",
		Input:           []byte(`[{"role":"user","content":"hello"}]`),
		MaxOutputTokens: &maxOutputTokens,
		Temperature:     &temperature,
		TopP:            &topP,
	})

	require.NoError(t, err)
	assert.Equal(t, temperature, *got.Temperature)
	assert.Equal(t, topP, *got.TopP)
}
