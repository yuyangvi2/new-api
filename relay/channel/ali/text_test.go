package ali

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestOpenAI2AliDoesNotDefaultMissingTopP(t *testing.T) {
	got := requestOpenAI2Ali(dto.GeneralOpenAIRequest{})

	assert.Nil(t, got.TopP)
}

func TestRequestOpenAI2AliClampsExplicitTopP(t *testing.T) {
	tests := []struct {
		name string
		topP float64
		want float64
	}{
		{name: "zero", topP: 0, want: 0.001},
		{name: "one", topP: 1, want: 0.999},
		{name: "valid", topP: 0.7, want: 0.7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requestOpenAI2Ali(dto.GeneralOpenAIRequest{TopP: lo.ToPtr(tt.topP)})

			require.NotNil(t, got.TopP)
			assert.Equal(t, tt.want, *got.TopP)
		})
	}
}
