package ratio_setting

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

const deepSeekPricingTimezone = "Asia/Shanghai"

var defaultModelPricing = map[string]types.ModelPricingSchedule{
	"deepseek-chat": {
		Timezone: deepSeekPricingTimezone,
		Periods: []types.ModelPricingPeriod{
			{Name: "off_peak", Start: "00:30", End: "08:30", InputPrice: 0.135, CacheHitPrice: 0.035, OutputPrice: 0.55},
			{Name: "peak", Start: "08:30", End: "00:30", InputPrice: 0.27, CacheHitPrice: 0.07, OutputPrice: 1.10},
		},
	},
	"deepseek-coder": {
		Timezone: deepSeekPricingTimezone,
		Periods: []types.ModelPricingPeriod{
			{Name: "off_peak", Start: "00:30", End: "08:30", InputPrice: 0.135, CacheHitPrice: 0.035, OutputPrice: 0.55},
			{Name: "peak", Start: "08:30", End: "00:30", InputPrice: 0.27, CacheHitPrice: 0.07, OutputPrice: 1.10},
		},
	},
	"deepseek-reasoner": {
		Timezone: deepSeekPricingTimezone,
		Periods: []types.ModelPricingPeriod{
			{Name: "off_peak", Start: "00:30", End: "08:30", InputPrice: 0.275, CacheHitPrice: 0.07, OutputPrice: 1.095},
			{Name: "peak", Start: "08:30", End: "00:30", InputPrice: 0.55, CacheHitPrice: 0.14, OutputPrice: 2.19},
		},
	},
}

var modelPricingMap = types.NewRWMap[string, types.ModelPricingSchedule]()

func InitModelPricing() {
	modelPricingMap.AddAll(defaultModelPricing)
}

func GetModelPricingMap() map[string]types.ModelPricingSchedule {
	return modelPricingMap.ReadAll()
}

func ModelPricing2JSONString() string {
	return modelPricingMap.MarshalJSONString()
}

func UpdateModelPricingByJSONString(jsonStr string) error {
	var schedules map[string]types.ModelPricingSchedule
	if err := common.Unmarshal([]byte(jsonStr), &schedules); err != nil {
		return err
	}
	for modelName, schedule := range schedules {
		if err := validateModelPricingSchedule(modelName, schedule); err != nil {
			return err
		}
	}
	return types.LoadFromJsonStringWithCallback(modelPricingMap, jsonStr, InvalidateExposedDataCache)
}

func ResolveModelPricing(modelName string, now time.Time) (*types.ModelPricingSnapshot, bool, error) {
	schedule, ok := modelPricingMap.Get(FormatMatchingModelName(modelName))
	if !ok {
		return nil, false, nil
	}
	if schedule.Timezone == "" {
		schedule.Timezone = time.UTC.String()
	}
	location, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return nil, false, fmt.Errorf("invalid pricing timezone %q: %w", schedule.Timezone, err)
	}
	localNow := now.In(location)
	for _, period := range schedule.Periods {
		if !modelPricingPeriodContains(period, localNow) {
			continue
		}
		if period.InputPrice <= 0 || period.CacheHitPrice < 0 || period.OutputPrice <= 0 {
			return nil, false, fmt.Errorf("invalid pricing values for %s/%s", modelName, period.Name)
		}
		return &types.ModelPricingSnapshot{
			PeriodName:      period.Name,
			Timezone:        schedule.Timezone,
			InputPrice:      period.InputPrice,
			CacheHitPrice:   period.CacheHitPrice,
			OutputPrice:     period.OutputPrice,
			ModelRatio:      period.InputPrice / 2,
			CacheRatio:      period.CacheHitPrice / period.InputPrice,
			CompletionRatio: period.OutputPrice / period.InputPrice,
		}, true, nil
	}
	return nil, false, nil
}

func modelPricingPeriodContains(period types.ModelPricingPeriod, now time.Time) bool {
	start, ok := parsePricingMinutes(period.Start)
	if !ok {
		return false
	}
	end, ok := parsePricingMinutes(period.End)
	if !ok {
		return false
	}
	current := now.Hour()*60 + now.Minute()
	if start < end {
		return current >= start && current < end
	}
	if start > end {
		return current >= start || current < end
	}
	return true
}

func parsePricingMinutes(value string) (int, bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, false
	}
	var hour, minute int
	if _, err := fmt.Sscanf(value, "%d:%d", &hour, &minute); err != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func validateModelPricingSchedule(modelName string, schedule types.ModelPricingSchedule) error {
	if schedule.Timezone == "" {
		return fmt.Errorf("pricing timezone is required for %s", modelName)
	}
	if _, err := time.LoadLocation(schedule.Timezone); err != nil {
		return fmt.Errorf("invalid pricing timezone %q: %w", schedule.Timezone, err)
	}
	if len(schedule.Periods) == 0 {
		return fmt.Errorf("pricing periods are required for %s", modelName)
	}
	for _, period := range schedule.Periods {
		if _, ok := parsePricingMinutes(period.Start); !ok {
			return fmt.Errorf("invalid pricing start time %q for %s/%s", period.Start, modelName, period.Name)
		}
		if _, ok := parsePricingMinutes(period.End); !ok {
			return fmt.Errorf("invalid pricing end time %q for %s/%s", period.End, modelName, period.Name)
		}
		if period.InputPrice <= 0 || period.CacheHitPrice < 0 || period.OutputPrice <= 0 {
			return fmt.Errorf("invalid pricing values for %s/%s", modelName, period.Name)
		}
	}
	return nil
}

func init() {
	InitModelPricing()
}
