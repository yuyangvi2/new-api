package geminichat

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsageFromGeminiMetadataDoesNotDeriveNegativeCompletionTokens(t *testing.T) {
	usage := UsageFromGeminiMetadata(&dto.GeminiUsageMetadata{
		PromptTokenCount: 1000,
		TotalTokenCount:  1,
	}, 0)

	require.NotNil(t, usage)
	assert.Equal(t, 1000, usage.PromptTokens)
	assert.Equal(t, 0, usage.CompletionTokens)
}
