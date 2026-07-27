package common

import "strings"

var (
	// OpenAIResponseOnlyModels is a list of models that are only available for OpenAI responses.
	OpenAIResponseOnlyModels = []string{
		"o3-pro",
		"o3-deep-research",
		"o4-mini-deep-research",
	}
	ImageGenerationModels = []string{
		"dall-e-3",
		"dall-e-2",
		"gpt-image-1",
		"prefix:imagen-",
		"flux-",
		"flux.1-",
		"exact:grok-imagine",
		"grok-imagine-image",
		"grok-2-image",
	}
	OpenAIVideoGenerationModels = []string{
		"grok-imagine-video",
	}
	OpenAITextModels = []string{
		"gpt-",
		"o1",
		"o3",
		"o4",
		"chatgpt",
	}
)

func IsOpenAIResponseOnlyModel(modelName string) bool {
	for _, m := range OpenAIResponseOnlyModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

func IsImageGenerationModel(modelName string) bool {
	return matchesModelPattern(modelName, ImageGenerationModels)
}

func IsOpenAIVideoGenerationModel(modelName string) bool {
	return matchesModelPattern(modelName, OpenAIVideoGenerationModels)
}

func matchesModelPattern(modelName string, patterns []string) bool {
	modelName = strings.ToLower(modelName)
	for _, pattern := range patterns {
		pattern = strings.ToLower(pattern)
		if strings.HasPrefix(pattern, "exact:") {
			if modelName == strings.TrimPrefix(pattern, "exact:") {
				return true
			}
			continue
		}
		if strings.HasPrefix(pattern, "prefix:") {
			if strings.HasPrefix(modelName, strings.TrimPrefix(pattern, "prefix:")) {
				return true
			}
			continue
		}
		if strings.Contains(modelName, pattern) {
			return true
		}
	}
	return false
}

func IsOpenAITextModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range OpenAITextModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}
