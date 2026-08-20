package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestRemoveUnsupportedClaudeSamplingParamsDropsOverrideValues(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatOpenAI,
		ChannelMeta: &ChannelMeta{
			ApiType:           constant.APITypeAnthropic,
			UpstreamModelName: "claude-opus-5",
			ParamOverride: map[string]interface{}{
				"temperature": 0.7,
				"top_p":       0.9,
				"top_k":       40,
			},
		},
		RequestConversionChain: []types.RelayFormat{
			types.RelayFormatOpenAI,
			types.RelayFormatClaude,
		},
	}
	input := []byte(`{
		"model":"claude-opus-5",
		"messages":[{"role":"user","content":"hello"}],
		"max_tokens":64
	}`)

	overridden, err := ApplyParamOverrideWithRelayInfo(input, info)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(overridden, "temperature").Exists())
	require.True(t, gjson.GetBytes(overridden, "top_p").Exists())
	require.True(t, gjson.GetBytes(overridden, "top_k").Exists())

	out, err := RemoveUnsupportedClaudeSamplingParams(overridden, info)

	require.NoError(t, err)
	assert.False(t, gjson.GetBytes(out, "temperature").Exists())
	assert.False(t, gjson.GetBytes(out, "top_p").Exists())
	assert.False(t, gjson.GetBytes(out, "top_k").Exists())
	assert.Equal(t, "claude-opus-5", gjson.GetBytes(out, "model").String())
	assert.Equal(t, int64(64), gjson.GetBytes(out, "max_tokens").Int())
}

func TestRemoveUnsupportedClaudeSamplingParamsKeepsOpenAICompatiblePayload(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatOpenAI,
		ChannelMeta: &ChannelMeta{
			ApiType:           constant.APITypeOpenAI,
			UpstreamModelName: "claude-opus-5",
		},
	}
	input := []byte(`{
		"model":"claude-opus-5",
		"messages":[{"role":"user","content":"hello"}],
		"temperature":0.7,
		"top_p":0.9,
		"top_k":40
	}`)

	out, err := RemoveUnsupportedClaudeSamplingParams(input, info)

	require.NoError(t, err)
	assert.True(t, gjson.GetBytes(out, "temperature").Exists())
	assert.True(t, gjson.GetBytes(out, "top_p").Exists())
	assert.True(t, gjson.GetBytes(out, "top_k").Exists())
}

func TestRemoveUnsupportedClaudeSamplingParamsHandlesMissingChannelMeta(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatClaude,
	}
	input := []byte(`{
		"model":"models/claude-opus-4-8",
		"temperature":0.7,
		"top_p":0.9,
		"top_k":40
	}`)

	out, err := RemoveUnsupportedClaudeSamplingParams(input, info)

	require.NoError(t, err)
	assert.False(t, gjson.GetBytes(out, "temperature").Exists())
	assert.False(t, gjson.GetBytes(out, "top_p").Exists())
	assert.False(t, gjson.GetBytes(out, "top_k").Exists())
}
