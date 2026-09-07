package relay

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestTaskModel2PollDtoDoesNotExposeFailReasonAsResultURL(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_public",
		Status:     model.TaskStatusFailure,
		FailReason: "copyright restriction",
		Progress:   "100%",
		Data:       json.RawMessage(`{"status":"failed"}`),
	}

	dto := TaskModel2PollDto(task)

	assert.Equal(t, "task_public", dto.TaskID)
	assert.Equal(t, "FAILURE", dto.Status)
	assert.Equal(t, "copyright restriction", dto.FailReason)
	assert.Empty(t, dto.ResultURL)

	data, err := common.Marshal(dto)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "debug_result")
	assert.Equal(t, "copyright restriction", gjson.GetBytes(data, "error.message").String())
	assert.Equal(t, "upstream_task_failed", gjson.GetBytes(data, "error.code").String())
}

func TestTaskModel2UserDtoSanitizesSensitiveFailureReason(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_public",
		Status:     model.TaskStatusFailure,
		FailReason: "Invalid API key sk-proj-abcdefghijklmnop for upstream account",
		Progress:   "100%",
	}

	userDTO := TaskModel2UserDto(task)
	data, err := common.Marshal(userDTO)

	require.NoError(t, err)
	assert.Equal(t, "Upstream authentication failed, please contact administrator", gjson.GetBytes(data, "fail_reason").String())
	assert.Equal(t, "Upstream authentication failed, please contact administrator", gjson.GetBytes(data, "error.message").String())
	assert.Equal(t, "upstream_authentication_failed", gjson.GetBytes(data, "error.code").String())
	assert.False(t, gjson.GetBytes(data, "result_url").Exists())
	assert.NotContains(t, string(data), "sk-proj-")
}

func TestSanitizeOpenAIVideoTaskResponseOverridesUnsafeAdapterError(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_public",
		Status:     model.TaskStatusFailure,
		FailReason: "Rate limit exceeded, retry after 10 seconds.",
	}
	rawResponse := []byte(`{
		"id":"task_public",
		"object":"video",
		"status":"failed",
		"error":{"code":"AccessDenied","message":"Invalid API key sk-proj-abcdefghijklmnop"},
		"metadata":{"url":"https://internal.example.com/private-result"}
	}`)

	data, err := sanitizeOpenAIVideoTaskResponse(task, rawResponse)

	require.NoError(t, err)
	assert.Equal(t, "Rate limit exceeded, retry after 10 seconds.", gjson.GetBytes(data, "error.message").String())
	assert.Equal(t, "upstream_task_failed", gjson.GetBytes(data, "error.code").String())
	assert.False(t, gjson.GetBytes(data, "metadata.url").Exists())
	assert.NotContains(t, string(data), "sk-proj-")
	assert.NotContains(t, string(data), "internal.example.com")
}
