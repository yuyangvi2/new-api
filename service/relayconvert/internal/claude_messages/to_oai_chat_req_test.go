package claudemessages

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeMessagesRequestToOpenAIChatPreservesExtraBody(t *testing.T) {
	extraBody := []byte(`{"google":{"thinking_config":{"thinking_budget":0}}}`)

	got, err := ClaudeMessagesRequestToOpenAIChat(dto.ClaudeRequest{
		Model:     "claude-test",
		ExtraBody: extraBody,
		Messages:  []dto.ClaudeMessage{{Role: "user", Content: "hello"}},
	}, nil)

	require.NoError(t, err)
	assert.JSONEq(t, string(extraBody), string(got.ExtraBody))
}

func TestClaudeMessagesRequestToOpenAIChatToolChoiceNoneAndRemoteImageURL(t *testing.T) {
	got, err := ClaudeMessagesRequestToOpenAIChat(dto.ClaudeRequest{
		Model:      "claude-test",
		Tools:      []dto.Tool{{Name: "lookup", InputSchema: map[string]any{"type": "object"}}},
		ToolChoice: map[string]any{"type": "none"},
		Messages: []dto.ClaudeMessage{{
			Role: "user",
			Content: []any{
				map[string]any{
					"type": "image",
					"source": map[string]any{
						"type": "url",
						"url":  "https://example.com/image.png",
					},
				},
			},
		}},
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, "none", got.ToolChoice)
	require.Len(t, got.Messages, 1)
	parts := got.Messages[0].ParseContent()
	require.Len(t, parts, 1)
	image := parts[0].GetImageMedia()
	require.NotNil(t, image)
	assert.Equal(t, "https://example.com/image.png", image.Url)
}

func TestClaudeMessagesRequestToOpenAIChatBase64ImageUsesStringData(t *testing.T) {
	got, err := ClaudeMessagesRequestToOpenAIChat(dto.ClaudeRequest{
		Model: "claude-test",
		Messages: []dto.ClaudeMessage{{
			Role: "user",
			Content: []any{
				map[string]any{
					"type": "image",
					"source": map[string]any{
						"type":       "base64",
						"media_type": "image/png",
						"data":       "abc123",
					},
				},
			},
		}},
	}, nil)

	require.NoError(t, err)
	parts := got.Messages[0].ParseContent()
	require.Len(t, parts, 1)
	image := parts[0].GetImageMedia()
	require.NotNil(t, image)
	assert.Equal(t, "data:image/png;base64,abc123", image.Url)
}
