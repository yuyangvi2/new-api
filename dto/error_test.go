package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneralErrorResponsePreservesUpstreamErrorDetails(t *testing.T) {
	body := []byte(`{
		"error": {
			"type": "invalid_request_error",
			"message": "Invalid request",
			"param": "top_p",
			"code": "invalid_request",
			"details": [
				{
					"loc": ["body", "top_p"],
					"msg": "extra fields not permitted"
				}
			]
		}
	}`)
	var response GeneralErrorResponse
	require.NoError(t, common.Unmarshal(body, &response))

	openAIError := response.TryToOpenAIError()

	require.NotNil(t, openAIError)
	assert.Equal(t, "invalid_request_error", openAIError.Type)
	assert.Equal(t, "top_p", openAIError.Param)
	assert.Equal(t, "invalid_request", openAIError.Code)
	assert.Contains(t, openAIError.Message, "Invalid request")
	assert.Contains(t, openAIError.Message, "top_p")
	assert.Contains(t, openAIError.Message, "extra fields not permitted")
	assert.Contains(t, string(openAIError.Metadata), "details")
	assert.Contains(t, string(openAIError.Metadata), "param")
}

func TestGeneralErrorResponseToMessagePreservesObjectDetails(t *testing.T) {
	body := []byte(`{
		"error": {
			"type": "invalid_request_error",
			"message": "Invalid request",
			"detail": "top_p is not supported for this model"
		}
	}`)
	var response GeneralErrorResponse
	require.NoError(t, common.Unmarshal(body, &response))

	message := response.ToMessage()

	assert.Contains(t, message, "Invalid request")
	assert.Contains(t, message, "top_p is not supported")
}
