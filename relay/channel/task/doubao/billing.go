package doubao

import (
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const seedanceDefaultUSDExchangeRate = 7.14

func SeedancePricePerMillionCNY(modelName, resolution string, hasVideoInput bool) (float64, bool) {
	return seedancePricePerMillionCNY(videoPriceTable, modelName, resolution, hasVideoInput)
}

func seedancePricePerMillionCNY(priceTable map[string]map[videoPriceKey]float64, modelName, resolution string, hasVideoInput bool) (float64, bool) {
	prices, ok := priceTable[NormalizeSeedanceModelName(modelName)]
	if !ok {
		return 0, false
	}
	resolution = strings.ToLower(strings.TrimSpace(resolution))
	price, ok := prices[videoPriceKey{
		is1080p:  resolution == "1080p",
		is4k:     resolution == "4k",
		hasVideo: hasVideoInput,
	}]
	if ok {
		return price, true
	}
	price, ok = prices[videoPriceKey{hasVideo: hasVideoInput}]
	if ok {
		return price, true
	}
	price, ok = prices[videoPriceKey{}]
	return price, ok
}

func EstimateSeedanceVideoTokens(inputDuration float64, outputDuration int, ratio, resolution string) int {
	longEdge, shortEdge := SeedanceOutputSize(ratio, resolution)
	seconds := clampSeedanceDurationFloat(inputDuration) + float64(clampSeedanceDurationInt(outputDuration))
	return int(math.Round(seconds * float64(longEdge) * float64(shortEdge) * 24 / 1024))
}

func EstimateSeedanceQuota(modelName, resolution, ratio string, outputDuration int, inputDuration float64, hasVideoInput bool, groupRatio float64) (int, int, float64, bool) {
	return estimateSeedanceQuota(videoPriceTable, modelName, resolution, ratio, outputDuration, inputDuration, hasVideoInput, groupRatio)
}

func EstimateSeedanceQuotaFromUsageTokens(modelName, resolution string, totalTokens int, hasVideoInput bool, groupRatio, pricePerMillionCNY, usdExchangeRate float64) (int, *common.QuotaClamp, bool) {
	if totalTokens <= 0 {
		return 0, nil, false
	}
	if pricePerMillionCNY <= 0 {
		price, ok := SeedancePricePerMillionCNY(modelName, resolution, hasVideoInput)
		if !ok {
			return 0, nil, false
		}
		pricePerMillionCNY = price
	}
	if groupRatio < 0 {
		groupRatio = 0
	}
	if usdExchangeRate <= 0 {
		usdExchangeRate = seedanceUSDExchangeRate()
	}
	amountCNY := float64(totalTokens) * pricePerMillionCNY / 1_000_000
	quota, clamp := common.QuotaFromFloatChecked(amountCNY * groupRatio * common.QuotaPerUnit / usdExchangeRate)
	return quota, clamp, true
}

func estimateSeedanceQuota(priceTable map[string]map[videoPriceKey]float64, modelName, resolution, ratio string, outputDuration int, inputDuration float64, hasVideoInput bool, groupRatio float64) (int, int, float64, bool) {
	pricePerMillionCNY, ok := seedancePricePerMillionCNY(priceTable, modelName, resolution, hasVideoInput)
	if !ok {
		return 0, 0, 0, false
	}
	tokens := EstimateSeedanceVideoTokens(inputDuration, outputDuration, ratio, resolution)
	amountCNY := float64(tokens) * pricePerMillionCNY / 1_000_000
	if groupRatio < 0 {
		groupRatio = 0
	}
	quota := common.QuotaRound(amountCNY * groupRatio * common.QuotaPerUnit / seedanceUSDExchangeRate())
	return quota, tokens, pricePerMillionCNY, true
}

func EstimateSeedanceQuotaForRequest(modelName string, req relaycommon.TaskSubmitReq, hasVideoInput bool, groupRatio float64) (int, int, float64, bool) {
	return estimateSeedanceQuotaForRequest(videoPriceTable, modelName, req, hasVideoInput, groupRatio)
}

func estimateSeedanceQuotaForRequest(priceTable map[string]map[videoPriceKey]float64, modelName string, req relaycommon.TaskSubmitReq, hasVideoInput bool, groupRatio float64) (int, int, float64, bool) {
	resolution := stringFromAny(req.Metadata["resolution"])
	if resolution == "" && strings.HasSuffix(strings.ToLower(strings.TrimSpace(req.Size)), "p") {
		resolution = req.Size
	}
	if resolution == "" {
		resolution = "720p"
	}

	ratio := stringFromAny(req.Metadata["ratio"])
	if ratio == "" {
		ratio = req.Size
	}
	if ratio == "" {
		ratio = "16:9"
	}

	outputDuration := req.Duration
	if outputDuration <= 0 {
		if parsed, err := strconv.Atoi(strings.TrimSpace(req.Seconds)); err == nil {
			outputDuration = parsed
		}
	}
	if outputDuration <= 0 {
		outputDuration = clampSeedanceDurationFloatToInt(floatFromAny(req.Metadata["duration"]))
	}
	if outputDuration <= 0 {
		outputDuration = 5
	}
	outputDuration = clampSeedanceDurationInt(outputDuration)

	inputDuration := clampSeedanceDurationFloat(firstPositiveFloat(
		req.Metadata["input_video_duration"],
		req.Metadata["inputVideoDuration"],
		req.Metadata["input_duration"],
	))
	if hasVideoInput && inputDuration <= 0 {
		inputDuration = float64(outputDuration)
	}

	return estimateSeedanceQuota(priceTable, modelName, resolution, ratio, outputDuration, inputDuration, hasVideoInput, groupRatio)
}

func SeedanceOutputSize(ratioValue, resolutionValue string) (int, int) {
	shortEdge := 720
	switch strings.ToLower(strings.TrimSpace(resolutionValue)) {
	case "480p":
		shortEdge = 480
	case "1080p":
		shortEdge = 1080
	case "4k":
		shortEdge = 2160
	}

	ratio := parseSeedanceRatio(ratioValue)
	longEdge := int(math.Round(float64(shortEdge) * math.Max(ratio, 1/ratio)))
	return longEdge, shortEdge
}

func NormalizeSeedanceModelName(modelName string) string {
	normalized := strings.ToLower(strings.TrimSpace(modelName))
	switch normalized {
	case "seedance_2.0", "doubao-seedance-2.0":
		return "seedance2.0_direct"
	case "seedance_2.0_fast", "doubao-seedance-2-0-fast":
		return "seedance2.0_fast_direct"
	case "seedance_2.0_mini", "seedance2.0_mini", "doubao-seedance-2-0-mini":
		return "Seedance_2.0_mini"
	case "seedance_2.0_mini_lite", "seedance2.0_mini_lite", "seedance2.0_fast_mini":
		return "Seedance_2.0_mini_lite"
	default:
		return strings.TrimSpace(modelName)
	}
}

func IsSeedanceModel(modelName string) bool {
	_, ok := videoPriceTable[NormalizeSeedanceModelName(modelName)]
	return ok
}

func parseSeedanceRatio(value string) float64 {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if strings.Contains(normalized, "x") {
		parts := strings.Split(normalized, "x")
		if len(parts) == 2 {
			width, widthErr := strconv.ParseFloat(parts[0], 64)
			height, heightErr := strconv.ParseFloat(parts[1], 64)
			if widthErr == nil && heightErr == nil && width > 0 && height > 0 {
				return normalizeSeedanceRatio(width / height)
			}
		}
	}

	parts := strings.Split(normalized, ":")
	if len(parts) != 2 {
		return 16.0 / 9.0
	}
	width, err := strconv.ParseFloat(parts[0], 64)
	if err != nil || width <= 0 {
		return 16.0 / 9.0
	}
	height, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || height <= 0 {
		return 16.0 / 9.0
	}
	return normalizeSeedanceRatio(width / height)
}

func normalizeSeedanceRatio(ratio float64) float64 {
	if math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio <= 0 {
		return 16.0 / 9.0
	}
	if ratio < 0.25 || ratio > 4 {
		return 16.0 / 9.0
	}
	return ratio
}

func clampSeedanceDurationFloatToInt(value float64) int {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0
	}
	if value > relaycommon.MaxTaskDurationSeconds {
		return relaycommon.MaxTaskDurationSeconds
	}
	return int(math.Round(value))
}

func clampSeedanceDurationFloat(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0
	}
	if value > relaycommon.MaxTaskDurationSeconds {
		return float64(relaycommon.MaxTaskDurationSeconds)
	}
	return value
}

func clampSeedanceDurationInt(value int) int {
	if value <= 0 {
		return 0
	}
	if value > relaycommon.MaxTaskDurationSeconds {
		return relaycommon.MaxTaskDurationSeconds
	}
	return value
}

func seedanceUSDExchangeRate() float64 {
	if operation_setting.USDExchangeRate > 0 {
		return operation_setting.USDExchangeRate
	}
	return seedanceDefaultUSDExchangeRate
}

func SeedanceUSDExchangeRate() float64 {
	return seedanceUSDExchangeRate()
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	default:
		return ""
	}
}

func firstPositiveFloat(values ...any) float64 {
	for _, value := range values {
		parsed := floatFromAny(value)
		if parsed > 0 {
			return parsed
		}
	}
	return 0
}

func floatFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}
