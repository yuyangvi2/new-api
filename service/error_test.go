package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResetStatusCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		statusCode       int
		statusCodeConfig string
		expectedCode     int
	}{
		{
			name:             "map string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"503"}`,
			expectedCode:     503,
		},
		{
			name:             "map int value",
			statusCode:       429,
			statusCodeConfig: `{"429":503}`,
			expectedCode:     503,
		},
		{
			name:             "skip invalid string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"bad-code"}`,
			expectedCode:     429,
		},
		{
			name:             "skip status code 200",
			statusCode:       200,
			statusCodeConfig: `{"200":503}`,
			expectedCode:     200,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			newAPIError := &types.NewAPIError{
				StatusCode: tc.statusCode,
			}
			ResetStatusCode(newAPIError, tc.statusCodeConfig)
			require.Equal(t, tc.expectedCode, newAPIError.StatusCode)
		})
	}
}

func TestRelayErrorHandlerTruncatesInvalidJSONBodyInLog(t *testing.T) {
	withDebugEnabled(t, false)

	body := strings.Repeat("b", common.LocalLogContentLimit+256)
	var logBuffer bytes.Buffer

	common.LogWriterMu.Lock()
	oldWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logBuffer
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = oldWriter
		common.LogWriterMu.Unlock()
	})

	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, "bad response status code 500", newAPIError.Error())
	require.Contains(t, logBuffer.String(), "[truncated")
	require.Contains(t, logBuffer.String(), fmt.Sprintf("original_length=%d", len(body)))
	require.NotContains(t, logBuffer.String(), strings.Repeat("b", common.LocalLogContentLimit+1))
}

func TestRelayErrorHandlerKeepsStructuredErrorMessage(t *testing.T) {
	message := strings.Repeat("c", common.LocalLogContentLimit+256)
	body := `{"message":"` + message + `"}`
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, message, newAPIError.Error())
}

func TestRelayErrorHandlerKeepsOpenAIErrorMessage(t *testing.T) {
	message := strings.Repeat("d", common.LocalLogContentLimit+256)
	body := `{"error":{"message":"` + message + `","type":"server_error","code":"server_error"}}`
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, message, newAPIError.Error())
}

func TestRelayErrorHandlerKeepsTopLevelErrorWhenUnrelatedFieldsHaveDifferentTypes(t *testing.T) {
	body := `{
		"object":"error",
		"message":"Model only supports text input; received unsupported content type 'image_url'.",
		"type":"BadRequestError",
		"param":"messages[0].content",
		"code":400,
		"detail":{"internal":"must not break message parsing"}
	}`
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, "Model only supports text input; received unsupported content type 'image_url'.", newAPIError.Error())
	openAIError := newAPIError.ToOpenAIError()
	require.Equal(t, "BadRequestError", openAIError.Type)
	require.Equal(t, "messages[0].content", openAIError.Param)
	require.EqualValues(t, 400, openAIError.Code)
}

func TestRelayErrorHandlerPreservesDottedDiagnosticsWithoutExposingSecrets(t *testing.T) {
	body := `{
		"error":{
			"message":"Invalid value for reasoning.effort in image.png; credential sk-proj-abcdefghijklmnop; endpoint https://internal.example.com/v1",
			"type":"validation.failed",
			"param":"reasoning.effort",
			"code":"invalid.reasoning_effort",
			"metadata":{
				"field":"tools.0.function.name",
				"api_key":"plain-secret",
				"endpoint":"https://internal.example.com/v1"
			}
		}
	}`
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	openAIError := newAPIError.ToOpenAIError()
	require.Contains(t, openAIError.Message, "reasoning.effort")
	require.Contains(t, openAIError.Message, "image.png")
	require.NotContains(t, openAIError.Message, "sk-proj-")
	require.NotContains(t, openAIError.Message, "internal.example.com")
	require.Equal(t, "validation.failed", openAIError.Type)
	require.Equal(t, "reasoning.effort", openAIError.Param)
	require.Equal(t, "invalid.reasoning_effort", openAIError.Code)
	require.Contains(t, string(openAIError.Metadata), "tools.0.function.name")
	require.NotContains(t, string(openAIError.Metadata), "plain-secret")
	require.NotContains(t, string(openAIError.Metadata), "internal.example.com")

	claudeError := newAPIError.ToClaudeError()
	require.Contains(t, claudeError.Message, "reasoning.effort")
	require.Contains(t, claudeError.Message, "image.png")

	geminiError := newAPIError.ToGeminiError()
	require.Contains(t, geminiError.Message, "reasoning.effort")
	require.Contains(t, geminiError.Message, "image.png")
}

func TestRelayErrorHandlerUsesNonEmptyFallbackWhenUpstreamOmitsMessage(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`{"error":{},"code":400}`)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, "Upstream returned an error without details", newAPIError.Error())
}

func TestRelayErrorHandlerHidesUpstreamCredentialFailure(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body: io.NopCloser(strings.NewReader(`{
			"message":"Invalid bearer sk-proj-abcdefghijklmnop",
			"type":"authentication_error",
			"code":"invalid_api_key"
		}`)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, "Upstream authentication failed, please contact administrator", newAPIError.Error())
	require.NotContains(t, newAPIError.ToOpenAIError().Message, "sk-proj-")
}

func TestRelayErrorHandlerHidesProviderAccountBalance(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body: io.NopCloser(strings.NewReader(`{
			"error":{"message":"Insufficient account balance","type":"billing_error"}
		}`)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, "Upstream service temporarily unavailable, please contact administrator", newAPIError.Error())
}

func TestRelayErrorHandlerSanitizesUpstreamTypeAndCode(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body: io.NopCloser(strings.NewReader(`{
			"message":"Invalid request",
			"type":"Authorization: custom-secret",
			"code":{"x-api-key":"plain-secret"}
		}`)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	serialized, err := common.Marshal(newAPIError.ToOpenAIError())
	require.NoError(t, err)
	require.NotContains(t, string(serialized), "custom-secret")
	require.NotContains(t, string(serialized), "plain-secret")
}

func TestRelayErrorHandlerAuthenticationTakesPrecedenceOverPolicyWords(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body: io.NopCloser(strings.NewReader(`{
			"message":"Moderation service authentication failed for upstream account",
			"type":"authentication_error",
			"code":"unauthorized"
		}`)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, "Upstream authentication failed, please contact administrator", newAPIError.Error())
	require.Equal(t, "upstream_authentication_failed", newAPIError.ToOpenAIError().Code)
}

func TestRelayErrorHandlerDefaultsMissingTopLevelCode(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`{"message":"Unsupported image format"}`)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, types.ErrorCodeBadResponseStatusCode, newAPIError.ToOpenAIError().Code)
}

func TestSanitizeTaskRelayErrorProtectsHTTP200BusinessFailure(t *testing.T) {
	taskErr := &dto.TaskError{
		Code:       "invalid_api_key",
		Message:    "Invalid API key x-api-key=plain-secret for moderation service",
		Data:       map[string]any{"authorization": "custom-secret"},
		StatusCode: http.StatusBadRequest,
	}

	safeError := SanitizeTaskRelayError(taskErr)

	require.Equal(t, "Upstream authentication failed, please contact administrator", safeError.Message)
	require.Equal(t, "upstream_authentication_failed", safeError.Code)
	require.Nil(t, safeError.Data)
	require.NotContains(t, safeError.Error.Error(), "plain-secret")
}

func TestSanitizeTaskRelayErrorSuppliesFallbackMessage(t *testing.T) {
	taskErr := &dto.TaskError{
		Code:       "task_failed",
		StatusCode: http.StatusBadRequest,
	}

	safeError := SanitizeTaskRelayError(taskErr)

	require.NotEmpty(t, safeError.Message)
	require.NotEmpty(t, safeError.Error.Error())
}

func TestSanitizeUpstreamTaskErrorPreservesSafeCode(t *testing.T) {
	safeError := SanitizeUpstreamTaskErrorWithCode("Width is invalid", "InvalidParameter")

	require.Equal(t, "Width is invalid", safeError.Message)
	require.Equal(t, "InvalidParameter", safeError.Code)
}

func TestSanitizeUpstreamTaskErrorPreservesDottedDiagnostics(t *testing.T) {
	safeError := SanitizeUpstreamTaskErrorWithCode(
		"Invalid value for tools.0.function.name in schema.json",
		"validation.failed",
	)

	require.Equal(t, "Invalid value for tools.0.function.name in schema.json", safeError.Message)
	require.Equal(t, "validation.failed", safeError.Code)
}

func TestRelayErrorHandlerKeepsPolicyViolationReason(t *testing.T) {
	message := "Request rejected due to terms of use violation."
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body: io.NopCloser(strings.NewReader(`{
			"error":{"message":"` + message + `","type":"permission_error"}
		}`)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, message, newAPIError.Error())
}

func TestSanitizeUpstreamTaskErrorPreservesActionableReasons(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"The request failed because the output video may be related to copyright restrictions. Request id: req_123",
		"Width must be between 300px and 6000px.",
		"The parameter duration is not valid for this model.",
		"Rate limit exceeded, retry after 10 seconds.",
		"This model is not available in the requested region.",
	}

	for _, message := range testCases {
		message := message
		t.Run(message, func(t *testing.T) {
			t.Parallel()

			safeError := SanitizeUpstreamTaskError(message)

			require.Equal(t, message, safeError.Message)
			require.Equal(t, "upstream_task_failed", safeError.Code)
		})
	}
}

func TestSanitizeUpstreamTaskErrorHidesProviderSecrets(t *testing.T) {
	t.Parallel()

	safeError := SanitizeUpstreamTaskError("Invalid API key sk-proj-abcdefghijklmnop for upstream account")

	require.Equal(t, "Upstream authentication failed, please contact administrator", safeError.Message)
	require.Equal(t, "upstream_authentication_failed", safeError.Code)
	require.NotContains(t, safeError.Message, "sk-proj-")
}

func TestSanitizeUpstreamTaskErrorReplacesMissingDetails(t *testing.T) {
	t.Parallel()

	for _, message := range []string{"", "Unknown error"} {
		safeError := SanitizeUpstreamTaskError(message)

		require.Equal(t, "Upstream task failed without error details", safeError.Message)
		require.Equal(t, "upstream_task_failed", safeError.Code)
	}
}

func TestRelayErrorHandlerMasksSensitiveFallbackMessageBeforeLogging(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body: io.NopCloser(strings.NewReader(`{
			"msg":"Authorization: Bearer token.payload.signature failed for user@example.com"
		}`)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.NotContains(t, newAPIError.Error(), "token.payload.signature")
	require.NotContains(t, newAPIError.Error(), "user@example.com")
}

func TestRelayErrorHandlerMasksSensitiveInvalidJSONBodyInLog(t *testing.T) {
	withDebugEnabled(t, false)

	body := "upstream failed Authorization: Bearer token.payload.signature for user@example.com"
	var logBuffer bytes.Buffer

	common.LogWriterMu.Lock()
	oldWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logBuffer
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = oldWriter
		common.LogWriterMu.Unlock()
	})

	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.NotContains(t, logBuffer.String(), "token.payload.signature")
	require.NotContains(t, logBuffer.String(), "user@example.com")
}

func TestRelayErrorHandlerKeepsInvalidJSONBodyInDebugLog(t *testing.T) {
	withDebugEnabled(t, true)

	body := strings.Repeat("e", common.LocalLogContentLimit+256)
	var logBuffer bytes.Buffer

	common.LogWriterMu.Lock()
	oldWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logBuffer
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = oldWriter
		common.LogWriterMu.Unlock()
	})

	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.NotContains(t, logBuffer.String(), "[truncated")
	require.Contains(t, logBuffer.String(), body)
}

func withDebugEnabled(t *testing.T, enabled bool) {
	t.Helper()

	oldDebug := common.DebugEnabled
	common.DebugEnabled = enabled
	t.Cleanup(func() {
		common.DebugEnabled = oldDebug
	})
}
