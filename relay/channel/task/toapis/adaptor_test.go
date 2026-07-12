package toapis

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSubmitPayloadTextToVideo(t *testing.T) {
	req := &relaycommon.TaskSubmitReq{
		Model:    "seedance-2-fast",
		Prompt:   "make a rainy street video",
		Duration: 5,
		Size:     "1280x720",
		Metadata: map[string]interface{}{
			"resolution":     "720p",
			"generate_audio": true,
		},
	}

	payload, videoSR, err := buildSubmitPayload(req, "seedance-2-fast", "seedance-2-fast", false)
	require.NoError(t, err)

	require.Nil(t, videoSR)
	assert.Equal(t, "seedance-2-fast", payload["model"])
	assert.Equal(t, "make a rainy street video", payload["prompt"])
	assert.Equal(t, 5, payload["duration"])
	assert.Equal(t, "16:9", payload["aspect_ratio"])
	assert.Equal(t, "720p", payload["resolution"])
	assert.Equal(t, true, payload["generate_audio"])
}

func TestBuildSubmitPayloadReferenceMedia(t *testing.T) {
	req := &relaycommon.TaskSubmitReq{
		Model:  "seedance-2-mini",
		Prompt: "follow the references",
		Image:  "data:image/png;base64,abcd",
		Images: []string{"https://example.com/ref.png"},
		Metadata: map[string]interface{}{
			"referenceVideoUrls": []interface{}{"https://example.com/ref.mp4"},
			"audio_files":        []interface{}{"https://example.com/ref.mp3"},
			"generate_audio":     true,
		},
	}

	payload, videoSR, err := buildSubmitPayload(req, "seedance-2-mini", "seedance-2-mini", false)
	require.NoError(t, err)

	require.Nil(t, videoSR)
	assert.Equal(t, "seedance-2-mini", payload["model"])
	assert.NotContains(t, payload, "generate_audio")
	assert.Equal(t, []roleMedia{
		{URL: "data:image/png;base64,abcd", Role: "reference_image"},
		{URL: "https://example.com/ref.png", Role: "reference_image"},
	}, payload["image_with_roles"])
	assert.Equal(t, []roleMedia{
		{URL: "https://example.com/ref.mp4", Role: "reference_video"},
	}, payload["video_with_roles"])
	assert.Equal(t, []roleMedia{
		{URL: "https://example.com/ref.mp3", Role: "reference_audio"},
	}, payload["audio_with_roles"])
	assert.True(t, hasVideoInput(*req))
}

func TestBuildSubmitPayloadRejectsAudioOnly(t *testing.T) {
	req := &relaycommon.TaskSubmitReq{
		Model:  "seedance-2",
		Prompt: "use the audio",
		Metadata: map[string]interface{}{
			"referenceAudioUrls": []interface{}{"https://example.com/ref.mp3"},
		},
	}

	_, _, err := buildSubmitPayload(req, "seedance-2", "seedance-2", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audio_with_roles cannot be used alone")
}

func TestBuildSubmitPayloadVideoSuperResolutionRewrite(t *testing.T) {
	req := &relaycommon.TaskSubmitReq{
		Model:    "seedance-2",
		Prompt:   "cinematic city",
		Duration: 5,
		Size:     "3840x2160",
		Metadata: map[string]interface{}{
			"resolution": "4k",
		},
	}

	payload, videoSR, err := buildSubmitPayload(req, "seedance-2", "seedance-2", true)
	require.NoError(t, err)
	require.NotNil(t, videoSR)

	assert.Equal(t, "480p", payload["resolution"])
	assert.Equal(t, "4k", videoSR.TargetResolution)
	assert.Equal(t, "480p", videoSR.SourceResolution)
}

func TestBuildSubmitPayloadVideoSuperResolutionUsesResolutionLimit(t *testing.T) {
	req := &relaycommon.TaskSubmitReq{
		Model:    "seedance-2",
		Prompt:   "cinematic city",
		Duration: 5,
		Size:     "2560x1440",
		Metadata: map[string]interface{}{},
	}

	payload, videoSR, err := buildSubmitPayload(req, "seedance-2", "seedance-2", true)
	require.NoError(t, err)
	require.NotNil(t, videoSR)

	assert.Equal(t, "480p", payload["resolution"])
	assert.Equal(t, 1440, videoSR.ResolutionLimit)
	assert.Equal(t, "16:9", payload["aspect_ratio"])
}

func TestParseTaskResultCompleted(t *testing.T) {
	disableToapisTOSTransfer(t)

	body, err := common.Marshal(map[string]any{
		"id":         "tsk_vid_123",
		"object":     "generation.task",
		"model":      "seedance-2",
		"status":     "completed",
		"progress":   100,
		"created_at": 1781577600,
		"result": map[string]any{
			"type": "video",
			"data": []any{
				map[string]any{
					"url":    "https://files.example.com/video.mp4",
					"format": "mp4",
				},
			},
		},
	})
	require.NoError(t, err)

	adaptor := &TaskAdaptor{}
	info, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)

	assert.EqualValues(t, model.TaskStatusSuccess, info.Status)
	assert.Equal(t, "tsk_vid_123", info.TaskID)
	assert.Equal(t, "100%", info.Progress)
	assert.Equal(t, "https://files.example.com/video.mp4", info.Url)
}

func TestTOSPresignGetBuildsVolcengineURL(t *testing.T) {
	cfg := tosConfig{
		Bucket:       "ark-acg-ap-southeast-1",
		Region:       "ap-southeast-1",
		Endpoint:     "tos-ap-southeast-1.volces.com",
		AccessKey:    "AKLTexample",
		SecretKey:    "secret",
		PresignHours: 24,
	}
	now := time.Date(2026, 7, 3, 1, 9, 59, 0, time.UTC)

	rawURL := tosPresignGet(cfg, "dreamina-seedance-2-0/video file.mp4", now)
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)

	assert.Equal(t, "ark-acg-ap-southeast-1.tos-ap-southeast-1.volces.com", parsed.Host)
	assert.Equal(t, "/dreamina-seedance-2-0/video%20file.mp4", parsed.EscapedPath())
	values := parsed.Query()
	assert.Equal(t, "TOS4-HMAC-SHA256", values.Get("X-Tos-Algorithm"))
	assert.Equal(t, "AKLTexample/20260703/ap-southeast-1/tos/request", values.Get("X-Tos-Credential"))
	assert.Equal(t, "20260703T010959Z", values.Get("X-Tos-Date"))
	assert.Equal(t, "86400", values.Get("X-Tos-Expires"))
	assert.Equal(t, "host", values.Get("X-Tos-SignedHeaders"))
	assert.Len(t, values.Get("X-Tos-Signature"), 64)
}

func TestToapisTOSConfigFromEnv(t *testing.T) {
	t.Setenv("TOAPIS_TOS_BUCKET", "newapi-video-ap-southeast-1")
	t.Setenv("TOAPIS_TOS_REGION", "ap-southeast-1")
	t.Setenv("TOAPIS_TOS_ENDPOINT", "")
	t.Setenv("TOAPIS_TOS_ACCESS_KEY", "ak")
	t.Setenv("TOAPIS_TOS_SECRET_KEY", "sk")
	t.Setenv("TOAPIS_TOS_PREFIX", "dreamina-seedance-2-0/")
	t.Setenv("TOAPIS_TOS_PRESIGN_HOURS", "48")

	cfg, ok := toapisTOSConfig()
	require.True(t, ok)

	assert.Equal(t, "newapi-video-ap-southeast-1", cfg.Bucket)
	assert.Equal(t, "ap-southeast-1", cfg.Region)
	assert.Equal(t, "tos-ap-southeast-1.volces.com", cfg.Endpoint)
	assert.Equal(t, "dreamina-seedance-2-0", cfg.Prefix)
	assert.EqualValues(t, 48, cfg.PresignHours)
}

func TestToAPISModelAliases(t *testing.T) {
	assert.True(t, isSupportedModel("seedance-2"))
	assert.True(t, isSupportedModel("seedance-2-fast"))
	assert.True(t, isSupportedModel("seedance-2-mini"))
	assert.Equal(t, "seedance-2", toapisModel("dreamina-seedance-2-0-260128"))
	assert.Equal(t, "seedance-2-fast", toapisModel("dreamina-seedance-2-0-fast-260128"))
	assert.Equal(t, "seedance-2-mini", toapisModel("seedance2.0_mini"))
	billingModel, ok := seedanceBillingModel("seedance-2-fast")
	require.True(t, ok)
	assert.Equal(t, "seedance2.0_fast_direct", billingModel)
}

func disableToapisTOSTransfer(t *testing.T) {
	for _, key := range []string{
		"TOAPIS_TOS_BUCKET",
		"TOAPIS_TOS_REGION",
		"TOAPIS_TOS_ENDPOINT",
		"TOAPIS_TOS_ACCESS_KEY",
		"TOAPIS_TOS_SECRET_KEY",
		"TOAPIS_TOS_SECURITY_TOKEN",
		"TOAPIS_TOS_PREFIX",
		"TOAPIS_TOS_PRESIGN_HOURS",
	} {
		t.Setenv(key, "")
	}
}

func TestTOSCanonicalQueryUsesPercentEscaping(t *testing.T) {
	query := tosCanonicalQuery(map[string]string{
		"X-Tos-Credential": "AK/20260703/ap-southeast-1/tos/request",
		"X-Tos-Date":       "20260703T010959Z",
	})

	assert.True(t, strings.Contains(query, "X-Tos-Credential=AK%2F20260703%2Fap-southeast-1%2Ftos%2Frequest"))
	assert.True(t, strings.Contains(query, "X-Tos-Date=20260703T010959Z"))
}
