package oaichat

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIChatRequestToClaudeMessagesKeepsNoParameterTools(t *testing.T) {
	got, err := OpenAIChatRequestToClaudeMessages(nil, dto.GeneralOpenAIRequest{
		Model: "claude-test",
		Tools: []dto.ToolCallRequest{{
			Type: "function",
			Function: dto.FunctionRequest{
				Name:        "no_params",
				Description: "No params tool",
			},
		}},
		Messages: []dto.Message{{Role: "user", Content: "hello"}},
	})

	require.NoError(t, err)
	tools, ok := got.Tools.([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)
	tool, ok := tools[0].(*dto.Tool)
	require.True(t, ok)
	assert.Equal(t, "no_params", tool.Name)
	assert.Equal(t, map[string]interface{}{"type": "object"}, tool.InputSchema)
}

func TestOpenAIChatRequestToClaudeMessagesDefaultsNonStringSchemaType(t *testing.T) {
	got, err := OpenAIChatRequestToClaudeMessages(nil, dto.GeneralOpenAIRequest{
		Model: "claude-test",
		Tools: []dto.ToolCallRequest{{
			Type: "function",
			Function: dto.FunctionRequest{
				Name: "bad_type",
				Parameters: map[string]any{
					"type":       []any{"object"},
					"properties": map[string]any{},
				},
			},
		}},
		Messages: []dto.Message{{Role: "user", Content: "hello"}},
	})

	require.NoError(t, err)
	tools, ok := got.Tools.([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)
	tool, ok := tools[0].(*dto.Tool)
	require.True(t, ok)
	assert.Equal(t, "object", tool.InputSchema["type"])
	assert.Equal(t, map[string]any{}, tool.InputSchema["properties"])
}

func TestOpenAIChatRequestToClaudeMessagesOmitsNewClaudeSamplingParams(t *testing.T) {
	temperature := 0.7
	topP := 0.9
	topK := 40

	got, err := OpenAIChatRequestToClaudeMessages(nil, dto.GeneralOpenAIRequest{
		Model:       "claude-opus-5",
		Temperature: &temperature,
		TopP:        &topP,
		TopK:        &topK,
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, got.Temperature)
	assert.Nil(t, got.TopP)
	assert.Nil(t, got.TopK)
}
