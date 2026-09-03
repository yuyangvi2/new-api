package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestConvertOpenAIResponsesRequestUsesChatCompletionsWhenEnabled(t *testing.T) {
	input := common.GetPointer([]byte(`[
		{"role":"user","content":[{"type":"input_text","text":"hello"}]},
		{"type":"function_call","call_id":"call_1","name":"lookup","arguments":{"query":"weather"},"parameters":{"type":"object"}}
	]`))
	request := dto.OpenAIResponsesRequest{
		Model: "deepseek-v4",
		Input: *input,
	}
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenAI,
			UpstreamModelName: "deepseek-v4",
			ChannelSetting: dto.ChannelSettings{
				ResponsesViaChatCompletions: true,
			},
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	chatRequest, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Len(t, chatRequest.Messages, 2)
	assert.Equal(t, "user", chatRequest.Messages[0].Role)
	assert.Equal(t, "hello", chatRequest.Messages[0].StringContent())
	assert.Equal(t, "assistant", chatRequest.Messages[1].Role)

	toolCalls := chatRequest.Messages[1].ParseToolCalls()
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "call_1", toolCalls[0].ID)
	assert.Equal(t, "lookup", toolCalls[0].Function.Name)
	assert.JSONEq(t, `{"query":"weather"}`, toolCalls[0].Function.Arguments)
	assert.Nil(t, toolCalls[0].Function.Parameters)
}

func TestConvertOpenAIResponsesRequestKeepsNativeFormatByDefault(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model: "gpt-test",
		Input: []byte(`"hello"`),
	}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType:       constant.ChannelTypeOpenAI,
		UpstreamModelName: "gpt-test",
	}}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.NoError(t, err)
	_, ok := converted.(dto.OpenAIResponsesRequest)
	assert.True(t, ok)
}

func TestConvertOpenAIResponsesRequestNormalizesNestedFunctionSchema(t *testing.T) {
	tools, err := common.Marshal([]map[string]any{{
		"type": "namespace",
		"name": "automation",
		"tools": []any{map[string]any{
			"type": "function",
			"name": "automation_update",
			"parameters": map[string]any{
				"oneOf": []any{
					map[string]any{
						"type":       "object",
						"properties": map[string]any{"id": map[string]any{"type": "string"}},
						"required":   []any{"id"},
					},
					map[string]any{
						"type":       "object",
						"properties": map[string]any{"status": map[string]any{"type": "string"}},
						"required":   []any{"status"},
					},
				},
			},
		}},
	}})
	require.NoError(t, err)
	request := dto.OpenAIResponsesRequest{Model: "deepseek-v4-flash", Tools: tools}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType:       constant.ChannelTypeOpenAI,
		UpstreamModelName: "deepseek-v4-flash",
	}}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	native, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)

	assert.Equal(t, "object", gjson.GetBytes(native.Tools, "0.tools.0.parameters.type").String())
	assert.False(t, gjson.GetBytes(native.Tools, "0.tools.0.parameters.oneOf").Exists())
	assert.Equal(t, "string", gjson.GetBytes(native.Tools, "0.tools.0.parameters.properties.id.type").String())
	assert.Equal(t, "string", gjson.GetBytes(native.Tools, "0.tools.0.parameters.properties.status.type").String())
	assert.False(t, gjson.GetBytes(native.Tools, "0.tools.0.parameters.required").Exists())
}

func TestConvertOpenAIResponsesRequestMovesLegacyReasoningContentToSummary(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model: "deepseek-v4-flash",
		Input: []byte(`[{"type":"reasoning","id":"rs_1","content":[{"type":"summary_text","text":"thinking"}]}]`),
	}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType:       constant.ChannelTypeOpenAI,
		UpstreamModelName: "deepseek-v4-flash",
	}}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	native, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)

	assert.False(t, gjson.GetBytes(native.Input, "0.content").Exists())
	assert.Equal(t, "summary_text", gjson.GetBytes(native.Input, "0.summary.0.type").String())
	assert.Equal(t, "thinking", gjson.GetBytes(native.Input, "0.summary.0.text").String())
}

func TestGetRequestURLUsesChatCompletionsForResponsesCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		info     *relaycommon.RelayInfo
		wantPath string
	}{
		{
			name: "OpenAI-compatible channel",
			info: &relaycommon.RelayInfo{
				RelayMode:      relayconstant.RelayModeResponses,
				RequestURLPath: "/v1/responses",
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:    constant.ChannelTypeOpenAI,
					ChannelBaseUrl: "https://fw.example",
					ChannelSetting: dto.ChannelSettings{ResponsesViaChatCompletions: true},
				},
			},
			wantPath: "/v1/chat/completions",
		},
		{
			name: "Azure channel",
			info: &relaycommon.RelayInfo{
				RelayMode:      relayconstant.RelayModeResponses,
				RequestURLPath: "/v1/responses",
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:       constant.ChannelTypeAzure,
					ChannelBaseUrl:    "https://azure.example",
					ApiVersion:        "2025-04-01-preview",
					UpstreamModelName: "deepseek-v4",
					ChannelSetting:    dto.ChannelSettings{ResponsesViaChatCompletions: true},
				},
			},
			wantPath: "/openai/deployments/deepseek-v4/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestURL, err := (&Adaptor{}).GetRequestURL(tt.info)
			require.NoError(t, err)

			parsed, err := url.Parse(requestURL)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, parsed.Path)
		})
	}
}

func TestDoResponseConvertsChatCompletionBackToResponsesWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(common.RequestIdKey, "responses-bridge-test")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{
			"id":"chatcmpl_1",
			"object":"chat.completion",
			"created":1710000000,
			"model":"deepseek-v4",
			"choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}
		}`)),
	}
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenAI,
			UpstreamModelName: "deepseek-v4",
			ChannelSetting: dto.ChannelSettings{
				ResponsesViaChatCompletions: true,
			},
		},
	}

	usage, newAPIError := (&Adaptor{}).DoResponse(c, resp, info)

	require.Nil(t, newAPIError)
	require.NotNil(t, usage)
	assert.Contains(t, recorder.Body.String(), `"object":"response"`)
	assert.Contains(t, recorder.Body.String(), `"type":"output_text","text":"hello"`)
}
