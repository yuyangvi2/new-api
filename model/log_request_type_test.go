package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLogsFiltersImageRequests(t *testing.T) {
	truncateTables(t)

	logs := []*Log{
		{UserId: 1, CreatedAt: 100, Type: LogTypeConsume, ModelName: "image-a", Other: `{"request_path":"/v1/images/generations"}`},
		{UserId: 1, CreatedAt: 101, Type: LogTypeConsume, ModelName: "image-b", Other: `{"request_path":"/pg/images/generations"}`},
		{UserId: 1, CreatedAt: 102, Type: LogTypeConsume, ModelName: "chat", Other: `{"request_path":"/v1/chat/completions"}`},
		{UserId: 2, CreatedAt: 103, Type: LogTypeConsume, ModelName: "image-c", Other: `{"request_path":"/v1/images/edits"}`},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	allLogs, total, err := GetAllLogs(LogTypeConsume, 0, 0, "", "", "", 0, 20, 0, "", "", "", "image")
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, allLogs, 3)

	userLogs, userTotal, err := GetUserLogs(1, LogTypeConsume, 0, 0, "", "", 0, 20, "", "", "", "image")
	require.NoError(t, err)
	assert.Equal(t, int64(2), userTotal)
	assert.Len(t, userLogs, 2)
}
