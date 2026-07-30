package controller

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

func ListSeedanceMVideoTasks(c *gin.Context) {
	proxySeedanceMVideoTaskRequest(c, http.MethodGet, "/contents/generations/tasks")
}

func GetSeedanceMVideoTask(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("upstream_task_id"))
	if taskID == "" {
		common.ApiErrorMsg(c, "upstream_task_id is required")
		return
	}
	proxySeedanceMVideoTaskRequest(c, http.MethodGet, "/contents/generations/tasks/"+url.PathEscape(taskID))
}

func DeleteSeedanceMVideoTask(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("upstream_task_id"))
	if taskID == "" {
		common.ApiErrorMsg(c, "upstream_task_id is required")
		return
	}
	proxySeedanceMVideoTaskRequest(c, http.MethodDelete, "/contents/generations/tasks/"+url.PathEscape(taskID))
}

func proxySeedanceMVideoTaskRequest(c *gin.Context, method string, path string) {
	channelID := common.GetContextKeyInt(c, constant.ContextKeyChannelId)
	channel, err := model.CacheGetChannel(channelID)
	if err != nil || channel == nil || channel.Type != constant.ChannelTypeSeedanceM {
		common.ApiErrorMsg(c, "selected channel is not Seedance")
		return
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
	req.Header.Set("Authorization", "Bearer "+common.GetContextKeyString(c, constant.ContextKeyChannelKey))
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
