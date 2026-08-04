package gemini

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const maxGeminiCandidateCount = 8

// validateGeminiCompatibleOpenAIRequest validates OpenAI-format requests for Gemini compatibility.
// This catches fields that Gemini doesn't support to avoid silent drops or unexpected behavior.
func validateGeminiCompatibleOpenAIRequest(request *dto.GeneralOpenAIRequest) error {
	if request == nil {
		return errors.New("request is nil")
	}
	if request.MaxTokens != nil && *request.MaxTokens == 0 {
		return errors.New("max_tokens must be greater than 0")
	}
	if request.MaxCompletionTokens != nil && *request.MaxCompletionTokens == 0 {
		return errors.New("max_completion_tokens must be greater than 0")
	}
	if request.N != nil && (*request.N < 1 || *request.N > maxGeminiCandidateCount) {
		return fmt.Errorf("n must be between 1 and %d for Gemini", maxGeminiCandidateCount)
	}
	if request.LogProbs != nil && *request.LogProbs {
		return errors.New("logprobs is not supported for Gemini chat compatibility")
	}
	if request.TopLogProbs != nil {
		return errors.New("top_logprobs is not supported for Gemini chat compatibility")
	}
	if err := validateGeminiExtraBody(request); err != nil {
		return err
	}
	if err := validateOpenAIToolResultIDs(request.Messages); err != nil {
		return err
	}
	if err := validateGeminiToolSchemas(request.Tools); err != nil {
		return err
	}
	if err := validateGeminiResponseFormat(request.ResponseFormat); err != nil {
		return err
	}
	return nil
}

// validateGeminiCompatibleClaudeRequest validates Claude-format requests for Gemini compatibility.
// Checks for conflicting thinking configuration between Claude's thinking field and extra_body.google.thinking_config.
func validateGeminiCompatibleClaudeRequest(request *dto.ClaudeRequest) error {
	if request == nil {
		return errors.New("request is nil")
	}
	if request.MaxTokens != nil && *request.MaxTokens == 0 {
		return errors.New("max_tokens must be greater than 0")
	}
	if request.Thinking == nil || len(request.ExtraBody) == 0 {
		return nil
	}
	var extraBody map[string]any
	if err := common.Unmarshal(request.ExtraBody, &extraBody); err != nil {
		return fmt.Errorf("invalid extra_body: %w", err)
	}
	googleBody, ok := extraBody["google"].(map[string]any)
	if !ok {
		return nil
	}
	if _, ok := googleBody["thinking_config"]; ok {
		return errors.New("thinking cannot be used with extra_body.google.thinking_config")
	}
	return nil
}

func validateGeminiExtraBody(request *dto.GeneralOpenAIRequest) error {
	if len(request.ExtraBody) == 0 {
		return nil
	}
	var extraBody map[string]any
	if err := common.Unmarshal(request.ExtraBody, &extraBody); err != nil {
		return fmt.Errorf("invalid extra_body: %w", err)
	}
	googleRaw, ok := extraBody["google"]
	if !ok {
		return nil
	}
	googleBody, ok := googleRaw.(map[string]any)
	if !ok {
		return errors.New("extra_body.google must be an object")
	}
	hasThinkingConfig := false
	for key := range googleBody {
		switch key {
		case "thinking_config":
			hasThinkingConfig = true
		case "thinkingConfig":
			return errors.New("extra_body.google.thinkingConfig is not supported, use extra_body.google.thinking_config instead")
		case "image_config":
		case "imageConfig":
			return errors.New("extra_body.google.imageConfig is not supported, use extra_body.google.image_config instead")
		case "cached_content":
		default:
			return fmt.Errorf("extra_body.google.%s is not supported", key)
		}
	}
	if hasThinkingConfig && strings.TrimSpace(request.ReasoningEffort) != "" {
		return errors.New("reasoning_effort cannot be used with extra_body.google.thinking_config")
	}
	if cachedContent, ok := googleBody["cached_content"]; ok {
		cachedContentStr, ok := cachedContent.(string)
		if !ok || strings.TrimSpace(cachedContentStr) == "" {
			return errors.New("extra_body.google.cached_content must be a non-empty string")
		}
	}
	return nil
}

func validateOpenAIToolResultIDs(messages []dto.Message) error {
	available := make(map[string]bool)
	for _, message := range messages {
		if message.Role == "assistant" {
			for _, call := range message.ParseToolCalls() {
				if strings.TrimSpace(call.ID) != "" {
					available[call.ID] = true
				}
			}
			continue
		}
		if message.Role != "tool" && message.Role != "function" {
			continue
		}
		// Tool/function messages should have a tool_call_id
		trimmedID := strings.TrimSpace(message.ToolCallId)
		if trimmedID == "" {
			// Allow empty tool_call_id for backward compatibility, but it's not ideal
			// In strict mode, this should be an error
			continue
		}
		if !available[trimmedID] {
			return fmt.Errorf("tool result references unknown tool_call_id %q", trimmedID)
		}
		delete(available, trimmedID)
	}
	return nil
}

func validateGeminiToolSchemas(tools []dto.ToolCallRequest) error {
	for _, tool := range tools {
		if tool.Type != "" && tool.Type != "function" {
			continue
		}
		if err := validateJSONSchema(tool.Function.Parameters, "tools[].function.parameters"); err != nil {
			return err
		}
	}
	return nil
}

func validateGeminiResponseFormat(format *dto.ResponseFormat) error {
	if format == nil || format.Type == "" {
		return nil
	}
	switch format.Type {
	case "json_object":
		return nil
	case "json_schema":
		if len(format.JsonSchema) == 0 {
			return errors.New("response_format.json_schema is required")
		}
		var jsonSchema dto.FormatJsonSchema
		if err := common.Unmarshal(format.JsonSchema, &jsonSchema); err != nil {
			return fmt.Errorf("invalid response_format.json_schema: %w", err)
		}
		if strings.TrimSpace(jsonSchema.Name) == "" {
			return errors.New("response_format.json_schema.name is required")
		}
		return validateJSONSchema(jsonSchema.Schema, "response_format.json_schema.schema")
	default:
		return fmt.Errorf("response_format.type %q is not supported for Gemini", format.Type)
	}
}

// validateJSONSchema validates a JSON Schema object for Gemini compatibility.
// Gemini supports a subset of JSON Schema: type, enum, properties, items, anyOf, required.
// Unsupported features: $ref, allOf, oneOf, not, patternProperties, dependencies, etc.
// See: https://ai.google.dev/gemini-api/docs/json-mode
func validateJSONSchema(schema any, path string) error {
	if schema == nil {
		return nil
	}
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return fmt.Errorf("%s must be an object", path)
	}
	if rawType, ok := schemaMap["type"]; ok {
		if err := validateJSONSchemaType(rawType, path+".type"); err != nil {
			return err
		}
	}
	if enumValue, ok := schemaMap["enum"]; ok {
		enumLen, ok := jsonArrayLen(enumValue)
		if !ok {
			return fmt.Errorf("%s.enum must be an array", path)
		}
		if enumLen == 0 {
			return fmt.Errorf("%s.enum must not be empty", path)
		}
	}
	if props, ok := schemaMap["properties"]; ok {
		propsMap, ok := props.(map[string]any)
		if !ok {
			return fmt.Errorf("%s.properties must be an object", path)
		}
		for name, propSchema := range propsMap {
			if err := validateJSONSchema(propSchema, path+".properties."+name); err != nil {
				return err
			}
		}
	}
	if items, ok := schemaMap["items"]; ok {
		if err := validateJSONSchema(items, path+".items"); err != nil {
			return err
		}
	}
	if anyOf, ok := schemaMap["anyOf"]; ok {
		items, ok := anyOf.([]any)
		if !ok {
			return fmt.Errorf("%s.anyOf must be an array", path)
		}
		if len(items) == 0 {
			return fmt.Errorf("%s.anyOf must not be empty", path)
		}
		for idx, item := range items {
			if err := validateJSONSchema(item, fmt.Sprintf("%s.anyOf[%d]", path, idx)); err != nil {
				return err
			}
		}
	}
	if required, ok := schemaMap["required"]; ok {
		if _, ok := jsonStringArrayLen(required); !ok {
			return fmt.Errorf("%s.required must be an array", path)
		}
	}
	return nil
}

func validateJSONSchemaType(rawType any, path string) error {
	switch typed := rawType.(type) {
	case string:
		if !isValidJSONSchemaType(typed) {
			return fmt.Errorf("%s has unsupported value %q", path, typed)
		}
	case []any:
		if len(typed) == 0 {
			return fmt.Errorf("%s must not be empty", path)
		}
		for _, item := range typed {
			itemType, ok := item.(string)
			if !ok || !isValidJSONSchemaType(itemType) {
				return fmt.Errorf("%s has unsupported value %v", path, item)
			}
		}
	case []string:
		if len(typed) == 0 {
			return fmt.Errorf("%s must not be empty", path)
		}
		for _, item := range typed {
			if !isValidJSONSchemaType(item) {
				return fmt.Errorf("%s has unsupported value %v", path, item)
			}
		}
	default:
		return fmt.Errorf("%s must be a string or array of strings", path)
	}
	return nil
}

func jsonArrayLen(value any) (int, bool) {
	switch typed := value.(type) {
	case []any:
		return len(typed), true
	case []string:
		return len(typed), true
	case []bool:
		return len(typed), true
	case []int:
		return len(typed), true
	case []float64:
		return len(typed), true
	default:
		return 0, false
	}
}

func jsonStringArrayLen(value any) (int, bool) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if _, ok := item.(string); !ok {
				return 0, false
			}
		}
		return len(typed), true
	case []string:
		return len(typed), true
	default:
		return 0, false
	}
}

func isValidJSONSchemaType(schemaType string) bool {
	switch strings.ToLower(strings.TrimSpace(schemaType)) {
	case "object", "array", "string", "integer", "number", "boolean", "null":
		return true
	default:
		return false
	}
}

func isOpenAIToolChoiceNone(toolChoice any) bool {
	switch choice := toolChoice.(type) {
	case string:
		return choice == "none"
	case map[string]any:
		return choice["type"] == "none"
	default:
		return false
	}
}

func geminiResponseHasFunctionCall(response *dto.GeminiChatResponse) bool {
	if response == nil {
		return false
	}
	for _, candidate := range response.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				return true
			}
		}
	}
	return false
}
