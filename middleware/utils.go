package middleware

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func abortWithOpenAiMessage(c *gin.Context, statusCode int, message string, code ...types.ErrorCode) {
	codeStr := ""
	if len(code) > 0 {
		codeStr = string(code[0])
	}
	responseMessage := common.MessageWithRequestId(message, c.GetString(common.RequestIdKey))
	userId := c.GetInt("id")
	openAIError := types.OpenAIError{
		Message: responseMessage,
		Type:    "new_api_error",
		Code:    codeStr,
	}
	newAPIError := types.WithOpenAIError(openAIError, statusCode, types.ErrOptionWithSkipRetry())
	switch inferErrorRelayFormat(c) {
	case types.RelayFormatClaude:
		c.JSON(statusCode, gin.H{
			"type":  "error",
			"error": newAPIError.ToClaudeError(),
		})
	case types.RelayFormatGemini:
		c.JSON(statusCode, gin.H{
			"error": newAPIError.ToGeminiError(),
		})
	default:
		c.JSON(statusCode, gin.H{
			"error": openAIError,
		})
	}
	c.Abort()
	logger.LogError(c.Request.Context(), fmt.Sprintf("user %d | %s", userId, message))
}

func inferErrorRelayFormat(c *gin.Context) types.RelayFormat {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return types.RelayFormatOpenAI
	}
	path := c.Request.URL.Path
	if strings.Contains(path, "/v1/messages") ||
		(strings.HasPrefix(path, "/v1/models") && c.GetHeader("x-api-key") != "" && c.GetHeader("anthropic-version") != "") {
		return types.RelayFormatClaude
	}
	if strings.HasPrefix(path, "/v1beta/models") ||
		(strings.HasPrefix(path, "/v1/models") && (c.GetHeader("x-goog-api-key") != "" || c.Query("key") != "")) {
		return types.RelayFormatGemini
	}
	return types.RelayFormatOpenAI
}

func abortWithMidjourneyMessage(c *gin.Context, statusCode int, code int, description string) {
	c.JSON(statusCode, gin.H{
		"description": description,
		"type":        "new_api_error",
		"code":        code,
	})
	c.Abort()
	logger.LogError(c.Request.Context(), description)
}
