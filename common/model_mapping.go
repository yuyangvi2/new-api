package common

import (
	"errors"
	"fmt"
	"strings"
)

// ResolveStrictModelMapping follows a channel model-mapping chain and requires
// the requested model to have at least one mapping entry.
func ResolveStrictModelMapping(modelName string, mappingJSON string) (string, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return "", errors.New("model is required")
	}

	var mappings map[string]string
	if mappingJSON == "" || mappingJSON == "{}" || UnmarshalJsonStr(mappingJSON, &mappings) != nil {
		return "", fmt.Errorf("model mapping must include %s", modelName)
	}

	current := modelName
	visited := map[string]bool{current: true}
	mapped := false
	for {
		next, ok := mappings[current]
		if !ok || strings.TrimSpace(next) == "" {
			break
		}
		next = strings.TrimSpace(next)
		if visited[next] {
			return "", errors.New("model mapping contains a cycle")
		}
		visited[next] = true
		current = next
		mapped = true
	}
	if !mapped {
		return "", fmt.Errorf("model mapping must include %s", modelName)
	}
	return current, nil
}
