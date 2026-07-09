package toapis

import (
	"testing"

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

	payload, err := buildSubmitPayload(req, "seedance-2-fast", "seedance-2-fast")
	require.NoError(t, err)

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

	payload, err := buildSubmitPayload(req, "seedance-2-mini", "seedance-2-mini")
	require.NoError(t, err)

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

	_, err := buildSubmitPayload(req, "seedance-2", "seedance-2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audio_with_roles cannot be used alone")
}

func TestParseTaskResultCompleted(t *testing.T) {
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
