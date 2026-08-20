package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayValidationErrorUsesClaudeEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/v1/messages", func(c *gin.Context) {
		Relay(c, types.RelayFormatClaude)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var body struct {
		Type  string            `json:"type"`
		Error types.ClaudeError `json:"error"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "error", body.Type)
	assert.Equal(t, "invalid_request_error", body.Error.Type)
	assert.Contains(t, body.Error.Message, "messages is required")
}

func TestRelayValidationErrorUsesGeminiEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/v1beta/models/gemini-test:generateContent", func(c *gin.Context) {
		Relay(c, types.RelayFormatGemini)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-test:generateContent", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var body struct {
		Error types.GeminiError `json:"error"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, http.StatusBadRequest, body.Error.Code)
	assert.Equal(t, "INVALID_ARGUMENT", body.Error.Status)
	assert.Contains(t, body.Error.Message, "contents is required")
}
