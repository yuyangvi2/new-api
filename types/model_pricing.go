package types

type ModelPricingPeriod struct {
	Name          string  `json:"name"`
	Start         string  `json:"start"`
	End           string  `json:"end"`
	InputPrice    float64 `json:"input_price"`
	CacheHitPrice float64 `json:"cache_hit_price"`
	OutputPrice   float64 `json:"output_price"`
}

type ModelPricingSchedule struct {
	Timezone string               `json:"timezone"`
	Periods  []ModelPricingPeriod `json:"periods"`
}

type ModelPricingSnapshot struct {
	PeriodName      string  `json:"period_name"`
	Timezone        string  `json:"timezone"`
	InputPrice      float64 `json:"input_price"`
	CacheHitPrice   float64 `json:"cache_hit_price"`
	OutputPrice     float64 `json:"output_price"`
	ModelRatio      float64 `json:"model_ratio"`
	CacheRatio      float64 `json:"cache_ratio"`
	CompletionRatio float64 `json:"completion_ratio"`
}
