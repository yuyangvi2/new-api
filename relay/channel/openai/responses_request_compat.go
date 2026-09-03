package openai

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const (
	responsesNamespaceToolNamesContextKey = "openai_responses_namespace_tool_names"
	responsesFunctionNameMaxLength        = 64
)

type responsesNamespaceToolName struct {
	Namespace string
	Name      string
}

func normalizeResponsesRequestCompatibility(request *dto.OpenAIResponsesRequest) (map[string]responsesNamespaceToolName, error) {
	if request == nil {
		return nil, nil
	}

	tools, namespaceNames, changed, err := normalizeResponsesTools(request.Tools)
	if err != nil {
		return nil, err
	}
	if changed {
		request.Tools = tools
	}

	input, changed, err := normalizeResponsesInput(request.Input, namespaceNames)
	if err != nil {
		return nil, err
	}
	if changed {
		request.Input = input
	}

	toolChoice, changed, err := normalizeResponsesToolChoice(request.ToolChoice, namespaceNames)
	if err != nil {
		return nil, err
	}
	if changed {
		request.ToolChoice = toolChoice
	}
	return namespaceNames, nil
}

func normalizeResponsesTools(raw []byte) ([]byte, map[string]responsesNamespaceToolName, bool, error) {
	if len(raw) == 0 {
		return raw, nil, false, nil
	}

	var tools []any
	if err := common.Unmarshal(raw, &tools); err != nil {
		return nil, nil, false, fmt.Errorf("decode Responses tools: %w", err)
	}

	topLevelNames := make(map[string]struct{})
	for _, value := range tools {
		tool, ok := value.(map[string]any)
		if !ok || strings.TrimSpace(responsesStringValue(tool["type"])) == "namespace" {
			continue
		}
		name := strings.TrimSpace(responsesStringValue(tool["name"]))
		if name != "" {
			topLevelNames[name] = struct{}{}
		}
	}

	namespaceNames := make(map[string]responsesNamespaceToolName)
	for _, value := range tools {
		tool, ok := value.(map[string]any)
		if !ok || strings.TrimSpace(responsesStringValue(tool["type"])) != "namespace" {
			continue
		}
		namespace := strings.TrimSpace(responsesStringValue(tool["name"]))
		if namespace == "" {
			return nil, nil, false, fmt.Errorf("Responses namespace tool name must not be empty")
		}
		children := responsesNamespaceChildren(tool)
		if len(children) == 0 {
			return nil, nil, false, fmt.Errorf("Responses namespace tool %q must contain function tools", namespace)
		}
		for _, rawChild := range children {
			child, ok := rawChild.(map[string]any)
			if !ok || strings.TrimSpace(responsesStringValue(child["type"])) != "function" {
				return nil, nil, false, fmt.Errorf("Responses namespace tool %q contains a non-function child", namespace)
			}
			name := strings.TrimSpace(responsesStringValue(child["name"]))
			if name == "" {
				return nil, nil, false, fmt.Errorf("Responses namespace tool %q contains a function with an empty name", namespace)
			}
			flatName := flattenResponsesNamespaceToolName(namespace, name)
			if _, exists := topLevelNames[flatName]; exists {
				return nil, nil, false, fmt.Errorf("namespace tool %q/%q flattens to %q which conflicts with a top-level tool of the same name", namespace, name, flatName)
			}
			entry := responsesNamespaceToolName{Namespace: namespace, Name: name}
			if previous, exists := namespaceNames[flatName]; exists && previous != entry {
				return nil, nil, false, fmt.Errorf("namespace tools %q/%q and %q/%q both flatten to %q", previous.Namespace, previous.Name, namespace, name, flatName)
			}
			namespaceNames[flatName] = entry
		}
	}

	changed := false
	normalizedTools := make([]any, 0, len(tools)+len(namespaceNames))
	seenNamespaceTools := make(map[string]struct{})
	for _, value := range tools {
		tool, ok := value.(map[string]any)
		if !ok {
			normalizedTools = append(normalizedTools, value)
			continue
		}
		if strings.TrimSpace(responsesStringValue(tool["type"])) != "namespace" {
			if normalizeResponsesToolEntry(tool) {
				changed = true
			}
			normalizedTools = append(normalizedTools, tool)
			continue
		}

		changed = true
		namespace := strings.TrimSpace(responsesStringValue(tool["name"]))
		for _, rawChild := range responsesNamespaceChildren(tool) {
			child := rawChild.(map[string]any)
			name := strings.TrimSpace(responsesStringValue(child["name"]))
			flatName := flattenResponsesNamespaceToolName(namespace, name)
			if _, exists := seenNamespaceTools[flatName]; exists {
				continue
			}
			seenNamespaceTools[flatName] = struct{}{}
			flatChild := make(map[string]any, len(child))
			for key, childValue := range child {
				flatChild[key] = childValue
			}
			flatChild["name"] = flatName
			normalizeResponsesToolEntry(flatChild)
			normalizedTools = append(normalizedTools, flatChild)
		}
	}
	if !changed {
		return raw, nil, false, nil
	}

	normalized, err := common.Marshal(normalizedTools)
	if err != nil {
		return nil, nil, false, fmt.Errorf("encode normalized Responses tools: %w", err)
	}
	return normalized, namespaceNames, true, nil
}

func responsesNamespaceChildren(tool map[string]any) []any {
	if children, ok := tool["tools"].([]any); ok && len(children) > 0 {
		return children
	}
	children, _ := tool["children"].([]any)
	return children
}

func flattenResponsesNamespaceToolName(namespace string, name string) string {
	fullName := namespace + "__" + name
	if len(fullName) <= responsesFunctionNameMaxLength {
		return fullName
	}

	digest := sha256.Sum256([]byte(fullName))
	suffix := "__" + hex.EncodeToString(digest[:4])
	prefixLimit := responsesFunctionNameMaxLength - len(suffix)
	var prefix strings.Builder
	for _, character := range fullName {
		encoded := string(character)
		if prefix.Len()+len(encoded) > prefixLimit {
			break
		}
		prefix.WriteString(encoded)
	}
	return prefix.String() + suffix
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

func normalizeResponsesInput(raw []byte, namespaceNames map[string]responsesNamespaceToolName) ([]byte, bool, error) {
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
			if normalizeResponsesInputValue(item, namespaceNames) {
				changed = true
			}
		}
	case map[string]any:
		changed = normalizeResponsesInputValue(value, namespaceNames)
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

func normalizeResponsesInputValue(value any, namespaceNames map[string]responsesNamespaceToolName) bool {
	changed := false
	switch item := value.(type) {
	case []any:
		for _, child := range item {
			changed = normalizeResponsesInputValue(child, namespaceNames) || changed
		}
	case map[string]any:
		if normalizeResponsesReasoningItem(item) {
			changed = true
		}
		if strings.TrimSpace(responsesStringValue(item["type"])) == "function_call" && rewriteResponsesNamespaceCall(item, namespaceNames) {
			changed = true
		}
		for _, child := range item {
			changed = normalizeResponsesInputValue(child, namespaceNames) || changed
		}
	}
	return changed
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

func normalizeResponsesToolChoice(raw []byte, namespaceNames map[string]responsesNamespaceToolName) ([]byte, bool, error) {
	if len(raw) == 0 || len(namespaceNames) == 0 {
		return raw, false, nil
	}
	var choice any
	if err := common.Unmarshal(raw, &choice); err != nil {
		return nil, false, fmt.Errorf("decode Responses tool_choice: %w", err)
	}
	choiceObject, ok := choice.(map[string]any)
	if !ok {
		return raw, false, nil
	}

	changed := false
	if strings.TrimSpace(responsesStringValue(choiceObject["type"])) == "namespace" {
		choice = "auto"
		changed = true
	} else if rewriteResponsesNamespaceCall(choiceObject, namespaceNames) {
		changed = true
	}
	if !changed {
		return raw, false, nil
	}
	normalized, err := common.Marshal(choice)
	if err != nil {
		return nil, false, fmt.Errorf("encode normalized Responses tool_choice: %w", err)
	}
	return normalized, true, nil
}

func rewriteResponsesNamespaceCall(item map[string]any, namespaceNames map[string]responsesNamespaceToolName) bool {
	namespace := strings.TrimSpace(responsesStringValue(item["namespace"]))
	name := strings.TrimSpace(responsesStringValue(item["name"]))
	if namespace == "" || name == "" {
		return false
	}
	flatName := flattenResponsesNamespaceToolName(namespace, name)
	entry, exists := namespaceNames[flatName]
	if !exists || entry.Namespace != namespace || entry.Name != name {
		return false
	}
	item["name"] = flatName
	delete(item, "namespace")
	return true
}

func restoreResponsesNamespaceCalls(payload []byte, namespaceNames map[string]responsesNamespaceToolName) ([]byte, bool, error) {
	if len(payload) == 0 || len(namespaceNames) == 0 {
		return payload, false, nil
	}
	var value any
	if err := common.Unmarshal(payload, &value); err != nil {
		return payload, false, err
	}
	if !restoreResponsesNamespaceValue(value, namespaceNames) {
		return payload, false, nil
	}
	restored, err := common.Marshal(value)
	if err != nil {
		return payload, false, err
	}
	return restored, true, nil
}

func restoreResponsesNamespaceValue(value any, namespaceNames map[string]responsesNamespaceToolName) bool {
	changed := false
	switch item := value.(type) {
	case []any:
		for _, child := range item {
			changed = restoreResponsesNamespaceValue(child, namespaceNames) || changed
		}
	case map[string]any:
		if strings.TrimSpace(responsesStringValue(item["type"])) == "function_call" {
			flatName := strings.TrimSpace(responsesStringValue(item["name"]))
			if entry, exists := namespaceNames[flatName]; exists {
				item["name"] = entry.Name
				item["namespace"] = entry.Namespace
				changed = true
			}
		}
		for _, child := range item {
			changed = restoreResponsesNamespaceValue(child, namespaceNames) || changed
		}
	}
	return changed
}

func responsesStringValue(value any) string {
	text, _ := value.(string)
	return text
}
