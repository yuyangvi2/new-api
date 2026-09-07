package dto

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

//type OpenAIError struct {
//	Message string `json:"message"`
//	Type    string `json:"type"`
//	Param   string `json:"param"`
//	Code    any    `json:"code"`
//}

type OpenAIErrorWithStatusCode struct {
	Error      types.OpenAIError `json:"error"`
	StatusCode int               `json:"status_code"`
	LocalError bool
}

type GeneralErrorResponse struct {
	Error      json.RawMessage `json:"error"`
	Message    string          `json:"message"`
	Msg        string          `json:"msg"`
	Err        string          `json:"err"`
	ErrorMsg   string          `json:"error_msg"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	Detail     string          `json:"detail,omitempty"`
	DetailData json.RawMessage `json:"-"`
	Type       string          `json:"type,omitempty"`
	Param      string          `json:"param,omitempty"`
	Code       any             `json:"code,omitempty"`
	Header     struct {
		Message string `json:"message"`
	} `json:"header"`
	Response struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"response"`
}

func (e *GeneralErrorResponse) UnmarshalJSON(data []byte) error {
	*e = GeneralErrorResponse{}
	var fields map[string]json.RawMessage
	if err := common.Unmarshal(data, &fields); err != nil {
		return err
	}

	e.Error = fields["error"]
	e.Message = jsonString(fields["message"])
	e.Msg = jsonString(fields["msg"])
	e.Err = jsonString(fields["err"])
	e.ErrorMsg = jsonString(fields["error_msg"])
	e.Metadata = fields["metadata"]
	e.Detail = jsonString(fields["detail"])
	if e.Detail == "" && len(fields["detail"]) > 0 {
		e.DetailData = fields["detail"]
	}
	e.Type = jsonString(fields["type"])
	e.Param = jsonString(fields["param"])
	if rawCode := fields["code"]; len(rawCode) > 0 {
		_ = common.Unmarshal(rawCode, &e.Code)
	}

	var headerFields map[string]json.RawMessage
	if rawHeader := fields["header"]; len(rawHeader) > 0 && common.Unmarshal(rawHeader, &headerFields) == nil {
		e.Header.Message = jsonString(headerFields["message"])
	}

	var responseFields map[string]json.RawMessage
	if rawResponse := fields["response"]; len(rawResponse) > 0 && common.Unmarshal(rawResponse, &responseFields) == nil {
		var responseErrorFields map[string]json.RawMessage
		if rawError := responseFields["error"]; len(rawError) > 0 && common.Unmarshal(rawError, &responseErrorFields) == nil {
			e.Response.Error.Message = jsonString(responseErrorFields["message"])
		}
	}

	return nil
}

func jsonString(data json.RawMessage) string {
	if common.GetJsonType(data) != "string" {
		return ""
	}
	var value string
	if err := common.Unmarshal(data, &value); err != nil {
		return ""
	}
	return value
}

func (e GeneralErrorResponse) TryToOpenAIError() *types.OpenAIError {
	var openAIError types.OpenAIError
	if len(e.Error) > 0 {
		err := common.Unmarshal(e.Error, &openAIError)
		if err == nil {
			if openAIError.Message == "" {
				openAIError.Message = upstreamErrorCodeText(openAIError.Code)
			}
			if openAIError.Message == "" {
				return nil
			}
			openAIError.Message = enrichUpstreamErrorMessage(openAIError.Message, e.Error)
			if len(openAIError.Metadata) == 0 {
				openAIError.Metadata = upstreamErrorMetadata(e.Error)
			}
			return &openAIError
		}
	}
	if e.Message != "" {
		code := e.Code
		if upstreamErrorCodeText(code) == "" {
			code = types.ErrorCodeBadResponseStatusCode
		}
		return &types.OpenAIError{
			Message: e.Message,
			Type:    e.Type,
			Param:   e.Param,
			Code:    code,
		}
	}
	return nil
}

func upstreamErrorCodeText(code any) string {
	switch value := code.(type) {
	case string:
		return strings.TrimSpace(value)
	case float64, json.Number:
		return strings.TrimSpace(fmt.Sprint(value))
	default:
		return ""
	}
}

func (e GeneralErrorResponse) ToMessage() string {
	if len(e.Error) > 0 {
		switch common.GetJsonType(e.Error) {
		case "object":
			var openAIError types.OpenAIError
			err := common.Unmarshal(e.Error, &openAIError)
			if err == nil && openAIError.Message != "" {
				return enrichUpstreamErrorMessage(openAIError.Message, e.Error)
			}
		case "string":
			var msg string
			err := common.Unmarshal(e.Error, &msg)
			if err == nil && msg != "" {
				return msg
			}
		default:
			if message := safeRawUpstreamErrorDetail(e.Error); message != "" {
				return message
			}
		}
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != "" {
		return e.Err
	}
	if e.ErrorMsg != "" {
		return e.ErrorMsg
	}
	if e.Detail != "" {
		return e.Detail
	}
	if len(e.DetailData) > 0 {
		return safeRawUpstreamErrorDetail(e.DetailData)
	}
	if e.Header.Message != "" {
		return e.Header.Message
	}
	if e.Response.Error.Message != "" {
		return e.Response.Error.Message
	}
	return ""
}

func safeRawUpstreamErrorDetail(raw json.RawMessage) string {
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return upstreamErrorDetailString(value)
}

func enrichUpstreamErrorMessage(message string, rawError json.RawMessage) string {
	var fields map[string]any
	if err := common.Unmarshal(rawError, &fields); err != nil {
		return message
	}

	details := upstreamErrorDetailStrings(fields)
	if len(details) == 0 {
		return message
	}
	detailText := strings.Join(details, "; ")
	if strings.Contains(message, detailText) {
		return message
	}
	return fmt.Sprintf("%s: %s", message, detailText)
}

func upstreamErrorDetailStrings(fields map[string]any) []string {
	details := make([]string, 0, 4)
	for _, key := range []string{"param", "detail", "details", "metadata"} {
		if value, ok := fields[key]; ok {
			if text := upstreamErrorDetailString(value); text != "" {
				details = append(details, fmt.Sprintf("%s=%s", key, text))
			}
		}
	}
	return details
}

func upstreamErrorDetailString(value any) string {
	safeValue, ok := safeUpstreamErrorDetailValue(value, 0)
	if !ok {
		return ""
	}
	switch v := safeValue.(type) {
	case string:
		return v
	default:
		data, err := common.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(data)
	}
}

func safeUpstreamErrorDetailValue(value any, depth int) (any, bool) {
	if depth > 3 {
		return nil, false
	}
	switch v := value.(type) {
	case string, bool, float64, json.Number:
		return v, true
	case nil:
		return nil, false
	case []any:
		items := make([]any, 0, min(len(v), 8))
		for index, item := range v {
			if index >= 8 {
				break
			}
			if safeItem, ok := safeUpstreamErrorDetailValue(item, depth+1); ok {
				items = append(items, safeItem)
			}
		}
		return items, len(items) > 0
	case map[string]any:
		safeFields := make(map[string]any)
		for _, key := range []string{"message", "msg", "reason", "code", "type", "param", "field", "path", "loc", "location", "help", "hint"} {
			fieldValue, exists := v[key]
			if !exists {
				continue
			}
			if safeFieldValue, ok := safeUpstreamErrorDetailValue(fieldValue, depth+1); ok {
				safeFields[key] = safeFieldValue
			}
		}
		return safeFields, len(safeFields) > 0
	default:
		return nil, false
	}
}

func upstreamErrorMetadata(rawError json.RawMessage) json.RawMessage {
	var fields map[string]any
	if err := common.Unmarshal(rawError, &fields); err != nil {
		return nil
	}
	metadata := map[string]any{}
	for _, key := range []string{"param", "detail", "details", "metadata"} {
		if value, ok := fields[key]; ok {
			if safeValue, safe := safeUpstreamErrorDetailValue(value, 0); safe {
				metadata[key] = safeValue
			}
		}
	}
	if len(metadata) == 0 {
		return nil
	}
	data, err := common.Marshal(metadata)
	if err != nil {
		return nil
	}
	return data
}
