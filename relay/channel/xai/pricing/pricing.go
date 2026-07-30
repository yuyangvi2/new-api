package pricing

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

const (
	grokImagineStandardOutputPrice   = 0.02
	grokImagineStandardInputPrice    = 0.002
	grokImagineQuality1KPrice        = 0.05
	grokImagineQuality2KPrice        = 0.07
	grokImagineQualityInputPrice     = 0.01
	grokImagineVideoInputImagePrice  = 0.002
	grokImagineVideoInputSecondPrice = 0.01
	grokImagineVideo15InputPrice     = 0.01
)

func normalizeGrokImagineResolution(value string) string {
	resolution := strings.ToLower(strings.TrimSpace(value))
	switch resolution {
	case "2k", "2048", "2048x2048":
		return "2k"
	case "1080p", "1080":
		return "1080p"
	case "720p", "720":
		return "720p"
	case "480p", "480":
		return "480p"
	default:
		return "1k"
	}
}

func isGrokImagineImageModel(modelName string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	return modelName == "grok-imagine" ||
		modelName == "grok-imagine-image" ||
		modelName == "grok-imagine-image-pro" ||
		modelName == "grok-imagine-image-quality"
}

func isGrokImagineQualityImageModel(modelName string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	return modelName == "grok-imagine-image-pro" || modelName == "grok-imagine-image-quality"
}

func GrokImagineImagePricePerImage(modelName, resolution string, inputImageCount int) (float64, bool) {
	if !isGrokImagineImageModel(modelName) {
		return 0, false
	}
	inputPrice := grokImagineStandardInputPrice
	outputPrice := grokImagineStandardOutputPrice
	if isGrokImagineQualityImageModel(modelName) {
		inputPrice = grokImagineQualityInputPrice
		if normalizeGrokImagineResolution(resolution) == "2k" {
			outputPrice = grokImagineQuality2KPrice
		} else {
			outputPrice = grokImagineQuality1KPrice
		}
	}
	if inputImageCount < 0 {
		inputImageCount = 0
	}
	return outputPrice + float64(inputImageCount)*inputPrice, true
}

func GrokImagineImageRequestPrice(modelName, resolution string, outputCount, inputImageCount int) (float64, bool) {
	perImage, ok := GrokImagineImagePricePerImage(modelName, resolution, inputImageCount)
	if !ok {
		return 0, false
	}
	if outputCount < 1 {
		outputCount = 1
	}
	inputPrice := 0.0
	if isGrokImagineQualityImageModel(modelName) {
		inputPrice = grokImagineQualityInputPrice
	} else {
		inputPrice = grokImagineStandardInputPrice
	}
	outputOnly := perImage - float64(inputImageCount)*inputPrice
	return outputOnly*float64(outputCount) + float64(inputImageCount)*inputPrice, true
}

func GrokImagineVideoPrice(modelName, resolution string, duration int, inputImageCount int) (float64, bool) {
	return GrokImagineVideoPriceWithInputVideo(modelName, resolution, duration, inputImageCount, 0)
}

func GrokImagineVideoPriceWithInputVideo(modelName, resolution string, duration int, inputImageCount int, inputVideoSeconds float64) (float64, bool) {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	if duration < 1 {
		duration = 1
	}
	switch modelName {
	case "grok-imagine-video":
		outputPrice, ok := GrokImagineVideoOutputSecondPrice(resolution)
		if !ok {
			return 0, false
		}
		if inputImageCount < 0 {
			inputImageCount = 0
		}
		if inputVideoSeconds < 0 {
			inputVideoSeconds = 0
		}
		return outputPrice*float64(duration) +
			grokImagineVideoInputImagePrice*float64(inputImageCount) +
			grokImagineVideoInputSecondPrice*inputVideoSeconds, true
	case "grok-imagine-video-1.5":
		outputPrice, ok := GrokImagineVideo15OutputSecondPrice(resolution)
		if !ok {
			return 0, false
		}
		if inputImageCount < 1 {
			inputImageCount = 1
		}
		return outputPrice*float64(duration) + grokImagineVideo15InputPrice*float64(inputImageCount), true
	default:
		return 0, false
	}
}

func GrokImagineVideoOutputSecondPrice(resolution string) (float64, bool) {
	switch normalizeGrokImagineResolution(resolution) {
	case "480p":
		return 0.05, true
	case "720p", "1k":
		return 0.07, true
	default:
		return 0, false
	}
}

func GrokImagineVideo15OutputSecondPrice(resolution string) (float64, bool) {
	switch normalizeGrokImagineResolution(resolution) {
	case "480p":
		return 0.08, true
	case "720p", "1k":
		return 0.14, true
	case "1080p":
		return 0.25, true
	default:
		return 0, false
	}
}

func GrokImagineTaskPrice(req relaycommon.TaskSubmitReq, modelName string) (float64, bool) {
	resolution := stringFromTaskMetadata(req, "resolution")
	if resolution == "" {
		resolution = stringFromTaskMetadata(req, "size")
	}
	duration := req.Duration
	if duration <= 0 {
		duration = intFromString(req.Seconds)
	}
	if duration <= 0 {
		duration = intFromAny(req.Metadata["duration"])
	}
	inputImages := 0
	if strings.TrimSpace(req.Image) != "" {
		inputImages = 1
	} else {
		inputImages = len(req.Images)
	}
	inputVideoSeconds := firstPositiveFloat(
		req.Metadata["input_video_duration"],
		req.Metadata["inputVideoDuration"],
		req.Metadata["input_duration"],
	)
	return GrokImagineVideoPriceWithInputVideo(modelName, resolution, duration, inputImages, inputVideoSeconds)
}

func firstPositiveFloat(values ...any) float64 {
	for _, value := range values {
		switch v := value.(type) {
		case int:
			if v > 0 {
				return float64(v)
			}
		case int64:
			if v > 0 {
				return float64(v)
			}
		case float64:
			if v > 0 {
				return v
			}
		}
	}
	return 0
}

func stringFromTaskMetadata(req relaycommon.TaskSubmitReq, key string) string {
	value, ok := req.Metadata[key]
	if !ok {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func intFromString(value string) int {
	var n int
	for _, r := range strings.TrimSpace(value) {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		if v > 0 && v <= int64(^uint(0)>>1) {
			return int(v)
		}
	case float64:
		if v > 0 && v == float64(int(v)) {
			return int(v)
		}
	}
	return 0
}

func CountImageRequestInputs(request dto.ImageRequest) int {
	count := 0
	if len(request.Image) > 0 && string(request.Image) != "null" {
		count++
	}
	if len(request.Images) > 0 && string(request.Images) != "null" {
		var list []any
		if err := common.Unmarshal(request.Images, &list); err == nil {
			count += len(list)
		} else {
			count++
		}
	}
	return count
}
