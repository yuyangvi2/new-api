package doubao

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedanceOfficialParseTaskResultSuccess(t *testing.T) {
	result, err := (&SeedanceOfficialTaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_123",
		"status":"succeeded",
		"content":{"video_url":"https://example.com/video.mp4"},
		"usage":{"completion_tokens":12,"total_tokens":34}
	}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "task_123", result.TaskID)
	assert.Equal(t, model.TaskStatusSuccess, result.Status)
	assert.Equal(t, taskcommon.ProgressComplete, result.Progress)
	assert.Equal(t, "https://example.com/video.mp4", result.Url)
	assert.Equal(t, 12, result.CompletionTokens)
	assert.Equal(t, 34, result.TotalTokens)
}

func TestSeedanceOfficialParseTaskResultFailure(t *testing.T) {
	result, err := (&SeedanceOfficialTaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_123",
		"status":"failed",
		"error":{"code":"InvalidParameter","message":"bad request"}
	}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, model.TaskStatusFailure, result.Status)
	assert.Equal(t, taskcommon.ProgressComplete, result.Progress)
	assert.Equal(t, "bad request", result.Reason)
}

func TestSeedanceOfficialParseTaskResultErrorWithoutStatus(t *testing.T) {
	result, err := (&SeedanceOfficialTaskAdaptor{}).ParseTaskResult([]byte(`{
		"ResponseMetadata":{"Error":{"Code":"AccessDenied","Message":"invalid api key"}}
	}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, model.TaskStatusFailure, result.Status)
	assert.Equal(t, taskcommon.ProgressComplete, result.Progress)
	assert.Equal(t, "invalid api key", result.Reason)
}

func TestSeedanceOfficialParseTaskResultMissingStatusWithoutError(t *testing.T) {
	result, err := (&SeedanceOfficialTaskAdaptor{}).ParseTaskResult([]byte(`{"id":"task_123"}`))

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestSeedanceOfficialParseTaskResultUnknownStatusFails(t *testing.T) {
	result, err := (&SeedanceOfficialTaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_123",
		"status":"paused"
	}`))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, model.TaskStatusFailure, result.Status)
	assert.Equal(t, taskcommon.ProgressComplete, result.Progress)
	assert.Equal(t, "unknown status: paused", result.Reason)
}

func TestSeedanceOfficialValidateRequestBounds(t *testing.T) {
	hugeDuration := dto.IntValue(3601)
	hugeFrames := dto.IntValue(86401)

	tests := []struct {
		name string
		req  *requestPayload
	}{
		{
			name: "duration too large",
			req: &requestPayload{
				Model:    "doubao-seedance-2-0-260128",
				Content:  []ContentItem{{Type: "text", Text: "make a video"}},
				Duration: &hugeDuration,
			},
		},
		{
			name: "frames too large",
			req: &requestPayload{
				Model:   "doubao-seedance-2-0-260128",
				Content: []ContentItem{{Type: "text", Text: "make a video"}},
				Frames:  &hugeFrames,
			},
		},
		{
			name: "unsupported ratio",
			req: &requestPayload{
				Model:   "doubao-seedance-2-0-260128",
				Content: []ContentItem{{Type: "text", Text: "make a video"}},
				Ratio:   "999999999:1",
			},
		},
		{
			name: "unsupported resolution",
			req: &requestPayload{
				Model:      "doubao-seedance-2-0-260128",
				Content:    []ContentItem{{Type: "text", Text: "make a video"}},
				Resolution: "999999999p",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, validateSeedanceOfficialRequest(tt.req))
		})
	}
}

func TestSeedanceOfficialValidateRequestAllowsAutoDuration(t *testing.T) {
	autoDuration := dto.IntValue(-1)
	req := &requestPayload{
		Model:      "doubao-seedance-2-0-260128",
		Content:    []ContentItem{{Type: "text", Text: "make a video"}},
		Duration:   &autoDuration,
		Ratio:      "16:9",
		Resolution: "720p",
	}

	assert.NoError(t, validateSeedanceOfficialRequest(req))
}
