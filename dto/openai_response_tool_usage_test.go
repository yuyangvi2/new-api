package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsagePreservesTencentWebSearchCallCount(t *testing.T) {
	var response OpenAITextResponse
	require.NoError(t, common.Unmarshal([]byte(`{
		"choices":[],
		"usage":{
			"prompt_tokens":10,
			"completion_tokens":5,
			"total_tokens":15,
			"tool_usage":{"web_search_call":2}
		}
	}`), &response))

	assert.Equal(t, 2, response.Usage.ToolUsage.WebSearchCall)

	encoded, err := common.Marshal(response)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"tool_usage":{"web_search_call":2}`)
}
