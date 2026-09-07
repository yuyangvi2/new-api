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

func TestGeneralErrorResponseDetailsExcludeEchoedRequestContent(t *testing.T) {
	body := []byte(`{
		"error": {
			"type": "invalid_request_error",
			"message": "Invalid request",
			"param": "messages[1].content",
			"details": [
				{
					"loc": ["body", "messages", 1, "content"],
					"msg": "Input should be a valid string",
					"input": "private system prompt that must not be returned"
				}
			]
		}
	}`)
	var response GeneralErrorResponse
	require.NoError(t, common.Unmarshal(body, &response))

	openAIError := response.TryToOpenAIError()

	require.NotNil(t, openAIError)
	assert.Contains(t, openAIError.Message, "Input should be a valid string")
	assert.NotContains(t, openAIError.Message, "private system prompt")
	assert.NotContains(t, string(openAIError.Metadata), "private system prompt")
}

func TestGeneralErrorResponsePreservesSafeTopLevelDetailArray(t *testing.T) {
	body := []byte(`{
		"detail":[{
			"loc":["body","width"],
			"msg":"Input should be less than or equal to 6000",
			"type":"less_than_equal",
			"input":999999
		}]
	}`)
	var response GeneralErrorResponse
	require.NoError(t, common.Unmarshal(body, &response))

	message := response.ToMessage()

	assert.Contains(t, message, "Input should be less than or equal to 6000")
	assert.Contains(t, message, "width")
	assert.NotContains(t, message, "999999")
}
