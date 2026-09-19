package constant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVODAIGCChannelRegistrationAndPathCapabilities(t *testing.T) {
	require.Greater(t, len(ChannelBaseURLs), ChannelTypeVODAIGC)
	assert.Equal(t, "Tencent VOD Image", GetChannelTypeName(ChannelTypeVODAIGC))
	assert.Equal(t, "https://vod.tencentcloudapi.com", ChannelBaseURLs[ChannelTypeVODAIGC])

	assert.True(t, ChannelTypeSupportsRelayPath(ChannelTypeVODAIGC, "/v1/images/tasks"))
	assert.True(t, ChannelTypeSupportsRelayPath(ChannelTypeAIArt, "/pg/images/tasks"))
	assert.False(t, ChannelTypeSupportsRelayPath(ChannelTypeGemini, "/v1/images/tasks"))
	assert.False(t, ChannelTypeSupportsRelayPath(ChannelTypeVODAIGC, "/v1/images/generations"))
	assert.False(t, ChannelTypeSupportsRelayPath(ChannelTypeVODAIGC, "/v1/video/generations"))
	assert.False(t, ChannelTypeSupportsRelayPath(ChannelTypeVODAIGC, "/v1/chat/completions"))
	assert.True(t, ChannelTypeSupportsRelayPath(ChannelTypeGemini, "/v1/images/generations"))
}

func TestParseVODAIGCUpstreamModel(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		fallback    string
		wantName    string
		wantVersion string
		wantError   bool
	}{
		{name: "explicit family", value: "Vidu:q2", fallback: "GG", wantName: "Vidu", wantVersion: "q2"},
		{name: "fallback family", value: "3.1", fallback: "gg", wantName: "GG", wantVersion: "3.1"},
		{name: "unknown family", value: "Unknown:1", fallback: "GG", wantError: true},
		{name: "empty version", value: "GG:", fallback: "GG", wantError: true},
		{name: "extra delimiter", value: "GG:3.1:extra", fallback: "GG", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, version, err := ParseVODAIGCUpstreamModel(tt.value, tt.fallback)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantVersion, version)
		})
	}
}
