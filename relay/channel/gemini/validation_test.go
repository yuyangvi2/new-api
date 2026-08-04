package gemini

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGeminiTestContext(t *testing.T) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	return c
}

func newGeminiRelayInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-3.5-flash",
		},
	}
}

func TestValidateGeminiCompatibleOpenAIRequestRejectsUnsupportedSilentDrops(t *testing.T) {
	tests := []struct {
		name    string
		request dto.GeneralOpenAIRequest
		wantErr string
	}{
		{
			name: "logprobs",
			request: dto.GeneralOpenAIRequest{
				Model:    "gemini-3.5-flash",
				LogProbs: common.GetPointer(true),
			},
			wantErr: "logprobs is not supported",
		},
		{
			name: "reasoning conflict",
			request: dto.GeneralOpenAIRequest{
				Model:           "gemini-3.5-flash",
				ReasoningEffort: "high",
				ExtraBody:       []byte(`{"google":{"thinking_config":{"thinking_level":"high"}}}`),
			},
			wantErr: "reasoning_effort cannot be used",
		},
		{
			name: "empty enum",
			request: dto.GeneralOpenAIRequest{
				Model: "gemini-3.5-flash",
				Tools: []dto.ToolCallRequest{{
					Type: "function",
					Function: dto.FunctionRequest{
						Name: "lookup",
						Parameters: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"value": map[string]any{"type": "string", "enum": []any{}},
							},
							"required": []any{"value"},
						},
					},
				}},
			},
			wantErr: "enum must not be empty",
		},
		{
			name: "bad response schema type",
			request: dto.GeneralOpenAIRequest{
				Model: "gemini-3.5-flash",
				ResponseFormat: &dto.ResponseFormat{
					Type:       "json_schema",
					JsonSchema: []byte(`{"name":"bad","schema":{"type":"not-a-real-json-schema-type"}}`),
				},
			},
			wantErr: "unsupported value",
		},
		{
			name: "n is zero",
			request: dto.GeneralOpenAIRequest{
				Model: "gemini-3.5-flash",
				N:     common.GetPointer(0),
			},
			wantErr: "n must be between 1 and",
		},
		{
			name: "n exceeds max",
			request: dto.GeneralOpenAIRequest{
				Model: "gemini-3.5-flash",
				N:     common.GetPointer(9),
			},
			wantErr: "n must be between 1 and",
		},
		{
			name: "empty cached_content",
			request: dto.GeneralOpenAIRequest{
				Model:     "gemini-3.5-flash",
				ExtraBody: []byte(`{"google":{"cached_content":"  "}}`),
			},
			wantErr: "cached_content must be a non-empty string",
		},
		{
			name: "top_logprobs not supported",
			request: dto.GeneralOpenAIRequest{
				Model:       "gemini-3.5-flash",
				TopLogProbs: common.GetPointer(5),
			},
			wantErr: "top_logprobs is not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGeminiCompatibleOpenAIRequest(&tt.request)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestConvertOpenAI2GeminiPreservesCompatibleContractFields(t *testing.T) {
	c := newGeminiTestContext(t)
	info := newGeminiRelayInfo()
	request := dto.GeneralOpenAIRequest{
		Model: "gemini-3.5-flash",
		N:     common.GetPointer(2),
		Messages: []dto.Message{
			{Role: "user", Name: common.GetPointer("sender_alpha"), Content: "CODE_A7Q2K9"},
		},
		Stop:      []any{"<STOP>"},
		ExtraBody: []byte(`{"google":{"cached_content":"cachedContents/example"}}`),
	}

	got, err := CovertOpenAI2Gemini(c, request, info)
	require.NoError(t, err)
	require.NotNil(t, got.GenerationConfig.CandidateCount)
	assert.Equal(t, 2, *got.GenerationConfig.CandidateCount)
	assert.Equal(t, "cachedContents/example", got.CachedContent)
	require.Len(t, got.GenerationConfig.StopSequences, 1)
	assert.Equal(t, "<STOP>", got.GenerationConfig.StopSequences[0])
	require.Len(t, got.Contents, 1)
	require.Len(t, got.Contents[0].Parts, 1)
	assert.Contains(t, got.Contents[0].Parts[0].Text, "Message from sender_alpha:")
	assert.Contains(t, got.Contents[0].Parts[0].Text, "CODE_A7Q2K9")
}

func TestConvertOpenAI2GeminiRejectsNonImageMimeForImagePart(t *testing.T) {
	c := newGeminiTestContext(t)
	info := newGeminiRelayInfo()
	request := dto.GeneralOpenAIRequest{
		Model: "gemini-3.5-flash",
		Messages: []dto.Message{{
			Role: "user",
			Content: []any{
				map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": "data:text/plain;base64,aGVsbG8=",
					},
				},
			},
		}},
	}

	_, err := CovertOpenAI2Gemini(c, request, info)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "image_url media type must be an image MIME type")
}

func TestClaudeToOpenAIPreservesToolChoiceNoneAndRemoteImageURL(t *testing.T) {
	info := newGeminiRelayInfo()
	claudeRequest := dto.ClaudeRequest{
		Model:      "gemini-3.5-flash",
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
	}

	openAIRequest, err := service.ClaudeToOpenAIRequest(claudeRequest, info)
	require.NoError(t, err)
	assert.Equal(t, "none", openAIRequest.ToolChoice)
	require.Len(t, openAIRequest.Messages, 1)
	parts := openAIRequest.Messages[0].ParseContent()
	require.Len(t, parts, 1)
	image := parts[0].GetImageMedia()
	require.NotNil(t, image)
	assert.Equal(t, "https://example.com/image.png", image.Url)
}

func TestResponseGeminiChat2OpenAIMapsClaudeStopSequence(t *testing.T) {
	c := newGeminiTestContext(t)
	info := &relaycommon.RelayInfo{
		RelayFormat:   types.RelayFormatClaude,
		StopSequences: []string{"<STOP>"},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-3.5-flash",
		},
	}
	stop := "STOP"
	geminiResponse := &dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{{
			FinishReason: &stop,
			Content: dto.GeminiChatContent{
				Parts: []dto.GeminiPart{{Text: "BEFORE<STOP>AFTER"}},
			},
		}},
	}

	openAIResponse := responseGeminiChat2OpenAI(c, info, geminiResponse)
	require.Len(t, openAIResponse.Choices, 1)
	assert.Equal(t, "stop_sequence", openAIResponse.Choices[0].FinishReason)
	assert.Equal(t, "BEFORE", openAIResponse.Choices[0].Message.StringContent())

	claudeResponse := service.ResponseOpenAI2Claude(openAIResponse, info)
	assert.Equal(t, "stop_sequence", claudeResponse.StopReason)
	require.NotNil(t, claudeResponse.StopSequence)
	assert.Equal(t, "<STOP>", *claudeResponse.StopSequence)
}

func TestResponseGeminiChat2OpenAIDoesNotLeakToolFinishReasonAcrossChoices(t *testing.T) {
	c := newGeminiTestContext(t)
	info := newGeminiRelayInfo()
	stop := "STOP"
	geminiResponse := &dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{
			{
				Index:        0,
				FinishReason: &stop,
				Content: dto.GeminiChatContent{
					Parts: []dto.GeminiPart{{
						FunctionCall: &dto.FunctionCall{FunctionName: "lookup", Arguments: map[string]any{"value": "x"}},
					}},
				},
			},
			{
				Index:        1,
				FinishReason: &stop,
				Content: dto.GeminiChatContent{
					Parts: []dto.GeminiPart{{Text: "plain text"}},
				},
			},
		},
	}

	openAIResponse := responseGeminiChat2OpenAI(c, info, geminiResponse)
	require.Len(t, openAIResponse.Choices, 2)
	assert.Equal(t, "tool_calls", openAIResponse.Choices[0].FinishReason)
	assert.Equal(t, "stop", openAIResponse.Choices[1].FinishReason)
}
