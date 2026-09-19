package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoProxyAllowsPrivateProviderBaseURL(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Task{}))

	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})
	service.InitHttpClient()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/videos/upstream-task/content", r.URL.Path)
		assert.Equal(t, "Bearer upstream-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("video-content"))
	}))
	defer upstream.Close()

	baseURL := upstream.URL
	require.NoError(t, db.Create(&model.Channel{
		Id:      91,
		Type:    constant.ChannelTypeXai,
		Key:     "upstream-key",
		Name:    "private-xai",
		BaseURL: &baseURL,
	}).Error)
	require.NoError(t, db.Create(&model.Task{
		TaskID:    "public-task",
		UserId:    17,
		ChannelId: 91,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "upstream-task",
		},
	}).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/public-task/content", nil)
	c.Params = gin.Params{{Key: "task_id", Value: "public-task"}}
	c.Set("id", 17)

	VideoProxy(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "video/mp4", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "video-content", recorder.Body.String())
}

func TestVideoProxyStillBlocksPrivateUpstreamResultURL(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Task{}))

	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})
	service.InitHttpClient()

	requested := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = true
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	require.NoError(t, db.Create(&model.Channel{
		Id:   92,
		Type: constant.ChannelTypeKling,
		Key:  "upstream-key",
		Name: "untrusted-result-url",
	}).Error)
	require.NoError(t, db.Create(&model.Task{
		TaskID:    "untrusted-task",
		UserId:    17,
		ChannelId: 92,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			ResultURL: upstream.URL + "/video.mp4",
		},
	}).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/untrusted-task/content", nil)
	c.Params = gin.Params{{Key: "task_id", Value: "untrusted-task"}}
	c.Set("id", 17)

	VideoProxy(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	assert.False(t, requested)
	assert.Contains(t, recorder.Body.String(), "request blocked")
}
