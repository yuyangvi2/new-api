package openai

import (
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOaiResponsesToChatStreamHandlerEmitsToolArgumentsOnce(t *testing.T) {
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
	fixture, err := os.ReadFile("testdata/responses_tool_call.sse")
	require.NoError(t, err)
	c, recorder, resp, info := newResponsesChatTestContext(t, string(fixture), true)
	usage, apiErr := OaiResponsesToChatStreamHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 155, usage.TotalTokens)
	arguments, name := "", ""
	finishCount, usageCount, doneCount := 0, 0, 0
	for _, line := range strings.Split(recorder.Body.String(), "\n") {
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			doneCount++
			continue
		}
		assert.Zero(t, doneCount, "chunks must precede DONE")
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.UnmarshalJsonStr(data, &chunk))
		if chunk.Usage != nil {
			usageCount++
			assert.Equal(t, 155, chunk.Usage.TotalTokens)
			assert.Equal(t, 1, finishCount)
		}
		for _, choice := range chunk.Choices {
			if choice.FinishReason != nil {
				finishCount++
				assert.Equal(t, "tool_calls", *choice.FinishReason)
			}
			for _, call := range choice.Delta.ToolCalls {
				assert.Zero(t, finishCount, "tool chunks must precede finish_reason")
				require.NotNil(t, call.Index)
				assert.Equal(t, 0, *call.Index)
				assert.Equal(t, "call_test", call.ID)
				name += call.Function.Name
				arguments += call.Function.Arguments
			}
		}
	}
	assert.Equal(t, "read", name)
	assert.Equal(t, `{"limit":2000,"path":"template/reusable-skill/SKILL.md"}`, arguments)
	var parsed map[string]any
	require.NoError(t, common.UnmarshalJsonStr(arguments, &parsed))
	assert.Equal(t, "template/reusable-skill/SKILL.md", parsed["path"])
	assert.Equal(t, 1, finishCount)
	assert.Equal(t, 1, usageCount)
	assert.Equal(t, 1, doneCount)
}

func TestOaiResponsesToChatBufferedStreamHandlerPreservesToolSnapshot(t *testing.T) {
	fixture, err := os.ReadFile("testdata/responses_tool_call.sse")
	require.NoError(t, err)
	c, recorder, resp, info := newResponsesChatTestContext(t, string(fixture), false)
	usage, apiErr := OaiResponsesToChatBufferedStreamHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	var response dto.OpenAITextResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Choices, 1)
	calls := response.Choices[0].Message.ParseToolCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "call_test", calls[0].ID)
	assert.Equal(t, "read", calls[0].Function.Name)
	assert.Equal(t, `{"limit":2000,"path":"template/reusable-skill/SKILL.md"}`, calls[0].Function.Arguments)
	assert.Equal(t, "tool_calls", response.Choices[0].FinishReason)
	assert.Equal(t, 155, response.Usage.TotalTokens)
}

func TestOaiResponsesToChatStreamHandlerReportsConflictingToolSnapshot(t *testing.T) {
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
	fixture, err := os.ReadFile("testdata/responses_tool_call.sse")
	require.NoError(t, err)
	lines := strings.Split(string(fixture), "\n")
	for i, line := range lines {
		if strings.Contains(line, `"type":"response.completed"`) {
			lines[i] = strings.ReplaceAll(line, `\"limit\":2000`, `\"limit\":3000`)
		}
	}
	c, recorder, resp, info := newResponsesChatTestContext(t, strings.Join(lines, "\n"), true)
	_, apiErr := OaiResponsesToChatStreamHandler(c, info, resp)
	require.NotNil(t, apiErr)
	assert.Contains(t, apiErr.Error(), "snapshot conflicts")
	assert.NotContains(t, recorder.Body.String(), `"finish_reason":"tool_calls"`)
	assert.NotContains(t, recorder.Body.String(), `data: [DONE]`)
}
