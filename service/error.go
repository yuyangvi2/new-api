package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/types"
)

func MidjourneyErrorWrapper(code int, desc string) *dto.MidjourneyResponse {
	return &dto.MidjourneyResponse{
		Code:        code,
		Description: desc,
	}
}

func MidjourneyErrorWithStatusCodeWrapper(code int, desc string, statusCode int) *dto.MidjourneyResponseWithStatusCode {
	return &dto.MidjourneyResponseWithStatusCode{
		StatusCode: statusCode,
		Response:   *MidjourneyErrorWrapper(code, desc),
	}
}

//// OpenAIErrorWrapper wraps an error into an OpenAIErrorWithStatusCode
//func OpenAIErrorWrapper(err error, code string, statusCode int) *dto.OpenAIErrorWithStatusCode {
//	text := err.Error()
//	lowerText := strings.ToLower(text)
//	if !strings.HasPrefix(lowerText, "get file base64 from url") && !strings.HasPrefix(lowerText, "mime type is not supported") {
//		if strings.Contains(lowerText, "post") || strings.Contains(lowerText, "dial") || strings.Contains(lowerText, "http") {
//			common.SysLog(fmt.Sprintf("error: %s", text))
//			text = "请求上游地址失败"
//		}
//	}
//	openAIError := dto.OpenAIError{
//		Message: text,
//		Type:    "new_api_error",
//		Code:    code,
//	}
//	return &dto.OpenAIErrorWithStatusCode{
//		Error:      openAIError,
//		StatusCode: statusCode,
//	}
//}
//
//func OpenAIErrorWrapperLocal(err error, code string, statusCode int) *dto.OpenAIErrorWithStatusCode {
//	openaiErr := OpenAIErrorWrapper(err, code, statusCode)
//	openaiErr.LocalError = true
//	return openaiErr
//}

func ClaudeErrorWrapper(err error, code string, statusCode int) *dto.ClaudeErrorWithStatusCode {
	text := err.Error()
	lowerText := strings.ToLower(text)
	if !strings.HasPrefix(lowerText, "get file base64 from url") {
		if strings.Contains(lowerText, "post") || strings.Contains(lowerText, "dial") || strings.Contains(lowerText, "http") {
			common.SysLog(fmt.Sprintf("error: %s", text))
			text = "请求上游地址失败"
		}
	}
	claudeError := types.ClaudeError{
		Message: text,
		Type:    "new_api_error",
	}
	return &dto.ClaudeErrorWithStatusCode{
		Error:      claudeError,
		StatusCode: statusCode,
	}
}

func ClaudeErrorWrapperLocal(err error, code string, statusCode int) *dto.ClaudeErrorWithStatusCode {
	claudeErr := ClaudeErrorWrapper(err, code, statusCode)
	claudeErr.LocalError = true
	return claudeErr
}

func RelayErrorHandler(ctx context.Context, resp *http.Response, showBodyWhenFail bool) (newApiErr *types.NewAPIError) {
	newApiErr = types.InitOpenAIError(types.ErrorCodeBadResponseStatusCode, resp.StatusCode)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	CloseResponseBodyGracefully(resp)
	var errResponse dto.GeneralErrorResponse
	responseBodyText := string(responseBody)
	responseBodyPreview := common.LocalLogPreview(responseBodyText)
	safeResponseBodyPreview := common.MaskSensitiveInfo(responseBodyPreview)
	buildErrWithBody := func(message string) error {
		if message == "" {
			return fmt.Errorf("bad response status code %d, body: %s", resp.StatusCode, responseBodyText)
		}
		return fmt.Errorf("bad response status code %d, message: %s, body: %s", resp.StatusCode, message, responseBodyText)
	}

	err = common.Unmarshal(responseBody, &errResponse)
	if err != nil {
		if showBodyWhenFail {
			newApiErr.Err = buildErrWithBody("")
		} else {
			logger.LogError(ctx, fmt.Sprintf("bad response status code %d, body: %s", resp.StatusCode, safeResponseBodyPreview))
			newApiErr.Err = fmt.Errorf("bad response status code %d", resp.StatusCode)
		}
		return
	}

	// General format error (OpenAI, Anthropic, Gemini, etc.). This also
	// preserves providers that put message/type/param/code at the top level.
	if oaiError := errResponse.TryToOpenAIError(); oaiError != nil {
		safeError := sanitizeUpstreamOpenAIError(*oaiError, resp.StatusCode)
		newApiErr = types.WithOpenAIError(safeError, resp.StatusCode)
		if showBodyWhenFail {
			newApiErr.Err = buildErrWithBody(newApiErr.Error())
		}
		return
	}
	message := strings.TrimSpace(errResponse.ToMessage())
	if message == "" {
		logger.LogError(ctx, fmt.Sprintf("upstream returned status %d without error details, body: %s", resp.StatusCode, safeResponseBodyPreview))
		message = "Upstream returned an error without details"
	}
	safeError := sanitizeUpstreamOpenAIError(types.OpenAIError{
		Message: message,
		Type:    string(types.ErrorCodeBadResponseStatusCode),
		Code:    types.ErrorCodeBadResponseStatusCode,
	}, resp.StatusCode)
	newApiErr = types.WithOpenAIError(safeError, resp.StatusCode)
	if showBodyWhenFail {
		newApiErr.Err = buildErrWithBody(newApiErr.Error())
	}
	return
}

func sanitizeUpstreamOpenAIError(upstreamError types.OpenAIError, statusCode int) types.OpenAIError {
	descriptor := strings.ToLower(fmt.Sprintf("%s %s %v", upstreamError.Message, upstreamError.Type, upstreamError.Code))
	isPolicyError := strings.Contains(descriptor, "terms of use violation") ||
		strings.Contains(descriptor, "safety system") ||
		strings.Contains(descriptor, "content policy") ||
		strings.Contains(descriptor, "prompt blocked")

	if !isPolicyError && (statusCode == http.StatusUnauthorized ||
		strings.Contains(descriptor, "invalid bearer") ||
		strings.Contains(descriptor, "invalid api key") ||
		strings.Contains(descriptor, "invalid_api_key") ||
		strings.Contains(descriptor, "authentication failed") ||
		strings.Contains(descriptor, "not authorized to make this call")) {
		upstreamError.Message = "Upstream authentication failed, please contact administrator"
		upstreamError.Type = "upstream_error"
		upstreamError.Param = ""
		upstreamError.Code = "upstream_authentication_failed"
		upstreamError.Metadata = nil
		return upstreamError
	}

	if strings.Contains(descriptor, "insufficient account balance") ||
		strings.Contains(descriptor, "insufficient balance") ||
		strings.Contains(descriptor, "no available accounts") ||
		strings.Contains(descriptor, "no healthy upstream account") ||
		strings.Contains(descriptor, "all available accounts exhausted") {
		upstreamError.Message = "Upstream service temporarily unavailable, please contact administrator"
		upstreamError.Type = "upstream_error"
		upstreamError.Param = ""
		upstreamError.Code = "upstream_account_unavailable"
		upstreamError.Metadata = nil
		return upstreamError
	}

	upstreamError.Message = common.MaskSensitiveInfo(strings.TrimSpace(upstreamError.Message))
	upstreamError.Param = common.MaskSensitiveInfo(upstreamError.Param)
	if len(upstreamError.Metadata) > 0 {
		maskedMetadata := common.MaskSensitiveInfo(string(upstreamError.Metadata))
		var metadataValue any
		if common.Unmarshal([]byte(maskedMetadata), &metadataValue) == nil {
			upstreamError.Metadata = json.RawMessage(maskedMetadata)
		} else {
			upstreamError.Metadata = nil
		}
	}
	return upstreamError
}

func ResetStatusCode(newApiErr *types.NewAPIError, statusCodeMappingStr string) {
	if newApiErr == nil {
		return
	}
	if statusCodeMappingStr == "" || statusCodeMappingStr == "{}" {
		return
	}
	statusCodeMapping := make(map[string]any)
	err := common.Unmarshal([]byte(statusCodeMappingStr), &statusCodeMapping)
	if err != nil {
		return
	}
	if newApiErr.StatusCode == http.StatusOK {
		return
	}
	codeStr := strconv.Itoa(newApiErr.StatusCode)
	if value, ok := statusCodeMapping[codeStr]; ok {
		intCode, ok := parseStatusCodeMappingValue(value)
		if !ok {
			return
		}
		newApiErr.StatusCode = intCode
	}
}

func parseStatusCodeMappingValue(value any) (int, bool) {
	switch v := value.(type) {
	case string:
		if v == "" {
			return 0, false
		}
		statusCode, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return statusCode, true
	case float64:
		if v != math.Trunc(v) {
			return 0, false
		}
		return int(v), true
	case int:
		return v, true
	case json.Number:
		statusCode, err := strconv.Atoi(v.String())
		if err != nil {
			return 0, false
		}
		return statusCode, true
	default:
		return 0, false
	}
}

func TaskErrorWrapperLocal(err error, code string, statusCode int) *dto.TaskError {
	openaiErr := TaskErrorWrapper(err, code, statusCode)
	openaiErr.LocalError = true
	return openaiErr
}

func TaskErrorWrapper(err error, code string, statusCode int) *dto.TaskError {
	text := err.Error()
	lowerText := strings.ToLower(text)
	if strings.Contains(lowerText, "post") || strings.Contains(lowerText, "dial") || strings.Contains(lowerText, "http") {
		common.SysLog(fmt.Sprintf("error: %s", text))
		//text = "请求上游地址失败"
		text = common.MaskSensitiveInfo(text)
	}
	//避免暴露内部错误
	taskError := &dto.TaskError{
		Code:       code,
		Message:    text,
		StatusCode: statusCode,
		Error:      err,
	}

	return taskError
}

// TaskErrorFromAPIError 将 PreConsumeBilling 返回的 NewAPIError 转换为 TaskError。
func TaskErrorFromAPIError(apiErr *types.NewAPIError) *dto.TaskError {
	if apiErr == nil {
		return nil
	}
	return &dto.TaskError{
		Code:       string(apiErr.GetErrorCode()),
		Message:    apiErr.Error(),
		StatusCode: apiErr.StatusCode,
		Error:      apiErr,
	}
}
