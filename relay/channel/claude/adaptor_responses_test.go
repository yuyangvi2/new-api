package claude

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIResponsesRequestToClaude(t *testing.T) {
	stream := true
	temperature := 0.0
	maxOutputTokens := uint(256)
	request := dto.OpenAIResponsesRequest{
		Model:             "claude-sonnet-test",
		Input:             []byte(`[{"role":"user","content":[{"type":"input_text","text":"read a file"}]}]`),
		Instructions:      []byte(`"Be concise"`),
		MaxOutputTokens:   &maxOutputTokens,
		Temperature:       &temperature,
		Stream:            &stream,
		ParallelToolCalls: []byte(`true`),
		ToolChoice:        []byte(`"auto"`),
		Tools:             []byte(`[{"type":"function","name":"read_file","description":"Read one file","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}}]`),
	}
	info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatOpenAIResponses}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)

	claudeRequest, ok := converted.(*dto.ClaudeRequest)
	require.True(t, ok)
	assert.Equal(t, "claude-sonnet-test", claudeRequest.Model)
	require.NotNil(t, claudeRequest.MaxTokens)
	assert.Equal(t, uint(256), *claudeRequest.MaxTokens)
	require.NotNil(t, claudeRequest.Temperature)
	assert.Equal(t, 0.0, *claudeRequest.Temperature)
	require.NotNil(t, claudeRequest.Stream)
	assert.True(t, *claudeRequest.Stream)
	assert.Equal(t, []types.RelayFormat{types.RelayFormatOpenAIResponses, types.RelayFormatClaude}, info.RequestConversionChain)

	system, err := common.Any2Type[[]dto.ClaudeMediaMessage](claudeRequest.System)
	require.NoError(t, err)
	require.Len(t, system, 1)
	assert.Equal(t, "Be concise", system[0].GetText())
	require.Len(t, claudeRequest.Messages, 1)
	assert.Equal(t, "user", claudeRequest.Messages[0].Role)

	tools, err := common.Any2Type[[]dto.Tool](claudeRequest.Tools)
	require.NoError(t, err)
	require.Len(t, tools, 1)
	assert.Equal(t, "read_file", tools[0].Name)
	require.NotNil(t, claudeRequest.ToolChoice)
}

func TestConvertOpenAIResponsesRequestRejectsStatefulFields(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model:              "claude-sonnet-test",
		Input:              []byte(`"hello"`),
		PreviousResponseID: "resp_previous",
	}

	_, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, &relaycommon.RelayInfo{}, request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "previous_response_id")
}
