package constant

import (
	"fmt"
	"strings"
)

const ChannelTypeVODAIGC = 9007

var vodAIGCModelNames = map[string]string{
	"og":      "OG",
	"gg":      "GG",
	"hunyuan": "Hunyuan",
	"vidu":    "Vidu",
	"kling":   "Kling",
	"mingmou": "Mingmou",
}

func init() {
	registerCustomChannel(ChannelTypeVODAIGC, "Tencent VOD Image", "https://vod.tencentcloudapi.com")
}

// ParseVODAIGCUpstreamModel validates a resolved channel model mapping and
// returns the canonical Tencent model name and version.
func ParseVODAIGCUpstreamModel(value string, fallbackModelName string) (string, string, error) {
	target := strings.TrimSpace(value)
	if target == "" {
		return "", "", fmt.Errorf("upstream model mapping is empty")
	}

	name := strings.TrimSpace(fallbackModelName)
	version := target
	if before, after, found := strings.Cut(target, ":"); found {
		name = strings.TrimSpace(before)
		version = strings.TrimSpace(after)
	}
	canonicalName, ok := vodAIGCModelNames[strings.ToLower(name)]
	if !ok {
		return "", "", fmt.Errorf("unsupported Tencent VOD image model name: %s", name)
	}
	if version == "" || strings.Contains(version, ":") {
		return "", "", fmt.Errorf("invalid Tencent VOD image model version: %s", version)
	}
	return canonicalName, version, nil
}

// IsImageTaskChannelType reports whether a channel implements the asynchronous
// image-task protocol exposed at /v1/images/tasks and /pg/images/tasks.
func IsImageTaskChannelType(channelType int) bool {
	return channelType == ChannelTypeAIArt || channelType == ChannelTypeVODAIGC
}

func IsImageTaskRequestPath(requestPath string) bool {
	return strings.HasPrefix(requestPath, "/v1/images/tasks") ||
		strings.HasPrefix(requestPath, "/pg/images/tasks")
}

func IsSynchronousImageRequestPath(requestPath string) bool {
	return strings.HasPrefix(requestPath, "/v1/images/generations") ||
		strings.HasPrefix(requestPath, "/pg/images/generations") ||
		strings.HasPrefix(requestPath, "/v1/images/edits") ||
		strings.HasPrefix(requestPath, "/v1/edits")
}

// ChannelTypeSupportsRelayPath prevents asynchronous image channels from being
// selected for synchronous image requests, and vice versa.
func ChannelTypeSupportsRelayPath(channelType int, requestPath string) bool {
	if channelType == ChannelTypeVODAIGC {
		return IsImageTaskRequestPath(requestPath)
	}

	if IsImageTaskRequestPath(requestPath) {
		return channelType == ChannelTypeAIArt
	}

	if IsSynchronousImageRequestPath(requestPath) && IsImageTaskChannelType(channelType) {
		return false
	}

	return true
}
