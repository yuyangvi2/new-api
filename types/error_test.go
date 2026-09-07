package types

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToOpenAIErrorSanitizesStructuredFieldsAtResponseBoundary(t *testing.T) {
	apiError := WithOpenAIError(OpenAIError{
		Message: "Invalid reasoning.effort in schema.json; credential sk-proj-abcdefghijklmnop",
		Type:    "validation.failed",
		Param:   "reasoning.effort",
		Code:    "invalid.reasoning_effort",
		Metadata: json.RawMessage(`{
			"field":"tools.0.function.name",
			"token":"private-token",
			"endpoint":"https://internal.example.com/v1"
		}`),
	}, http.StatusBadRequest)

	result := apiError.ToOpenAIError()

	require.Contains(t, result.Message, "reasoning.effort")
	require.Contains(t, result.Message, "schema.json")
	require.NotContains(t, result.Message, "sk-proj-")
	require.Equal(t, "validation.failed", result.Type)
	require.Equal(t, "reasoning.effort", result.Param)
	require.Equal(t, "invalid.reasoning_effort", result.Code)
	require.Contains(t, string(result.Metadata), "tools.0.function.name")
	require.NotContains(t, string(result.Metadata), "private-token")
	require.NotContains(t, string(result.Metadata), "internal.example.com")
}

func TestToOpenAIErrorRejectsUnsafeStructuredIdentifiers(t *testing.T) {
	apiError := WithOpenAIError(OpenAIError{
		Message: "Invalid parameter",
		Type:    "Authorization: custom-secret",
		Param:   "reasoning.effort\r\nX-Internal: secret",
		Code:    "api_key=plain-secret",
	}, http.StatusBadRequest)

	result := apiError.ToOpenAIError()

	require.Equal(t, "upstream_error", result.Type)
	require.Empty(t, result.Param)
	require.Equal(t, ErrorCodeBadResponseStatusCode, result.Code)
}
