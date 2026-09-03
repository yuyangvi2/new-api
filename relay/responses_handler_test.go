package relay

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponsesHelperReturnsClientErrorWhenAdaptorDoesNotSupportResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	common.SetContextKey(ctx, constant.ContextKeyChannelType, constant.ChannelTypeReplicate)
	common.SetContextKey(ctx, constant.ContextKeyOriginalModel, "replicate-model")

	err := ResponsesHelper(ctx, &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		RelayFormat:     types.RelayFormatOpenAIResponses,
		OriginModelName: "replicate-model",
		Request: &dto.OpenAIResponsesRequest{
			Model: "replicate-model",
			Input: []byte(`"hi"`),
		},
	})

	require.NotNil(t, err)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, types.ErrorCodeInvalidRequest, err.GetErrorCode())
	assert.Contains(t, err.Error(), "does not support /v1/responses")
	assert.True(t, types.IsSkipRetryError(err))
}

func TestResponsesHelperReturnsBadRequestForUnsupportedChatCompatibilityFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	common.SetContextKey(ctx, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
	common.SetContextKey(ctx, constant.ContextKeyOriginalModel, "deepseek-v4")
	common.SetContextKey(ctx, constant.ContextKeyChannelSetting, dto.ChannelSettings{
		ResponsesViaChatCompletions: true,
	})

	err := ResponsesHelper(ctx, &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		RelayFormat:     types.RelayFormatOpenAIResponses,
		OriginModelName: "deepseek-v4",
		Request: &dto.OpenAIResponsesRequest{
			Model:              "deepseek-v4",
			Input:              []byte(`"hi"`),
			PreviousResponseID: "resp_previous",
		},
	})

	require.NotNil(t, err)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, types.ErrorCodeInvalidRequest, err.GetErrorCode())
	assert.Contains(t, err.Error(), "previous_response_id")
	assert.True(t, types.IsSkipRetryError(err))
}

func TestResponsesHelperReturnsBadRequestForUnsupportedClaudeStatefulFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	common.SetContextKey(ctx, constant.ContextKeyChannelType, constant.ChannelTypeAnthropic)
	common.SetContextKey(ctx, constant.ContextKeyOriginalModel, "claude-sonnet-test")

	err := ResponsesHelper(ctx, &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		RelayFormat:     types.RelayFormatOpenAIResponses,
		OriginModelName: "claude-sonnet-test",
		Request: &dto.OpenAIResponsesRequest{
			Model:              "claude-sonnet-test",
			Input:              []byte(`"hi"`),
			PreviousResponseID: "resp_previous",
		},
	})

	require.NotNil(t, err)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, types.ErrorCodeInvalidRequest, err.GetErrorCode())
	assert.Contains(t, err.Error(), "previous_response_id")
	assert.True(t, types.IsSkipRetryError(err))
}
