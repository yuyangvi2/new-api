package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildWebhookPayload_WeCom(t *testing.T) {
	url := "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=c6259ee0-2b08-4b3a-8d98-e1f69e613f39"
	notify := dto.NewNotify(dto.NotifyTypeChannelUpdate, "通道「测试渠道」（#1）已被禁用", "通道「测试渠道」（#1）已被禁用，原因：401 Unauthorized", nil)

	payloadBytes, err := buildWebhookPayload(url, notify)
	require.NoError(t, err)

	var payload map[string]any
	err = common.Unmarshal(payloadBytes, &payload)
	require.NoError(t, err)

	assert.Equal(t, "markdown", payload["msgtype"])
	markdown, ok := payload["markdown"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, markdown["content"], "### 通道「测试渠道」（#1）已被禁用")
	assert.Contains(t, markdown["content"], "原因：401 Unauthorized")
}

func TestSendWebhookNotifyPostsPlatformPayload(t *testing.T) {
	fetchSetting := system_setting.GetFetchSetting()
	originalFetchSetting := *fetchSetting
	originalHTTPClient := httpClient
	originalProtectedClient := ssrfProtectedHTTPClient
	t.Cleanup(func() {
		*fetchSetting = originalFetchSetting
		httpClient = originalHTTPClient
		ssrfProtectedHTTPClient = originalProtectedClient
	})
	fetchSetting.EnableSSRFProtection = false
	httpClient = http.DefaultClient
	ssrfProtectedHTTPClient = http.DefaultClient

	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json; charset=utf-8", r.Header.Get("Content-Type"))
		require.NoError(t, common.DecodeJson(r.Body, &payload))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	notify := dto.NewNotify(
		dto.NotifyTypeChannelUpdate,
		"通道「OpenAI官方」（#1）已被禁用",
		"通道「OpenAI官方」（#1）已被系统自动禁用，原因：401 Invalid Authentication",
		nil,
	)

	err := SendWebhookNotify(server.URL, "", notify)
	require.NoError(t, err)
	assert.Equal(t, dto.NotifyTypeChannelUpdate, payload["type"])
	assert.Equal(t, "通道「OpenAI官方」（#1）已被禁用", payload["title"])
	assert.Contains(t, payload["content"], "401 Invalid Authentication")
}
