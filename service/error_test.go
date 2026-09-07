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
