package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

// WebhookPayload webhook 通知的负载数据
type WebhookPayload struct {
	Type      string        `json:"type"`
	Title     string        `json:"title"`
	Content   string        `json:"content"`
	Values    []interface{} `json:"values,omitempty"`
	Timestamp int64         `json:"timestamp"`
}

// generateSignature 生成 webhook 签名
func generateSignature(secret string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func webhookHostMatches(webhookURL, domain string) bool {
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return false
	}
	host := parsedURL.Hostname()
	return strings.EqualFold(host, domain) || strings.HasSuffix(strings.ToLower(host), "."+domain)
}

// buildWebhookPayload builds raw payload or platform-specific payload (e.g. WeCom / DingTalk / Feishu).
func buildWebhookPayload(webhookURL string, data dto.Notify) ([]byte, error) {
	content := data.Content
	for _, value := range data.Values {
		content = strings.Replace(content, dto.ContentValueParam, fmt.Sprintf("%v", value), 1)
	}

	// 适配企业微信群机器人 (qyapi.weixin.qq.com)
	if webhookHostMatches(webhookURL, "qyapi.weixin.qq.com") {
		markdownText := fmt.Sprintf("### %s\n\n%s", data.Title, content)
		payload := map[string]any{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"content": markdownText,
			},
		}
		return common.Marshal(payload)
	}

	// 适配钉钉群机器人 (oapi.dingtalk.com)
	if webhookHostMatches(webhookURL, "oapi.dingtalk.com") {
		markdownText := fmt.Sprintf("### %s\n\n%s", data.Title, content)
		payload := map[string]any{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"title": data.Title,
				"text":  markdownText,
			},
		}
		return common.Marshal(payload)
	}

	// 适配飞书群机器人 (open.feishu.cn)
	if webhookHostMatches(webhookURL, "open.feishu.cn") {
		payload := map[string]any{
			"msg_type": "text",
			"content": map[string]string{
				"text": fmt.Sprintf("[%s]\n%s", data.Title, content),
			},
		}
		return common.Marshal(payload)
	}

	// 默认通用 Webhook 负载
	payload := WebhookPayload{
		Type:      data.Type,
		Title:     data.Title,
		Content:   content,
		Values:    data.Values,
		Timestamp: time.Now().Unix(),
	}
	return common.Marshal(payload)
}

// SendWebhookNotify 发送 webhook 通知
func SendWebhookNotify(webhookURL string, secret string, data dto.Notify) error {
	payloadBytes, err := buildWebhookPayload(webhookURL, data)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %v", err)
	}

	// 创建 HTTP 请求
	var req *http.Request
	var resp *http.Response

	if system_setting.EnableWorker() {
		// 构建worker请求数据
		workerReq := &WorkerRequest{
			URL:    webhookURL,
			Key:    system_setting.WorkerValidKey,
			Method: http.MethodPost,
			Headers: map[string]string{
				"Content-Type": "application/json; charset=utf-8",
			},
			Body: payloadBytes,
		}

		// 如果有secret，添加签名到headers
		if secret != "" {
			signature := generateSignature(secret, payloadBytes)
			workerReq.Headers["X-Webhook-Signature"] = signature
			workerReq.Headers["Authorization"] = "Bearer " + secret
		}

		resp, err = DoWorkerRequest(workerReq)
		if err != nil {
			return fmt.Errorf("failed to send webhook request through worker: %v", err)
		}
		defer resp.Body.Close()

		// 检查响应状态
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("webhook request failed with status code: %d", resp.StatusCode)
		}
	} else {
		// SSRF防护：验证Webhook URL（非Worker模式）
		if err := ValidateSSRFProtectedFetchURL(webhookURL); err != nil {
			return fmt.Errorf("request reject: %v", err)
		}

		req, err = http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(payloadBytes))
		if err != nil {
			return fmt.Errorf("failed to create webhook request: %v", err)
		}

		// 设置请求头
		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		// 如果有 secret，生成签名
		if secret != "" {
			signature := generateSignature(secret, payloadBytes)
			req.Header.Set("X-Webhook-Signature", signature)
		}

		// 发送请求
		client := GetSSRFProtectedHTTPClient()
		resp, err = client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send webhook request: %v", err)
		}
		defer resp.Body.Close()

		// 检查响应状态
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("webhook request failed with status code: %d", resp.StatusCode)
		}
	}

	return nil
}
