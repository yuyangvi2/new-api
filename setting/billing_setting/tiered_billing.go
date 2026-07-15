package billing_setting

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/samber/lo"
)

const (
	BillingModeRatio      = "ratio"
	BillingModeTieredExpr = "tiered_expr"
	BillingModeField      = "billing_mode"
	BillingExprField      = "billing_expr"
)

// BillingSetting is managed by config.GlobalConfig.Register.
// DB keys: billing_setting.billing_mode, billing_setting.billing_expr
type BillingSetting struct {
	BillingMode map[string]string `json:"billing_mode"`
	BillingExpr map[string]string `json:"billing_expr"`
}

var defaultBillingMode = map[string]string{
	"glm-5.2":         BillingModeTieredExpr,
	"glm-5.1":         BillingModeTieredExpr,
	"glm-5-turbo":     BillingModeTieredExpr,
	"glm-5":           BillingModeTieredExpr,
	"glm-4.7":         BillingModeTieredExpr,
	"glm-4.5-air":     BillingModeTieredExpr,
	"glm-4.7-flashx":  BillingModeTieredExpr,
	"glm-5v-turbo":    BillingModeTieredExpr,
	"glm-4.6v":        BillingModeTieredExpr,
	"glm-4.6v-flashx": BillingModeTieredExpr,
	"glm-4.5v":        BillingModeTieredExpr,

	"qwen3.7-max":                   BillingModeTieredExpr,
	"qwen3.6-max-preview":           BillingModeTieredExpr,
	"qwen3-max":                     BillingModeTieredExpr,
	"qwen3.6-plus":                  BillingModeTieredExpr,
	"qwen3.5-plus":                  BillingModeTieredExpr,
	"qwen-plus":                     BillingModeTieredExpr,
	"qwen-plus-latest":              BillingModeTieredExpr,
	"qwen3.6-flash":                 BillingModeTieredExpr,
	"qwen3.6-flash-2026-04-16":      BillingModeTieredExpr,
	"qwen3.5-flash":                 BillingModeTieredExpr,
	"qwen-flash":                    BillingModeTieredExpr,
	"qwen-turbo":                    BillingModeTieredExpr,
	"qwq-plus":                      BillingModeTieredExpr,
	"qwen-long":                     BillingModeTieredExpr,
	"qwen3-vl-plus":                 BillingModeTieredExpr,
	"qwen3-vl-flash":                BillingModeTieredExpr,
	"qwen-vl-max":                   BillingModeTieredExpr,
	"qwen-vl-plus":                  BillingModeTieredExpr,
	"qwen3-coder":                   BillingModeTieredExpr,
	"qwen3-coder-plus":              BillingModeTieredExpr,
	"qwen3-coder-flash":             BillingModeTieredExpr,
	"qwen3-next":                    BillingModeTieredExpr,
	"qwen3-next-80b-a3b-thinking":   BillingModeTieredExpr,
	"qwen3-next-80b-a3b-instruct":   BillingModeTieredExpr,
	"qwen3-235b":                    BillingModeTieredExpr,
	"qwen3-235b-a22b":               BillingModeTieredExpr,
	"qwen3-235b-a22b-thinking-2507": BillingModeTieredExpr,
	"qwen3-235b-a22b-instruct-2507": BillingModeTieredExpr,
	"qwen3-32b":                     BillingModeTieredExpr,
	"qwen3-30b-a3b":                 BillingModeTieredExpr,
	"qwen3-30b-a3b-instruct-2507":   BillingModeTieredExpr,
	"qwen3.5-397b-a17b":             BillingModeTieredExpr,
	"qwen3.5-35b-a3b":               BillingModeTieredExpr,
	"qwen3.6-35b-a3b":               BillingModeTieredExpr,
	"qwen3.6-27b":                   BillingModeTieredExpr,
	"qwen3-vl-30b-a3b-instruct":     BillingModeTieredExpr,
	"qwen3-vl-235b-a22b-thinking":   BillingModeTieredExpr,
	"qwen3-vl-235b-a22b-instruct":   BillingModeTieredExpr,

	"kimi-k2.7-code":                BillingModeTieredExpr,
	"kimi-k2.7-code-highspeed":      BillingModeTieredExpr,
	"kimi/kimi-k2.7-code":           BillingModeTieredExpr,
	"kimi/kimi-k2.7-code-highspeed": BillingModeTieredExpr,
	"kimi-k2.6":                     BillingModeTieredExpr,
	"kimi/kimi-k2.6":                BillingModeTieredExpr,
	"kimi-k2.5":                     BillingModeTieredExpr,
	"kimi-k2-5-260127":              BillingModeTieredExpr,
	"kimi/kimi-k2.5":                BillingModeTieredExpr,
	"kimi-k2-thinking":              BillingModeTieredExpr,
	"Moonshot-Kimi-K2-Instruct":     BillingModeTieredExpr,

	"minimax-m3":             BillingModeTieredExpr,
	"MiniMax-M3":             BillingModeTieredExpr,
	"MiniMax/MiniMax-M3":     BillingModeTieredExpr,
	"minimax-m2.7":           BillingModeTieredExpr,
	"MiniMax-M2.7":           BillingModeTieredExpr,
	"MiniMax/MiniMax-M2.7":   BillingModeTieredExpr,
	"MiniMax-M2.7-highspeed": BillingModeTieredExpr,
	"minimax-m2.5":           BillingModeTieredExpr,
	"MiniMax-M2.5":           BillingModeTieredExpr,
	"MiniMax/MiniMax-M2.5":   BillingModeTieredExpr,
	"minimax-m2.5-highspeed": BillingModeTieredExpr,
	"MiniMax-M2.5-highspeed": BillingModeTieredExpr,
	"MiniMax-M2.1":           BillingModeTieredExpr,
	"MiniMax/MiniMax-M2.1":   BillingModeTieredExpr,
	"MiniMax-M2.1-highspeed": BillingModeTieredExpr,
	"MiniMax-M2":             BillingModeTieredExpr,
}

var defaultBillingExpr = map[string]string{
	// Zhipu official CNY prices converted to USD at 1 USD = 7.14 CNY.
	"glm-5.2":         `tier("base", p * 1.120448 + c * 3.921569 + cr * 0.280112)`,
	"glm-5.1":         `len < 32000 ? tier("short_context", p * 0.840336 + c * 3.361345 + cr * 0.182073) : tier("long_context", p * 1.120448 + c * 3.921569 + cr * 0.280112)`,
	"glm-5-turbo":     `len < 32000 ? tier("short_context", p * 0.700280 + c * 3.081232 + cr * 0.168067) : tier("long_context", p * 0.980392 + c * 3.641457 + cr * 0.252101)`,
	"glm-5":           `len < 32000 ? tier("short_context", p * 0.560224 + c * 2.521008 + cr * 0.140056) : tier("long_context", p * 0.840336 + c * 3.081232 + cr * 0.210084)`,
	"glm-4.7":         `len < 32000 && c < 200 ? tier("short_output", p * 0.280112 + c * 1.120448 + cr * 0.056022) : len < 32000 && c >= 200 ? tier("long_output", p * 0.420168 + c * 1.960784 + cr * 0.084034) : tier("mid_context", p * 0.560224 + c * 2.240896 + cr * 0.112045)`,
	"glm-4.5-air":     `len < 32000 && c < 200 ? tier("short_output", p * 0.112045 + c * 0.280112 + cr * 0.022409) : len < 32000 && c >= 200 ? tier("long_output", p * 0.112045 + c * 0.840336 + cr * 0.022409) : tier("mid_context", p * 0.168067 + c * 1.120448 + cr * 0.033613)`,
	"glm-4.7-flashx":  `tier("base", p * 0.070028 + c * 0.420168 + cr * 0.014006)`,
	"glm-5v-turbo":    `len < 32000 ? tier("short_context", p * 0.700280 + c * 3.081232 + cr * 0.168067) : tier("long_context", p * 0.980392 + c * 3.641457 + cr * 0.252101)`,
	"glm-4.6v":        `len < 32000 ? tier("short_context", p * 0.140056 + c * 0.420168 + cr * 0.028011) : tier("mid_context", p * 0.280112 + c * 0.840336 + cr * 0.056022)`,
	"glm-4.6v-flashx": `len < 32000 ? tier("short_context", p * 0.021008 + c * 0.210084 + cr * 0.004202) : tier("mid_context", p * 0.042017 + c * 0.420168 + cr * 0.004202)`,
	"glm-4.5v":        `len < 32000 ? tier("short_context", p * 0.280112 + c * 0.840336 + cr * 0.056022) : tier("mid_context", p * 0.560224 + c * 1.680672 + cr * 0.112045)`,

	// Qwen official CNY list prices converted to USD at 1 USD = 7.14 CNY.
	"qwen3.7-max":                   `tier("base", p * 1.680672 + c * 5.042017)`,
	"qwen3.6-max-preview":           `len <= 128000 ? tier("short_context", p * 1.260504 + c * 7.563025) : tier("mid_context", p * 2.100840 + c * 12.605042)`,
	"qwen3-max":                     `len <= 32000 ? tier("short_context", p * 0.350140 + c * 1.400560) : len <= 128000 ? tier("mid_context", p * 0.560224 + c * 2.240896) : tier("long_context", p * 0.980392 + c * 3.921569)`,
	"qwen3.6-plus":                  `len <= 256000 ? tier("short_context", p * 0.280112 + c * 1.680672) : tier("long_context", p * 1.120448 + c * 6.722689)`,
	"qwen3.5-plus":                  `len <= 128000 ? tier("short_context", p * 0.112045 + c * 0.672269) : len <= 256000 ? tier("mid_context", p * 0.280112 + c * 1.680672) : tier("long_context", p * 0.560224 + c * 3.361345)`,
	"qwen-plus":                     `len <= 128000 ? tier("short_context", p * 0.112045 + (rt > 0 ? (c + rt) * 1.120448 : c * 0.280112)) : len <= 256000 ? tier("mid_context", p * 0.336134 + (rt > 0 ? (c + rt) * 3.361345 : c * 2.801120)) : tier("long_context", p * 0.672269 + (rt > 0 ? (c + rt) * 8.963585 : c * 6.722689))`,
	"qwen-plus-latest":              `len <= 128000 ? tier("short_context", p * 0.112045 + (rt > 0 ? (c + rt) * 1.120448 : c * 0.280112)) : len <= 256000 ? tier("mid_context", p * 0.336134 + (rt > 0 ? (c + rt) * 3.361345 : c * 2.801120)) : tier("long_context", p * 0.672269 + (rt > 0 ? (c + rt) * 8.963585 : c * 6.722689))`,
	"qwen3.6-flash":                 `len <= 256000 ? tier("short_context", p * 0.168067 + c * 1.008403) : tier("long_context", p * 0.672269 + c * 4.033613)`,
	"qwen3.6-flash-2026-04-16":      `len <= 256000 ? tier("short_context", p * 0.168067 + c * 1.008403) : tier("long_context", p * 0.672269 + c * 4.033613)`,
	"qwen3.5-flash":                 `len <= 128000 ? tier("short_context", p * 0.028011 + c * 0.280112) : len <= 256000 ? tier("mid_context", p * 0.112045 + c * 1.120448) : tier("long_context", p * 0.168067 + c * 1.680672)`,
	"qwen-flash":                    `len <= 128000 ? tier("short_context", p * 0.021008 + c * 0.210084) : len <= 256000 ? tier("mid_context", p * 0.084034 + c * 0.840336) : tier("long_context", p * 0.168067 + c * 1.680672)`,
	"qwen-turbo":                    `tier("base", p * 0.042017 + (rt > 0 ? (c + rt) * 0.420168 : c * 0.084034))`,
	"qwq-plus":                      `tier("base", p * 0.224090 + c * 0.560224)`,
	"qwen-long":                     `tier("base", p * 0.070028 + c * 0.280112)`,
	"qwen3-vl-plus":                 `len <= 32000 ? tier("short_context", p * 0.140056 + c * 1.400560) : len <= 128000 ? tier("mid_context", p * 0.210084 + c * 2.100840) : tier("long_context", p * 0.420168 + c * 4.201681)`,
	"qwen3-vl-flash":                `len <= 32000 ? tier("short_context", p * 0.021008 + c * 0.210084) : len <= 128000 ? tier("mid_context", p * 0.042017 + c * 0.420168) : tier("long_context", p * 0.084034 + c * 0.840336)`,
	"qwen-vl-max":                   `tier("base", p * 0.224090 + c * 0.560224)`,
	"qwen-vl-plus":                  `tier("base", p * 0.112045 + c * 0.280112)`,
	"qwen3-coder":                   `len <= 32000 ? tier("short_context", p * 0.560224 + c * 2.240896) : len <= 128000 ? tier("mid_context", p * 0.840336 + c * 3.361345) : len <= 256000 ? tier("long_context", p * 1.400560 + c * 5.602241) : tier("max_context", p * 2.801120 + c * 28.011204)`,
	"qwen3-coder-plus":              `len <= 32000 ? tier("short_context", p * 0.560224 + c * 2.240896) : len <= 128000 ? tier("mid_context", p * 0.840336 + c * 3.361345) : len <= 256000 ? tier("long_context", p * 1.400560 + c * 5.602241) : tier("max_context", p * 2.801120 + c * 28.011204)`,
	"qwen3-coder-flash":             `len <= 32000 ? tier("short_context", p * 0.140056 + c * 0.560224) : len <= 128000 ? tier("mid_context", p * 0.210084 + c * 0.840336) : len <= 256000 ? tier("long_context", p * 0.350140 + c * 1.400560) : tier("max_context", p * 0.700280 + c * 3.501401)`,
	"qwen3-next":                    `tier("base", p * 0.140056 + (rt > 0 ? (c + rt) * 1.400560 : c * 0.560224))`,
	"qwen3-next-80b-a3b-thinking":   `tier("base", p * 0.140056 + c * 1.400560)`,
	"qwen3-next-80b-a3b-instruct":   `tier("base", p * 0.140056 + c * 0.560224)`,
	"qwen3-235b":                    `tier("base", p * 0.280112 + (rt > 0 ? (c + rt) * 2.801120 : c * 1.120448))`,
	"qwen3-235b-a22b":               `tier("base", p * 0.280112 + (rt > 0 ? (c + rt) * 2.801120 : c * 1.120448))`,
	"qwen3-235b-a22b-thinking-2507": `tier("base", p * 0.280112 + c * 2.801120)`,
	"qwen3-235b-a22b-instruct-2507": `tier("base", p * 0.280112 + c * 1.120448)`,
	"qwen3-32b":                     `tier("base", p * 0.280112 + (rt > 0 ? (c + rt) * 2.801120 : c * 1.120448))`,
	"qwen3-30b-a3b":                 `tier("base", p * 0.105042 + (rt > 0 ? (c + rt) * 1.050420 : c * 0.420168))`,
	"qwen3-30b-a3b-instruct-2507":   `tier("base", p * 0.105042 + c * 0.420168)`,
	"qwen3.5-397b-a17b":             `len <= 128000 ? tier("short_context", p * 0.168067 + c * 1.008403) : tier("mid_context", p * 0.420168 + c * 2.521008)`,
	"qwen3.5-35b-a3b":               `tier("base", p * 0.056022 + c * 0.448179)`,
	"qwen3.6-35b-a3b":               `tier("base", p * 0.252101 + c * 1.512605)`,
	"qwen3.6-27b":                   `tier("base", p * 0.420168 + c * 2.521008)`,
	"qwen3-vl-30b-a3b-instruct":     `tier("base", p * 0.105042 + c * 0.420168)`,
	"qwen3-vl-235b-a22b-thinking":   `tier("base", p * 0.280112 + c * 2.801120)`,
	"qwen3-vl-235b-a22b-instruct":   `tier("base", p * 0.280112 + c * 1.120448)`,

	// Kimi official CNY list prices converted to USD at 1 USD = 7.14 CNY.
	"kimi-k2.7-code":                `tier("base", p * 0.910364 + c * 3.781513 + cr * 0.182073)`,
	"kimi/kimi-k2.7-code":           `tier("base", p * 0.910364 + c * 3.781513 + cr * 0.182073)`,
	"kimi-k2.7-code-highspeed":      `tier("base", p * 1.820728 + c * 7.563025 + cr * 0.364146)`,
	"kimi/kimi-k2.7-code-highspeed": `tier("base", p * 1.820728 + c * 7.563025 + cr * 0.364146)`,
	"kimi-k2.6":                     `tier("base", p * 0.910364 + c * 3.781513 + cr * 0.154062)`,
	"kimi/kimi-k2.6":                `tier("base", p * 0.910364 + c * 3.781513 + cr * 0.154062)`,
	"kimi-k2.5":                     `tier("base", p * 0.560224 + c * 2.941176 + cr * 0.098039)`,
	"kimi-k2-5-260127":              `tier("base", p * 0.560224 + c * 2.941176 + cr * 0.098039)`,
	"kimi/kimi-k2.5":                `tier("base", p * 0.560224 + c * 2.941176 + cr * 0.098039)`,
	"kimi-k2-thinking":              `tier("base", p * 0.560224 + c * 2.240896)`,
	"Moonshot-Kimi-K2-Instruct":     `tier("base", p * 0.560224 + c * 2.240896)`,

	// MiniMax official CNY list prices converted to USD at 1 USD = 7.14 CNY.
	"minimax-m3":             `len <= 512000 ? tier("short_context", p * 0.588235 + c * 2.352941 + cr * 0.117647) : tier("long_context", p * 1.176471 + c * 4.705882 + cr * 0.235294)`,
	"MiniMax-M3":             `len <= 512000 ? tier("short_context", p * 0.588235 + c * 2.352941 + cr * 0.117647) : tier("long_context", p * 1.176471 + c * 4.705882 + cr * 0.235294)`,
	"MiniMax/MiniMax-M3":     `len <= 512000 ? tier("short_context", p * 0.588235 + c * 2.352941 + cr * 0.117647) : tier("long_context", p * 1.176471 + c * 4.705882 + cr * 0.235294)`,
	"minimax-m2.7":           `tier("base", p * 0.294118 + c * 1.176471 + cr * 0.058824 + cc * 0.367647)`,
	"MiniMax-M2.7":           `tier("base", p * 0.294118 + c * 1.176471 + cr * 0.058824 + cc * 0.367647)`,
	"MiniMax/MiniMax-M2.7":   `tier("base", p * 0.294118 + c * 1.176471 + cr * 0.058824 + cc * 0.367647)`,
	"MiniMax-M2.7-highspeed": `tier("base", p * 0.588235 + c * 2.352941 + cr * 0.058824 + cc * 0.367647)`,
	"minimax-m2.5":           `tier("base", p * 0.294118 + c * 1.176471 + cr * 0.029412 + cc * 0.367647)`,
	"MiniMax-M2.5":           `tier("base", p * 0.294118 + c * 1.176471 + cr * 0.029412 + cc * 0.367647)`,
	"MiniMax/MiniMax-M2.5":   `tier("base", p * 0.294118 + c * 1.176471 + cr * 0.029412 + cc * 0.367647)`,
	"minimax-m2.5-highspeed": `tier("base", p * 0.588235 + c * 2.352941 + cr * 0.029412 + cc * 0.367647)`,
	"MiniMax-M2.5-highspeed": `tier("base", p * 0.588235 + c * 2.352941 + cr * 0.029412 + cc * 0.367647)`,
	"MiniMax-M2.1":           `tier("base", p * 0.294118 + c * 1.176471 + cr * 0.029412 + cc * 0.367647)`,
	"MiniMax/MiniMax-M2.1":   `tier("base", p * 0.294118 + c * 1.176471 + cr * 0.029412 + cc * 0.367647)`,
	"MiniMax-M2.1-highspeed": `tier("base", p * 0.588235 + c * 2.352941 + cr * 0.029412 + cc * 0.367647)`,
	"MiniMax-M2":             `tier("base", p * 0.294118 + c * 1.176471 + cr * 0.029412 + cc * 0.367647)`,
}

var billingSetting = BillingSetting{
	BillingMode: lo.Assign(defaultBillingMode),
	BillingExpr: lo.Assign(defaultBillingExpr),
}

func init() {
	config.GlobalConfig.Register("billing_setting", &billingSetting)
}

// ---------------------------------------------------------------------------
// Read accessors (hot path, must be fast)
// ---------------------------------------------------------------------------

func GetBillingMode(model string) string {
	if mode, ok := billingSetting.BillingMode[model]; ok {
		return mode
	}
	return BillingModeRatio
}

func GetBillingExpr(model string) (string, bool) {
	expr, ok := billingSetting.BillingExpr[model]
	return expr, ok
}

func GetBillingModeCopy() map[string]string {
	return lo.Assign(billingSetting.BillingMode)
}

func GetBillingExprCopy() map[string]string {
	return lo.Assign(billingSetting.BillingExpr)
}

func BillingMode2JSONString() string {
	jsonBytes, err := common.Marshal(billingSetting.BillingMode)
	if err != nil {
		common.SysError("error marshalling billing mode: " + err.Error())
	}
	return string(jsonBytes)
}

func BillingExpr2JSONString() string {
	jsonBytes, err := common.Marshal(billingSetting.BillingExpr)
	if err != nil {
		common.SysError("error marshalling billing expr: " + err.Error())
	}
	return string(jsonBytes)
}

func DefaultBillingMode2JSONString() string {
	jsonBytes, err := common.Marshal(defaultBillingMode)
	if err != nil {
		common.SysError("error marshalling default billing mode: " + err.Error())
	}
	return string(jsonBytes)
}

func DefaultBillingExpr2JSONString() string {
	jsonBytes, err := common.Marshal(defaultBillingExpr)
	if err != nil {
		common.SysError("error marshalling default billing expr: " + err.Error())
	}
	return string(jsonBytes)
}

func GetPricingSyncData(base map[string]any) map[string]any {
	extra := make(map[string]any, 2)
	if modes := GetBillingModeCopy(); len(modes) > 0 {
		extra[BillingModeField] = modes
	}
	if exprs := GetBillingExprCopy(); len(exprs) > 0 {
		extra[BillingExprField] = exprs
	}
	return lo.Assign(base, extra)
}

// ---------------------------------------------------------------------------
// Smoke test (called externally for validation before save)
// ---------------------------------------------------------------------------

func SmokeTestExpr(exprStr string) error {
	return smokeTestExpr(exprStr)
}

func smokeTestExpr(exprStr string) error {
	vectors := []billingexpr.TokenParams{
		{P: 0, C: 0, Len: 0},
		{P: 1000, C: 1000, Len: 1000},
		{P: 100000, C: 100000, Len: 100000},
		{P: 1000000, C: 1000000, Len: 1000000},
	}
	requests := []billingexpr.RequestInput{
		{},
		{
			Headers: map[string]string{
				"anthropic-beta": "fast-mode-2026-02-01",
			},
			Body: []byte(`{"service_tier":"fast","stream_options":{"include_usage":true},"messages":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21]}`),
		},
	}

	for _, v := range vectors {
		for _, request := range requests {
			result, _, err := billingexpr.RunExprWithRequest(exprStr, v, request)
			if err != nil {
				return fmt.Errorf("vector {p=%g, c=%g}: run failed: %w", v.P, v.C, err)
			}
			if result < 0 {
				return fmt.Errorf("vector {p=%g, c=%g}: result %f < 0", v.P, v.C, result)
			}
		}
	}
	return nil
}
