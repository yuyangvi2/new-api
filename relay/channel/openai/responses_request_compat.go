package openai

import (
	"fmt"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func normalizeResponsesRequestCompatibility(request *dto.OpenAIResponsesRequest) error {
	if request == nil {
		return nil
	}

	tools, changed, err := normalizeResponsesTools(request.Tools)
	if err != nil {
		return err
	}
	if changed {
		request.Tools = tools
	}

	input, changed, err := normalizeResponsesReasoningInput(request.Input)
	if err != nil {
		return err
	}
	if changed {
		request.Input = input
	}
	return nil
}

func normalizeResponsesTools(raw []byte) ([]byte, bool, error) {
	if len(raw) == 0 {
		return raw, false, nil
	}

	var tools []any
	if err := common.Unmarshal(raw, &tools); err != nil {
		return nil, false, fmt.Errorf("decode Responses tools: %w", err)
	}

	changed := false
	for _, tool := range tools {
		if normalizeResponsesToolEntry(tool) {
			changed = true
		}
	}
	if !changed {
		return raw, false, nil
	}

	normalized, err := common.Marshal(tools)
	if err != nil {
		return nil, false, fmt.Errorf("encode normalized Responses tools: %w", err)
	}
	return normalized, true, nil
}

func normalizeResponsesToolEntry(value any) bool {
	tool, ok := value.(map[string]any)
	if !ok {
		return false
	}

	changed := false
	if toolType, _ := tool["type"].(string); toolType == "function" {
		if parameters, ok := tool["parameters"].(map[string]any); ok {
			changed = normalizeResponsesFunctionParameters(parameters)
		}
	}
	if nestedTools, ok := tool["tools"].([]any); ok {
		for _, nestedTool := range nestedTools {
			if normalizeResponsesToolEntry(nestedTool) {
				changed = true
			}
		}
	}
	return changed
}

func normalizeResponsesFunctionParameters(schema map[string]any) bool {
	if schema == nil {
		return false
	}

	changed := false
	properties, _ := schema["properties"].(map[string]any)
	if properties == nil {
		properties = make(map[string]any)
	}
	required := responsesSchemaRequiredSet(schema["required"])

	if branches, exists := schema["allOf"]; exists {
		delete(schema, "allOf")
		changed = true
		for _, branch := range responsesSchemaBranches(branches) {
			normalizeResponsesFunctionParameters(branch)
			mergeResponsesSchemaProperties(properties, branch)
			for name := range responsesSchemaRequiredSet(branch["required"]) {
				required[name] = struct{}{}
			}
		}
	}

	for _, keyword := range []string{"oneOf", "anyOf"} {
		rawBranches, exists := schema[keyword]
		if !exists {
			continue
		}
		delete(schema, keyword)
		changed = true

		branches := responsesSchemaBranches(rawBranches)
		var commonRequired map[string]struct{}
		for i, branch := range branches {
			normalizeResponsesFunctionParameters(branch)
			mergeResponsesSchemaProperties(properties, branch)
			branchRequired := responsesSchemaRequiredSet(branch["required"])
			if i == 0 {
				commonRequired = branchRequired
				continue
			}
			for name := range commonRequired {
				if _, ok := branchRequired[name]; !ok {
					delete(commonRequired, name)
				}
			}
		}
		for name := range commonRequired {
			required[name] = struct{}{}
		}
	}

	if !changed {
		return false
	}
	schema["type"] = "object"
	if len(properties) > 0 {
		schema["properties"] = properties
	}
	if len(required) == 0 {
		delete(schema, "required")
	} else {
		requiredNames := make([]string, 0, len(required))
		for name := range required {
			requiredNames = append(requiredNames, name)
		}
		sort.Strings(requiredNames)
		schema["required"] = requiredNames
	}
	return true
}

func responsesSchemaBranches(value any) []map[string]any {
	rawBranches, ok := value.([]any)
	if !ok {
		return nil
	}
	branches := make([]map[string]any, 0, len(rawBranches))
	for _, rawBranch := range rawBranches {
		if branch, ok := rawBranch.(map[string]any); ok {
			branches = append(branches, branch)
		}
	}
	return branches
}

func mergeResponsesSchemaProperties(target map[string]any, source map[string]any) {
	properties, _ := source["properties"].(map[string]any)
	for name, property := range properties {
		if _, exists := target[name]; !exists {
			target[name] = property
		}
	}
}

func responsesSchemaRequiredSet(value any) map[string]struct{} {
	required := make(map[string]struct{})
	switch items := value.(type) {
	case []any:
		for _, item := range items {
			if name, ok := item.(string); ok {
				required[name] = struct{}{}
			}
		}
	case []string:
		for _, name := range items {
			required[name] = struct{}{}
		}
	}
	return required
}

func normalizeResponsesReasoningInput(raw []byte) ([]byte, bool, error) {
	if len(raw) == 0 {
		return raw, false, nil
	}

	var input any
	if err := common.Unmarshal(raw, &input); err != nil {
		return nil, false, fmt.Errorf("decode Responses input: %w", err)
	}

	changed := false
	switch value := input.(type) {
	case []any:
		for _, item := range value {
			if normalizeResponsesReasoningItem(item) {
				changed = true
			}
		}
	case map[string]any:
		changed = normalizeResponsesReasoningItem(value)
	}
	if !changed {
		return raw, false, nil
	}

	normalized, err := common.Marshal(input)
	if err != nil {
		return nil, false, fmt.Errorf("encode normalized Responses input: %w", err)
	}
	return normalized, true, nil
}

func normalizeResponsesReasoningItem(value any) bool {
	item, ok := value.(map[string]any)
	if !ok || item["type"] != "reasoning" {
		return false
	}
	content, exists := item["content"]
	if !exists {
		return false
	}
	if _, hasSummary := item["summary"]; !hasSummary {
		item["summary"] = content
	}
	delete(item, "content")
	return true
}
