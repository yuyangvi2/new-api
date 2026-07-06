package relay

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
}
