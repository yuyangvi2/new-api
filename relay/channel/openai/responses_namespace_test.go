package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestConvertOpenAIResponsesRequestFlattensNamespaceToolsAndQualifiedReferences(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	request := dto.OpenAIResponsesRequest{
		Model: "test-model",
		Tools: []byte(`[
			{"type":"function","name":"lookup","description":"ordinary","parameters":{"type":"object"}},
			{"type":"namespace","name":"automation","tools":[
				{"type":"function","name":"automation_update","description":"update an automation","strict":true,"parameters":{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}},
				{"type":"function","name":"automation_delete","parameters":{"type":"object"}}
			]}
		]`),
		Input: []byte(`[
			{"type":"function_call","call_id":"call_1","name":"automation_update","namespace":"automation","arguments":"{\"id\":\"1\"}"},
			{"type":"function_call_output","call_id":"call_1","output":"ok"}
		]`),
		ToolChoice: []byte(`{"type":"function","name":"automation_update","namespace":"automation"}`),
	}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType:       constant.ChannelTypeOpenAI,
		UpstreamModelName: "test-model",
	}}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(c, info, request)
	require.NoError(t, err)
	native, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)

	assert.Equal(t, "lookup", gjson.GetBytes(native.Tools, "0.name").String())
	assert.Equal(t, "automation__automation_update", gjson.GetBytes(native.Tools, "1.name").String())
	assert.Equal(t, "update an automation", gjson.GetBytes(native.Tools, "1.description").String())
	assert.True(t, gjson.GetBytes(native.Tools, "1.strict").Bool())
	assert.Equal(t, "automation__automation_delete", gjson.GetBytes(native.Tools, "2.name").String())
	assert.False(t, gjson.GetBytes(native.Tools, "#(type==\"namespace\")").Exists())
	assert.Equal(t, "automation__automation_update", gjson.GetBytes(native.Input, "0.name").String())
	assert.False(t, gjson.GetBytes(native.Input, "0.namespace").Exists())
	assert.Equal(t, "automation__automation_update", gjson.GetBytes(native.ToolChoice, "name").String())
	assert.False(t, gjson.GetBytes(native.ToolChoice, "namespace").Exists())

	rawMapping, exists := c.Get(responsesNamespaceToolNamesContextKey)
	require.True(t, exists)
	mapping, ok := rawMapping.(map[string]responsesNamespaceToolName)
	require.True(t, ok)
	assert.Equal(t, responsesNamespaceToolName{Namespace: "automation", Name: "automation_update"}, mapping["automation__automation_update"])
}

func TestConvertOpenAIResponsesRequestRejectsNamespaceFlatteningCollision(t *testing.T) {
	request := dto.OpenAIResponsesRequest{
		Model: "test-model",
		Tools: []byte(`[
			{"type":"function","name":"automation__automation_update","parameters":{"type":"object"}},
			{"type":"namespace","name":"automation","children":[{"type":"function","name":"automation_update","parameters":{"type":"object"}}]}
		]`),
	}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType:       constant.ChannelTypeOpenAI,
		UpstreamModelName: "test-model",
	}}

	_, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicts with a top-level tool")
}

func TestConvertOpenAIResponsesRequestUsesBoundedStableNamespaceToolNames(t *testing.T) {
	namespace := strings.Repeat("namespace", 8)
	name := strings.Repeat("function", 8)
	request := dto.OpenAIResponsesRequest{
		Model: "test-model",
		Tools: []byte(`[{"type":"namespace","name":"` + namespace + `","tools":[{"type":"function","name":"` + name + `","parameters":{"type":"object"}}]}]`),
	}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType:       constant.ChannelTypeOpenAI,
		UpstreamModelName: "test-model",
	}}

	first, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	second, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	firstName := gjson.GetBytes(first.(dto.OpenAIResponsesRequest).Tools, "0.name").String()
	secondName := gjson.GetBytes(second.(dto.OpenAIResponsesRequest).Tools, "0.name").String()

	assert.Len(t, firstName, responsesFunctionNameMaxLength)
	assert.Equal(t, firstName, secondName)
	assert.Contains(t, firstName, "__")
}

func TestOaiResponsesHandlerRestoresNamespaceFunctionCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(responsesNamespaceToolNamesContextKey, map[string]responsesNamespaceToolName{
		"automation__automation_update": {Namespace: "automation", Name: "automation_update"},
	})
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{
			"id":"resp_1","object":"response","status":"completed",
			"output":[{"type":"function_call","id":"fc_1","call_id":"call_1","name":"automation__automation_update","arguments":"{\"id\":\"1\"}"}],
			"usage":{"input_tokens":2,"output_tokens":3,"total_tokens":5}
		}`)),
	}

	usage, apiErr := OaiResponsesHandler(c, &relaycommon.RelayInfo{}, resp)

	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 5, usage.TotalTokens)
	assert.Equal(t, "automation_update", gjson.Get(recorder.Body.String(), "output.0.name").String())
	assert.Equal(t, "automation", gjson.Get(recorder.Body.String(), "output.0.namespace").String())
}

func TestOaiResponsesStreamHandlerRestoresNamespaceAcrossEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(responsesNamespaceToolNamesContextKey, map[string]responsesNamespaceToolName{
		"automation__automation_update": {Namespace: "automation", Name: "automation_update"},
	})
	body := strings.Join([]string{
		`data: {"type":"response.output_item.added","sequence_number":1,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"automation__automation_update"}}`,
		`data: {"type":"response.output_item.done","sequence_number":2,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"automation__automation_update","arguments":"{}"}}`,
		`data: {"type":"response.completed","sequence_number":3,"response":{"id":"resp_1","status":"completed","output":[{"type":"function_call","id":"fc_1","call_id":"call_1","name":"automation__automation_update","arguments":"{}"}],"usage":{"input_tokens":2,"output_tokens":3,"total_tokens":5}}}`,
		`data: [DONE]`,
		``,
	}, "\n")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	info := &relaycommon.RelayInfo{DisablePing: true, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "test-model"}}

	usage, apiErr := OaiResponsesStreamHandler(c, info, resp)

	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 5, usage.TotalTokens)
	output := recorder.Body.String()
	assert.Equal(t, 3, strings.Count(output, `"name":"automation_update"`))
	assert.Equal(t, 3, strings.Count(output, `"namespace":"automation"`))
	assert.NotContains(t, output, "automation__automation_update")
	assert.Contains(t, output, `"sequence_number":1`)
}

func TestRestoreResponsesNamespaceCallsLeavesUnmappedPayloadUnchanged(t *testing.T) {
	payload := []byte(`{"type":"response.output_item.added","item":{"type":"function_call","name":"lookup"},"sequence_number":9}`)

	restored, changed, err := restoreResponsesNamespaceCalls(payload, nil)

	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, payload, restored)
}
