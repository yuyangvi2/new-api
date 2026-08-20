package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbortWithOpenAiMessageUsesClaudeErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	abortWithOpenAiMessage(ctx, http.StatusUnauthorized, "invalid token")

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	var body struct {
		Type  string            `json:"type"`
		Error types.ClaudeError `json:"error"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "error", body.Type)
	assert.Equal(t, "authentication_error", body.Error.Type)
	assert.Contains(t, body.Error.Message, "invalid token")
}

func TestAbortWithOpenAiMessageUsesGeminiErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-test:generateContent", nil)

	abortWithOpenAiMessage(ctx, http.StatusBadRequest, "invalid JSON request body")

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var body struct {
		Error types.GeminiError `json:"error"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, http.StatusBadRequest, body.Error.Code)
	assert.Equal(t, "INVALID_ARGUMENT", body.Error.Status)
	assert.Contains(t, body.Error.Message, "invalid JSON request body")
}
