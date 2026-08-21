package controller

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/service"
)

func ListSeedanceMVideoTasks(c *gin.Context) {
	platform := constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSeedanceM))
	tasks := model.TaskGetAllUserTask(c.GetInt("id"), 0, constant.TaskQueryLimit, model.SyncTaskQueryParams{
		Platform: platform,
	})
	items := make([]any, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, relay.TaskModel2UserDto(task))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func GetSeedanceMVideoTask(c *gin.Context) {
	task, channel, ok := ownedSeedanceMTask(c)
	if !ok {
		return
	}
	proxySeedanceMVideoTaskRequest(c, task, channel, http.MethodGet, "/contents/generations/tasks/"+url.PathEscape(task.GetUpstreamTaskID()))
}

func DeleteSeedanceMVideoTask(c *gin.Context) {
	task, channel, ok := ownedSeedanceMTask(c)
	if !ok {
		return
	}
	proxySeedanceMVideoTaskRequest(c, task, channel, http.MethodDelete, "/contents/generations/tasks/"+url.PathEscape(task.GetUpstreamTaskID()))
}

func ownedSeedanceMTask(c *gin.Context) (*model.Task, *model.Channel, bool) {
	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		common.ApiErrorMsg(c, "task_id is required")
		return nil, nil, false
	}
	task, exists, err := model.GetByTaskId(c.GetInt("id"), taskID)
	if err != nil {
		common.ApiErrorMsg(c, "failed to query task")
		return nil, nil, false
	}
	if !exists || task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "task not found"}})
		return nil, nil, false
	}
	channel, err := model.CacheGetChannel(task.ChannelId)
	if err != nil || channel == nil || channel.Type != constant.ChannelTypeSeedanceM {
		common.ApiErrorMsg(c, "task channel is not available")
		return nil, nil, false
	}
	return task, channel, true
}

func proxySeedanceMVideoTaskRequest(c *gin.Context, task *model.Task, channel *model.Channel, method string, path string) {
	apiKey := strings.TrimSpace(task.PrivateData.Key)
	if apiKey == "" {
		var keyErr error
		apiKey, _, keyErr = channel.GetNextEnabledKey()
		if keyErr != nil {
			common.ApiErrorMsg(c, "task channel has no available key")
			return
		}
	}
	base := strings.TrimRight(channel.GetBaseURL(), "/")
	if base == "" {
		base = strings.TrimRight(constant.ChannelBaseURLs[constant.ChannelTypeSeedanceM], "/")
	}
	query := c.Request.URL.Query()
	query.Del("model")
	query.Del("group")
	targetURL := base + path
	if len(query) > 0 {
		targetURL += "?" + query.Encode()
	}
	req, err := http.NewRequest(method, targetURL, nil)
	if err != nil {
		common.ApiErrorMsg(c, "request failed")
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	settings := channel.GetOtherSettings().SeedanceM
	if settings != nil && strings.TrimSpace(settings.ServiceVersion) != "" {
		req.Header.Set("service-version", strings.TrimSpace(settings.ServiceVersion))
	}
	client, err := service.GetHttpClientWithProxy(channel.GetSetting().Proxy)
	if err != nil {
		common.ApiErrorMsg(c, "request failed")
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		common.ApiErrorMsg(c, "request failed")
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSeedanceMProxyResponseBytes+1))
	if err != nil {
		common.ApiErrorMsg(c, "request failed")
		return
	}
	if len(body) > maxSeedanceMProxyResponseBytes {
		common.ApiErrorMsg(c, "response is too large")
		return
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		c.JSON(resp.StatusCode, gin.H{
			"error": gin.H{
				"message": "request failed",
			},
		})
		return
	}
	c.Data(resp.StatusCode, firstNonEmpty(resp.Header.Get("Content-Type"), "application/json"), body)
}
