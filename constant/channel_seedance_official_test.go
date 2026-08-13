package constant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedanceOfficialChannelRegistration(t *testing.T) {
	require.Greater(t, len(ChannelBaseURLs), ChannelTypeSeedanceOfficial)
	assert.Equal(t, "Seedance Official", GetChannelTypeName(ChannelTypeSeedanceOfficial))
	assert.Equal(t, "https://ark.cn-beijing.volces.com", ChannelBaseURLs[ChannelTypeSeedanceOfficial])
}
