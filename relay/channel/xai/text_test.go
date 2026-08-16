package xai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXAITokenDeltaNeverNegative(t *testing.T) {
	tests := []struct {
		name   string
		total  int
		prompt int
		want   int
	}{
		{name: "normal delta", total: 12, prompt: 5, want: 7},
		{name: "equal tokens", total: 5, prompt: 5, want: 0},
		{name: "total below prompt", total: 3, prompt: 5, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, xAITokenDelta(tt.total, tt.prompt))
		})
	}
}
