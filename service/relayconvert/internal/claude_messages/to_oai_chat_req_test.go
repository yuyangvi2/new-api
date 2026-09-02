package claudemessages

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
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

func TestClaudeMessagesRequestToOpenAIChatNormalizesClaudeCodeTextBlocks(t *testing.T) {
	got, err := ClaudeMessagesRequestToOpenAIChat(dto.ClaudeRequest{
		Model: "deepseek-v4-flash",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []any{
					map[string]any{"type": "text", "text": "<system-reminder>context</system-reminder>\n\n"},
					map[string]any{"type": "text", "text": "inspect the repository", "cache_control": map[string]any{"type": "ephemeral"}},
				},
			},
			{
				Role: "assistant",
				Content: []any{
					map[string]any{"type": "text", "text": "I'll inspect it."},
					map[string]any{
						"type":  "tool_use",
						"id":    "toolu_1",
						"name":  "Read",
						"input": map[string]any{"file_path": "README.md"},
					},
				},
			},
		},
	}, nil)

	require.NoError(t, err)
	require.Len(t, got.Messages, 2)
	assert.Equal(t, "<system-reminder>context</system-reminder>\n\ninspect the repository", got.Messages[0].Content)
	assert.Equal(t, "I'll inspect it.", got.Messages[1].Content)
	require.Len(t, got.Messages[1].ParseToolCalls(), 1)
	assert.Empty(t, got.Messages[0].ParseContent()[0].CacheControl)
}

func TestClaudeMessagesRequestToOpenAIChatPreservesOpenRouterPromptCaching(t *testing.T) {
	got, err := ClaudeMessagesRequestToOpenAIChat(dto.ClaudeRequest{
		Model: "anthropic/claude-test",
		Messages: []dto.ClaudeMessage{{
			Role: "user",
			Content: []any{
				map[string]any{"type": "text", "text": "cached prompt", "cache_control": map[string]any{"type": "ephemeral"}},
			},
		}},
	}, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType:       constant.ChannelTypeOpenRouter,
		UpstreamModelName: "anthropic/claude-test",
	}})

	require.NoError(t, err)
	require.Len(t, got.Messages, 1)
	parts := got.Messages[0].ParseContent()
	require.Len(t, parts, 1)
	assert.JSONEq(t, `{"type":"ephemeral"}`, string(parts[0].CacheControl))
}
