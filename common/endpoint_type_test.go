package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedanceOfficialEndpointTypes(t *testing.T) {
	endpointTypes := GetEndpointTypesByChannelType(
		constant.ChannelTypeSeedanceOfficial,
		"doubao-seedance-2-0-260128",
	)

	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeSeedance}, endpointTypes)
}

func TestSeedanceOfficialAPIType(t *testing.T) {
	apiType, ok := ChannelType2APIType(constant.ChannelTypeSeedanceOfficial)

	require.True(t, ok)
	assert.Equal(t, constant.APITypeVolcEngine, apiType)
}

func TestSeedanceMEndpointAndAPIType(t *testing.T) {
	endpointTypes := GetEndpointTypesByChannelType(
		constant.ChannelTypeSeedanceM,
		"doubao-seedance-2.0",
	)
	apiType, ok := ChannelType2APIType(constant.ChannelTypeSeedanceM)

	require.True(t, ok)
	assert.Equal(t, constant.APITypeVolcEngine, apiType)
	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIVideo}, endpointTypes)
}

func TestSeedanceDefaultEndpointInfo(t *testing.T) {
	info, ok := GetDefaultEndpointInfo(constant.EndpointTypeSeedance)

	require.True(t, ok)
	assert.Equal(t, "/volcengine/api/v3/contents/generations/tasks", info.Path)
	assert.Equal(t, "POST", info.Method)
}
