package claude

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
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
)

func TestClaudeResponsesHandlerReturnsOpenAIResponsesJSON(t *testing.T) {
	c, recorder := newClaudeResponsesTestContext(t)
	response := dto.ClaudeResponse{
		Id:         "msg_test",
		Type:       "message",
		Role:       "assistant",
		Model:      "claude-sonnet-test",
		StopReason: "tool_use",
		Content: []dto.ClaudeMediaMessage{
			{Type: "text", Text: common.GetPointer("I will ")},
			{Type: "text", Text: common.GetPointer("inspect it.")},
			{Type: "tool_use", Id: "toolu_1", Name: "read_file", Input: map[string]any{"path": "README.md"}},
		},
		Usage: &dto.ClaudeUsage{
			InputTokens:              10,
			CacheReadInputTokens:     6,
			CacheCreationInputTokens: 2,
			OutputTokens:             4,
		},
	}
	body, err := common.Marshal(response)
	require.NoError(t, err)

	usageValue, apiErr := (&Adaptor{}).DoResponse(c, &http.Response{Body: io.NopCloser(bytes.NewReader(body))}, newClaudeResponsesRelayInfo(false))
	require.Nil(t, apiErr)
	usage, ok := usageValue.(*dto.Usage)
	require.True(t, ok)
	require.NotNil(t, usage)
	assert.Equal(t, 10, usage.PromptTokens)
	assert.Equal(t, 4, usage.CompletionTokens)
	assert.Equal(t, 14, usage.TotalTokens)

	var got dto.OpenAIResponsesResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &got))
	assert.Equal(t, "msg_test", got.ID)
	assert.Equal(t, "response", got.Object)
	assert.Equal(t, "claude-sonnet-test", got.Model)
	require.Len(t, got.Output, 2)
	assert.Equal(t, "message", got.Output[0].Type)
	assert.Equal(t, "I will inspect it.", got.Output[0].Content[0].Text)
	assert.Equal(t, "function_call", got.Output[1].Type)
	assert.Equal(t, "toolu_1", got.Output[1].CallId)
	assert.Equal(t, "read_file", got.Output[1].Name)
	assert.JSONEq(t, `{"path":"README.md"}`, dto.ResponsesArgumentsString(got.Output[1].Arguments))
	require.NotNil(t, got.Usage)
	assert.Equal(t, 18, got.Usage.InputTokens)
	assert.Equal(t, 6, got.Usage.InputTokensDetails.CachedTokens)
	assert.Equal(t, 2, got.Usage.InputTokensDetails.CacheWriteTokens)
}

func TestClaudeResponsesStreamHandlerPreservesToolEventOrderAndUsage(t *testing.T) {
	c, recorder := newClaudeResponsesTestContext(t)
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 300
	t.Cleanup(func() { constant.StreamingTimeout = oldStreamingTimeout })
	streamBody := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_stream","type":"message","role":"assistant","model":"claude-sonnet-test","content":[],"usage":{"input_tokens":10,"cache_read_input_tokens":6,"cache_creation_input_tokens":2,"output_tokens":0}}}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"read_file","input":{}}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"path\":"}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"README.md\"}"}}`,
		``,
		`event: content_block_stop`,
		`data: {"type":"content_block_stop","index":0}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":4}}`,
		``,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n")

	usageValue, apiErr := (&Adaptor{}).DoResponse(c, &http.Response{Body: io.NopCloser(strings.NewReader(streamBody))}, newClaudeResponsesRelayInfo(true))
	require.Nil(t, apiErr)
	usage, ok := usageValue.(*dto.Usage)
	require.True(t, ok)
	require.NotNil(t, usage)
	assert.Equal(t, 14, usage.TotalTokens)

	got := recorder.Body.String()
	assert.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
	assert.Contains(t, got, `event: response.created`)
	assert.Contains(t, got, `event: response.function_call_arguments.delta`)
	assert.Contains(t, got, `event: response.completed`)
	assert.Contains(t, got, `"input_tokens":18`)
	assert.Contains(t, got, `"cached_tokens":6`)
	assert.Contains(t, got, `"cache_write_tokens":2`)
	requireOrderedClaudeResponsesSubstrings(t, got,
		`event: response.created`,
		`event: response.output_item.added`,
		`event: response.function_call_arguments.delta`,
		`event: response.function_call_arguments.done`,
		`event: response.output_item.done`,
		`event: response.completed`,
	)
}

func newClaudeResponsesTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(common.RequestIdKey, "claude-responses-test")
	return c, recorder
}

func newClaudeResponsesRelayInfo(isStream bool) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		IsStream:        isStream,
		RelayMode:       relayconstant.RelayModeResponses,
		RelayFormat:     types.RelayFormatOpenAIResponses,
		RequestURLPath:  "/v1/responses",
		DisablePing:     true,
		OriginModelName: "claude-sonnet-test",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-sonnet-test",
		},
	}
}

func requireOrderedClaudeResponsesSubstrings(t *testing.T, value string, parts ...string) {
	t.Helper()
	offset := 0
	for _, part := range parts {
		index := strings.Index(value[offset:], part)
		require.NotEqualf(t, -1, index, "missing %q after byte offset %d", part, offset)
		offset += index + len(part)
	}
}
