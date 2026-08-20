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
	Error    json.RawMessage `json:"error"`
	Message  string          `json:"message"`
	Msg      string          `json:"msg"`
	Err      string          `json:"err"`
	ErrorMsg string          `json:"error_msg"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	Detail   string          `json:"detail,omitempty"`
	Header   struct {
		Message string `json:"message"`
	} `json:"header"`
	Response struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"response"`
}

func (e GeneralErrorResponse) TryToOpenAIError() *types.OpenAIError {
	var openAIError types.OpenAIError
	if len(e.Error) > 0 {
		err := common.Unmarshal(e.Error, &openAIError)
		if err == nil && openAIError.Message != "" {
			openAIError.Message = enrichUpstreamErrorMessage(openAIError.Message, e.Error)
			if len(openAIError.Metadata) == 0 {
				openAIError.Metadata = upstreamErrorMetadata(e.Error)
			}
			return &openAIError
		}
	}
	return nil
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
			return string(e.Error)
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
	if e.Header.Message != "" {
		return e.Header.Message
	}
	if e.Response.Error.Message != "" {
		return e.Response.Error.Message
	}
	return ""
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
	switch v := value.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		data, err := common.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(data)
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
			metadata[key] = value
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
