package hailuo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoResponseUsesOfficialSubmitFormatForOfficialPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/video_generation", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"task_id":"upstream-task",
			"base_resp":{"status_code":0,"status_msg":"success"}
		}`)),
	}
	taskID, _, taskErr := (&TaskAdaptor{}).DoResponse(ctx, resp, &relaycommon.RelayInfo{
		OriginModelName: "MiniMax-Hailuo-02",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "public-task"},
	})

	require.Nil(t, taskErr)
	assert.Equal(t, "upstream-task", taskID)

	var body VideoResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "public-task", body.TaskID)
	assert.Equal(t, StatusSuccess, body.BaseResp.StatusCode)
	assert.Equal(t, "success", body.BaseResp.StatusMsg)
}

func TestConvertToHailuoVideoSanitizesStoredFailure(t *testing.T) {
	task := &model.Task{
		TaskID:     "public-task",
		Status:     model.TaskStatusFailure,
		FailReason: "Invalid API key sk-proj-abcdefghijklmnop",
		PrivateData: model.TaskPrivateData{
			ErrorCode: "AccessDenied",
		},
		Data: []byte(`{
			"task_id":"upstream-task",
			"status":"Fail",
			"base_resp":{"status_code":1004,"status_msg":"Invalid API key sk-proj-abcdefghijklmnop"}
		}`),
	}

	data, err := (&TaskAdaptor{}).ConvertToHailuoVideo(task)

	require.NoError(t, err)
	assert.Contains(t, string(data), "Upstream authentication failed")
	assert.Contains(t, string(data), `"status_code":1`)
	assert.NotContains(t, string(data), "sk-proj-")
}
