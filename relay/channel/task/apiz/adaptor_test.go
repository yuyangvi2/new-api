package apiz

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSubmitPayloadSeedanceAlias(t *testing.T) {
	req := &relaycommon.TaskSubmitReq{
		Model:    "seedance2.0_fast_vision",
		Prompt:   "make it move",
		Image:    "https://example.com/input.png",
		Duration: 5,
		Size:     "1280x720",
		Metadata: map[string]interface{}{
			"model":          "seedance_2.0",
			"watermark":      false,
			"generate_audio": true,
		},
	}

	payload, err := buildSubmitPayload(req, "seedance2.0_fast_vision", seedanceID)
	if err != nil {
		t.Fatalf("buildSubmitPayload returned error: %v", err)
	}
	if payload.Model != seedanceID {
		t.Fatalf("model = %q, want %q", payload.Model, seedanceID)
	}
	if got := payload.Params["model"]; got != "seedance_2.0_fast" {
		t.Fatalf("params.model = %v, want seedance_2.0_fast", got)
	}
	if got := payload.Params["image_url"]; got != req.Image {
		t.Fatalf("image_url = %v, want %q", got, req.Image)
	}
	if got := payload.Params["ratio"]; got != "16:9" {
		t.Fatalf("ratio = %v, want 16:9", got)
	}
	if got := payload.Params["duration"]; got != 5 {
		t.Fatalf("duration = %v, want 5", got)
	}
}

func TestParseTaskResultSuccess(t *testing.T) {
	body, err := common.Marshal(map[string]any{
		"code": 200,
		"data": map[string]any{
			"task_id":           "task-upstream",
			"status":            "completed",
			"progress":          100,
			"total_tokens":      1200,
			"completion_tokens": 1000,
			"result": map[string]any{
				"video_url": "https://cdn.example.com/out.mp4",
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	adaptor := &TaskAdaptor{}
	info, err := adaptor.ParseTaskResult(body)
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if info.Status != model.TaskStatusSuccess {
		t.Fatalf("status = %q, want %q", info.Status, model.TaskStatusSuccess)
	}
	if info.Url != "https://cdn.example.com/out.mp4" {
		t.Fatalf("url = %q", info.Url)
	}
	if info.TotalTokens != 1200 || info.CompletionTokens != 1000 {
		t.Fatalf("tokens = %d/%d", info.CompletionTokens, info.TotalTokens)
	}
}

func TestParseTaskResultFailureUsesNestedProviderError(t *testing.T) {
	body, err := common.Marshal(map[string]any{
		"code":    200,
		"message": "查询成功",
		"data": map[string]any{
			"task_id":  "task-upstream",
			"status":   "failed",
			"progress": 100,
			"result": map[string]any{
				"error": "copyright restriction",
			},
			"output": map[string]any{
				"error": "fallback error",
			},
		},
	})
	require.NoError(t, err)

	adaptor := &TaskAdaptor{}
	info, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)

	assert.EqualValues(t, model.TaskStatusFailure, info.Status)
	assert.Equal(t, "copyright restriction", info.Reason)
	assert.Equal(t, "100%", info.Progress)
}

func TestVariantForModelGenericArkDefaultsToFast(t *testing.T) {
	if got := variantForModel(seedanceID); got != "seedance_2.0_fast" {
		t.Fatalf("variantForModel(%q) = %q, want seedance_2.0_fast", seedanceID, got)
	}
}

func TestVariantForModelSeedanceMini(t *testing.T) {
	assert.True(t, isSupportedModel("seedance2.0_mini"))
	assert.True(t, isSupportedModel("seedance2.0_fast_mini"))
	assert.Equal(t, "seedance_2.0_mini", variantForModel("seedance2.0_mini"))
	assert.Equal(t, "seedance_2.0_fast_mini", variantForModel("seedance2.0_fast_mini"))
}
