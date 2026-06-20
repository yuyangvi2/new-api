package controller

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// setupPlaygroundContext authenticates the request via the login session
// (UserAuth middleware already ran), writes the user context, and installs a
// temporary token so downstream relay/task handlers can bill and route without
// the caller supplying an API key. Shared by chat, image and video playground
// endpoints. Returns a NewAPIError (already typed) on failure.
func setupPlaygroundContext(c *gin.Context, relayFormat types.RelayFormat) *types.NewAPIError {
	useAccessToken := c.GetBool("use_access_token")
	if useAccessToken {
		return types.NewError(errors.New("暂不支持使用 access token"), types.ErrorCodeAccessDenied, types.ErrOptionWithSkipRetry())
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, relayFormat, nil, nil)
	if err != nil {
		return types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	userId := c.GetInt("id")

	// Write user context to ensure acceptUnsetRatio is available
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		return types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
	}
	userCache.WriteContext(c)

	tempToken := &model.Token{
		UserId: userId,
		Name:   fmt.Sprintf("playground-%s", relayInfo.UsingGroup),
		Group:  relayInfo.UsingGroup,
	}
	_ = middleware.SetupContextForToken(c, tempToken)
	return nil
}

func Playground(c *gin.Context) {
	var newAPIError *types.NewAPIError

	defer func() {
		if newAPIError != nil {
			c.JSON(newAPIError.StatusCode, gin.H{
				"error": newAPIError.ToOpenAIError(),
			})
		}
	}()

	// Pick the relay format from the request path so the same handler can
	// serve both chat completions and image generation under /pg.
	relayFormat := types.RelayFormatOpenAI
	if strings.Contains(c.Request.URL.Path, "/images/generations") {
		relayFormat = types.RelayFormatOpenAIImage
	}

	if newAPIError = setupPlaygroundContext(c, relayFormat); newAPIError != nil {
		return
	}

	Relay(c, relayFormat)
}

// PlaygroundTask submits an async task (e.g. image-to-video generation) under
// the login session, mirroring Playground but routing to the task relay.
func PlaygroundTask(c *gin.Context) {
	var newAPIError *types.NewAPIError

	defer func() {
		if newAPIError != nil {
			c.JSON(newAPIError.StatusCode, gin.H{
				"error": newAPIError.ToOpenAIError(),
			})
		}
	}()

	if newAPIError = setupPlaygroundContext(c, types.RelayFormatTask); newAPIError != nil {
		return
	}

	RelayTask(c)
}

// PlaygroundTaskFetch queries an async task's status under the login session.
func PlaygroundTaskFetch(c *gin.Context) {
	var newAPIError *types.NewAPIError

	defer func() {
		if newAPIError != nil {
			c.JSON(newAPIError.StatusCode, gin.H{
				"error": newAPIError.ToOpenAIError(),
			})
		}
	}()

	if newAPIError = setupPlaygroundContext(c, types.RelayFormatTask); newAPIError != nil {
		return
	}

	RelayTaskFetch(c)
}
