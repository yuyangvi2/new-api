package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRespondTaskErrorPreservesActionableRateLimitMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	taskErr := &dto.TaskError{
		Code:       "rate_limit_exceeded",
		Message:    "Rate limit exceeded, retry after 10 seconds.",
		StatusCode: http.StatusTooManyRequests,
	}

	respondTaskError(ctx, taskErr)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	var response dto.TaskError
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "rate_limit_exceeded", response.Code)
	assert.Equal(t, "Rate limit exceeded, retry after 10 seconds.", response.Message)
}
