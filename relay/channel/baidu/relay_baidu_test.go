package baidu

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestOpenAI2BaiduPreservesExplicitZeroOptionals(t *testing.T) {
	got := requestOpenAI2Baidu(dto.GeneralOpenAIRequest{
		TopP:             lo.ToPtr(0.0),
		FrequencyPenalty: lo.ToPtr(0.0),
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	})

	require.NotNil(t, got.TopP)
	require.NotNil(t, got.PenaltyScore)
	assert.Equal(t, 0.0, *got.TopP)
	assert.Equal(t, 0.0, *got.PenaltyScore)

	body, err := common.Marshal(got)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"top_p":0`)
	assert.Contains(t, string(body), `"penalty_score":0`)
}
