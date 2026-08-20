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
	common.SetContextKey(ctx, constant.ContextKeyChannelType, constant.ChannelTypeAnthropic)
	common.SetContextKey(ctx, constant.ContextKeyOriginalModel, "claude-opus-4-1")

	err := ResponsesHelper(ctx, &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		RelayFormat:     types.RelayFormatOpenAIResponses,
		OriginModelName: "claude-opus-4-1",
		Request: &dto.OpenAIResponsesRequest{
			Model: "claude-opus-4-1",
			Input: []byte(`"hi"`),
		},
	})

	require.NotNil(t, err)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, types.ErrorCodeInvalidRequest, err.GetErrorCode())
	assert.Contains(t, err.Error(), "does not support /v1/responses")
	assert.True(t, types.IsSkipRetryError(err))
}
