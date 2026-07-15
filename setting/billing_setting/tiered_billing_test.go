package billing_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultBillingExprsAreValid(t *testing.T) {
	for model, expr := range defaultBillingExpr {
		t.Run(model, func(t *testing.T) {
			assert.Equal(t, BillingModeTieredExpr, defaultBillingMode[model])
			require.NoError(t, SmokeTestExpr(expr))
		})
	}
}

func TestDefaultBillingExprsMatchOfficialListPrices(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		params    billingexpr.TokenParams
		wantTier  string
		wantPrice float64
	}{
		{
			name:     "qwen3 max short context",
			model:    "qwen3-max",
			params:   billingexpr.TokenParams{P: 1, C: 1, Len: 32000},
			wantTier: "short_context",
			// 2.5 CNY input + 10 CNY output per 1M tokens, converted at 7.14.
			wantPrice: 0.350140 + 1.400560,
		},
		{
			name:     "qwen plus thinking output",
			model:    "qwen-plus",
			params:   billingexpr.TokenParams{P: 1, C: 2, RT: 3, Len: 128000},
			wantTier: "short_context",
			// 0.8 CNY input + 8 CNY thinking output per 1M tokens, converted at 7.14.
			wantPrice: 0.112045 + (2+3)*1.120448,
		},
		{
			name:     "kimi k2.6 cache read",
			model:    "kimi-k2.6",
			params:   billingexpr.TokenParams{P: 1, C: 1, CR: 1, Len: 1},
			wantTier: "base",
			// 6.5 CNY input + 27 CNY output + 1.1 CNY cache-read per 1M tokens.
			wantPrice: 0.910364 + 3.781513 + 0.154062,
		},
		{
			name:     "minimax m3 long context",
			model:    "minimax-m3",
			params:   billingexpr.TokenParams{P: 1, C: 1, CR: 1, Len: 600000},
			wantTier: "long_context",
			// Original list price, excluding the MiniMax M3 discount: 8.4/33.6/1.68 CNY.
			wantPrice: 1.176471 + 4.705882 + 0.235294,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expr, ok := defaultBillingExpr[test.model]
			require.True(t, ok)

			cost, trace, err := billingexpr.RunExpr(expr, test.params)
			require.NoError(t, err)
			assert.Equal(t, test.wantTier, trace.MatchedTier)
			assert.InDelta(t, test.wantPrice, cost, 0.000001)
		})
	}
}
