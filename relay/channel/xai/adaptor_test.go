package xai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newXAITestRelayInfo(format types.RelayFormat) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayFormat:    format,
		RelayMode:      constant.RelayModeChatCompletions,
		RequestURLPath: "/v1/messages",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl:    "https://api.x.ai",
			UpstreamModelName: "grok-4.20-non-reasoning",
		},
	}
}

func TestConvertClaudeRequestUsesOpenAICompatiblePayload(t *testing.T) {
	info := newXAITestRelayInfo(types.RelayFormatClaude)
	maxTokens := uint(16)
	temperature := 0.2

	converted, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, &dto.ClaudeRequest{
		Model:       "grok-4.20-non-reasoning",
		System:      "answer briefly",
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
		Messages: []dto.ClaudeMessage{{
			Role:    "user",
			Content: "hi",
		}},
		StopSequences: []string{"STOP"},
	})

	require.NoError(t, err)
	request := converted.(*dto.GeneralOpenAIRequest)
	assert.Equal(t, "grok-4.20-non-reasoning", request.Model)
	require.Len(t, request.Messages, 2)
	assert.Equal(t, "system", request.Messages[0].Role)
	assert.Equal(t, "user", request.Messages[1].Role)
	assert.Nil(t, request.Stop)
}

func TestConvertGeminiRequestUsesOpenAICompatiblePayload(t *testing.T) {
	info := newXAITestRelayInfo(types.RelayFormatGemini)
	maxTokens := uint(16)

	converted, err := (&Adaptor{}).ConvertGeminiRequest(nil, info, &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{{
			Role: "user",
			Parts: []dto.GeminiPart{{
				Text: "hi",
			}},
		}},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			MaxOutputTokens: &maxTokens,
			StopSequences:   []string{"STOP"},
		},
	})

	require.NoError(t, err)
	request := converted.(*dto.GeneralOpenAIRequest)
	assert.Equal(t, "grok-4.20-non-reasoning", request.Model)
	require.Len(t, request.Messages, 1)
	assert.Equal(t, "user", request.Messages[0].Role)
	assert.Nil(t, request.Stop)
}

func TestGetRequestURLMapsConvertedProtocolsToChatCompletions(t *testing.T) {
	for _, format := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatGemini} {
		t.Run(string(format), func(t *testing.T) {
			info := newXAITestRelayInfo(format)

			url, err := (&Adaptor{}).GetRequestURL(info)

			require.NoError(t, err)
			assert.Equal(t, "https://api.x.ai/v1/chat/completions", url)
		})
	}
}

func TestGetRequestURLDoesNotDuplicateV1ForConvertedProtocols(t *testing.T) {
	info := newXAITestRelayInfo(types.RelayFormatClaude)
	info.ChannelBaseUrl = "https://api.x.ai/v1"

	url, err := (&Adaptor{}).GetRequestURL(info)

	require.NoError(t, err)
	assert.Equal(t, "https://api.x.ai/v1/chat/completions", url)
}

func TestXAIHandlerConvertsNonOpenAIResponseFormats(t *testing.T) {
	info := newXAITestRelayInfo(types.RelayFormatGemini)
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(`{
			"id":"chatcmpl_1",
			"object":"chat.completion",
			"created":1,
			"model":"grok-4.20-non-reasoning",
			"choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"total_tokens":3}
		}`)),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	usage, err := xAIHandler(c, info, resp)

	require.Nil(t, err)
	require.NotNil(t, usage)
	assert.Equal(t, 2, usage.CompletionTokens)
	assert.Contains(t, w.Body.String(), `"candidates"`)
}
