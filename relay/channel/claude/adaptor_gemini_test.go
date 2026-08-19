package claude

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestUsesClaudePayload(t *testing.T) {
	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatGemini,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-opus-4-8",
		},
	}
	maxTokens := uint(16)

	converted, err := (&Adaptor{}).ConvertGeminiRequest(nil, info, &dto.GeminiChatRequest{
		SystemInstructions: &dto.GeminiChatContent{
			Parts: []dto.GeminiPart{{Text: "answer briefly"}},
		},
		Contents: []dto.GeminiChatContent{{
			Role:  "user",
			Parts: []dto.GeminiPart{{Text: "hi"}},
		}},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			MaxOutputTokens: &maxTokens,
			StopSequences:   []string{"STOP"},
		},
	})

	require.NoError(t, err)
	request := converted.(*dto.ClaudeRequest)
	assert.Equal(t, "claude-opus-4-8", request.Model)
	system := request.ParseSystem()
	require.Len(t, system, 1)
	assert.Equal(t, "answer briefly", system[0].GetText())
	require.Len(t, request.Messages, 1)
	assert.Equal(t, "user", request.Messages[0].Role)
	assert.Equal(t, "hi", request.Messages[0].Content)
	assert.Equal(t, []string{"STOP"}, request.StopSequences)
}

func TestRequestOpenAI2ClaudeMessageAcceptsStringStopSlice(t *testing.T) {
	converted, err := RequestOpenAI2ClaudeMessage(nil, dto.GeneralOpenAIRequest{
		Model: "claude-opus-4-8",
		Messages: []dto.Message{{
			Role:    "user",
			Content: "hi",
		}},
		Stop: []string{"A", "B"},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B"}, converted.StopSequences)
}

func TestClaudeHandlerConvertsGeminiResponseFormat(t *testing.T) {
	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatGemini,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-opus-4-8",
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{
			"id":"msg_1",
			"type":"message",
			"role":"assistant",
			"model":"claude-opus-4-8",
			"content":[{"type":"text","text":"Hi there!"}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":3,"output_tokens":2}
		}`)),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	usage, err := ClaudeHandler(c, resp, info)

	require.Nil(t, err)
	require.NotNil(t, usage)
	assert.Equal(t, 3, usage.PromptTokens)
	assert.Equal(t, 2, usage.CompletionTokens)

	var geminiResponse dto.GeminiChatResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &geminiResponse))
	require.Len(t, geminiResponse.Candidates, 1)
	require.Len(t, geminiResponse.Candidates[0].Content.Parts, 1)
	assert.Equal(t, "Hi there!", geminiResponse.Candidates[0].Content.Parts[0].Text)
	assert.Equal(t, 5, geminiResponse.UsageMetadata.TotalTokenCount)
}
