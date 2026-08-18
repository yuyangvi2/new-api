package geminichat

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiGenerateContentRequestToOpenAIChatTextAndTools(t *testing.T) {
	info := &relaycommon.RelayInfo{
		IsStream: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-test",
		},
	}
	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{{Text: "hello"}},
			},
			{
				Role: "model",
				Parts: []dto.GeminiPart{{FunctionCall: &dto.FunctionCall{FunctionName: "lookup", Arguments: map[string]any{"q": "x"}}}},
			},
			{
				Role: "user",
				Parts: []dto.GeminiPart{{FunctionResponse: &dto.GeminiFunctionResponse{Name: "lookup", Response: map[string]interface{}{"ok": true}}}},
			},
		},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			Temperature:     ptrFloat64(0),
			TopP:            ptrFloat64(0.8),
			MaxOutputTokens: ptrUint(128),
		},
	}

	got, err := GeminiGenerateContentRequestToOpenAIChat(req, info)

	require.NoError(t, err)
	assert.Equal(t, "gemini-test", got.Model)
	require.NotNil(t, got.Stream)
	assert.True(t, *got.Stream)
	require.NotNil(t, got.Temperature)
	assert.Equal(t, 0.0, *got.Temperature)
	require.NotNil(t, got.TopP)
	assert.Equal(t, 0.8, *got.TopP)
	require.NotNil(t, got.MaxTokens)
	assert.Equal(t, uint(128), *got.MaxTokens)
	require.Len(t, got.Messages, 3)
	assert.Equal(t, "user", got.Messages[0].Role)
	assert.Equal(t, "hello", got.Messages[0].StringContent())
	assert.Equal(t, "assistant", got.Messages[1].Role)
	toolCalls := got.Messages[1].ParseToolCalls()
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "lookup", toolCalls[0].Function.Name)
	assert.JSONEq(t, `{"q":"x"}`, toolCalls[0].Function.Arguments)
	assert.Equal(t, "tool", got.Messages[2].Role)
	assert.JSONEq(t, `{"ok":true}`, got.Messages[2].StringContent())
}

func TestGeminiGenerateContentRequestToOpenAIChatMedia(t *testing.T) {
	got, err := GeminiGenerateContentRequestToOpenAIChat(&dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{{
			Role: "user",
			Parts: []dto.GeminiPart{{InlineData: &dto.GeminiInlineData{MimeType: "image/png", Data: "abc"}}},
		}},
	}, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gemini-test"}})

	require.NoError(t, err)
	require.Len(t, got.Messages, 1)
	parts := got.Messages[0].ParseContent()
	require.Len(t, parts, 1)
	image := parts[0].GetImageMedia()
	require.NotNil(t, image)
	assert.Equal(t, "data:image/png;base64,abc", image.Url)
	assert.Equal(t, "image/png", image.MimeType)
}

func ptrFloat64(v float64) *float64 { return &v }
func ptrUint(v uint) *uint { return &v }
