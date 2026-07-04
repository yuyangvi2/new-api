// Package vipeak 实现 vipeak（www.123vips.com）第三方生成服务的任务型适配器（派发式）。
//
// 上游：https://www.123vips.com，Bearer 鉴权，异步 submit → 轮询 generation-records。
// 一个渠道类型(9003) 内按模型名自动派发到对应 provider / 端点：
//   - wan2.7-i2v-2026-04-25            : POST /api/advanced/generate   provider=wan27        (图生视频, 720p/1080p)
//   - wan2.7-image-pro                 : POST /api/wan27/image-edit    provider=wan27-image  (文生图/图像编辑, 1K/2K/4K, 4K 仅文生图)
//   - dreamina-seedance-2-0-260128     : POST /api/advanced/generate   provider=seedance     (视频, 480p/720p/1080p/4k)
//   - dreamina-seedance-2-0-fast-260128: POST /api/advanced/generate   provider=seedance     (视频, 480p/720p)
//
// 轮询：GET /api/generation-records/<taskId>，返回本地生成记录（record，含 progress / result URL / billing / prompt）。
//
// 注意：
//   - 上游在 Cloudflare 后面，Go 默认 UA "Go-http-client/1.1" 会被 1010 拦截，必须带浏览器 UA。
//   - 生成的视频 URL 24 小时后过期，结果应及时转存。
//   - apiKey 格式：直接是 Bearer token（sk-...），无需切分。
package vipeak

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
)

const (
	vipeakBaseURL = "https://www.123vips.com"

	// 浏览器 UA，绕过 Cloudflare 1010（数据中心 IP + 默认 Go UA 会被拦）。
	browserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"

	epAdvanced  = "/api/advanced/generate"
	epImageEdit = "/api/wan27/image-edit"
)

// modelDef 定义一个对外模型名的 vipeak 参数映射。
type modelDef struct {
	provider      string // 上游 provider 字段
	endpoint      string // 提交端点
	kind          string // "video" | "image"
	upstreamModel string // 非空时把对外别名转换成上游模型名
}

var modelDefs = map[string]modelDef{
	"wan2.7-i2v-2026-04-25": {
		provider: "wan27",
		endpoint: epAdvanced,
		kind:     "video",
	},
	"wan2.7-image-pro": {
		provider: "wan27-image",
		endpoint: epImageEdit,
		kind:     "image",
	},
	"dreamina-seedance-2-0-260128": {
		provider: "seedance",
		endpoint: epAdvanced,
		kind:     "video",
	},
	"dreamina-seedance-2-0-fast-260128": {
		provider: "seedance",
		endpoint: epAdvanced,
		kind:     "video",
	},
	// 兼容模型广场/渠道配置中常见的短名；上游仍使用完整模型名。
	"seedance": {
		provider:      "seedance",
		endpoint:      epAdvanced,
		kind:          "video",
		upstreamModel: "dreamina-seedance-2-0-260128",
	},
	"seedance-fast": {
		provider:      "seedance",
		endpoint:      epAdvanced,
		kind:          "video",
		upstreamModel: "dreamina-seedance-2-0-fast-260128",
	},
}

func canonicalModelName(requestModel string, def modelDef) string {
	if def.upstreamModel != "" {
		return def.upstreamModel
	}
	return requestModel
}

// ============================ 响应结构 ============================

// submitResponse 提交成功时的响应封装：
//
//	{ "ok": true, "taskId": "...", "task": {"status": "..."}, "record": {...}, "user": {...} }
//
// taskId 兜底路径：顶层 taskId → task.taskId → record.taskId。
type submitResponse struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Detail  string `json:"detail"`
	TaskID  string `json:"taskId"`
	Task    struct {
		TaskID string `json:"taskId"`
		Status string `json:"status"`
	} `json:"task"`
	Record map[string]any `json:"record"`
}

func (r *submitResponse) resolveTaskID() string {
	if r.TaskID != "" {
		return r.TaskID
	}
	if r.Task.TaskID != "" {
		return r.Task.TaskID
	}
	if r.Record != nil {
		if v, ok := r.Record["taskId"].(string); ok && v != "" {
			return v
		}
		if v, ok := r.Record["id"].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// ============================ 适配器 ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) base() string {
	if a.baseURL != "" {
		return normalizeBaseURL(a.baseURL)
	}
	return vipeakBaseURL
}

func normalizeBaseURL(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	for _, suffix := range []string{
		epAdvanced,
		epImageEdit,
		"/api/generation-records",
		"/api",
	} {
		base = strings.TrimSuffix(base, suffix)
	}
	return base
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	var peek relaycommon.TaskSubmitReq
	_ = common.UnmarshalBodyReusable(c, &peek)
	upModel := strings.ToLower(peek.Model)
	def, ok := modelDefs[upModel]
	if !ok {
		supported := make([]string, 0, len(modelDefs))
		for k := range modelDefs {
			supported = append(supported, k)
		}
		sort.Strings(supported)
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("unsupported model: %s (supported: %s)", peek.Model, strings.Join(supported, ", ")),
			"invalid_model", http.StatusBadRequest)
	}
	// action 用 provider 名占位（存进 task.Action，轮询时不需要它，但保持统一约定）。
	return relaycommon.ValidateBasicTaskRequest(c, info, def.provider)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	def, ok := modelDefs[strings.ToLower(info.UpstreamModelName)]
	if !ok {
		return "", fmt.Errorf("unsupported model: %s", info.UpstreamModelName)
	}
	return a.base() + def.endpoint, nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req := v.(relaycommon.TaskSubmitReq)

	body := buildRequest(&req, info.UpstreamModelName)

	// metadata 透传/覆盖（供 provider 特有参数：ratio, resolution, duration,
	// generateAudio, watermark, prompt_extend, n, mediaMode, seedanceMode,
	// referenceImages, firstFrameUrl, imageAssetIds 等）。
	if err := taskcommon.UnmarshalMetadata(req.Metadata, &body); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}

	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// buildRequest 按 provider 构造 vipeak 请求体。标准字段做基础映射，
// 其余 provider 特有参数由 metadata 透传补充/覆盖。
func buildRequest(req *relaycommon.TaskSubmitReq, upstreamModel string) map[string]any {
	def := modelDefs[strings.ToLower(upstreamModel)]
	canonicalModel := canonicalModelName(upstreamModel, def)
	m := map[string]any{
		"provider": def.provider,
		"model":    canonicalModel,
	}

	if req.Prompt != "" {
		m["prompt"] = req.Prompt
	}

	// ratio / resolution：Size 若形如 "W:H" 视为 ratio，否则视为 resolution。
	if req.Size != "" {
		if strings.Contains(req.Size, ":") {
			m["ratio"] = req.Size
		} else {
			m["resolution"] = req.Size
		}
	}

	// 输入图：按 provider 语义映射。
	firstImage := req.Image
	if firstImage == "" && len(req.Images) > 0 {
		firstImage = req.Images[0]
	}

	switch def.provider {
	case "wan27": // 图生视频：firstFrameUrl + mediaMode
		if firstImage != "" {
			m["firstFrameUrl"] = firstImage
			m["mediaMode"] = "first_frame"
		}
		if req.Duration > 0 {
			m["duration"] = req.Duration
		}
		m["parameters"] = map[string]any{
			"prompt_extend": false,
			"watermark":     false,
		}

	case "seedance": // 视频：referenceImages（多图）
		imgs := collectImages(req)
		if len(imgs) > 0 {
			refs := make([]map[string]any, 0, len(imgs))
			for i, u := range imgs {
				refs = append(refs, map[string]any{
					"url":      u,
					"fileName": fmt.Sprintf("image%d.png", i+1),
				})
			}
			m["referenceImages"] = refs
			m["seedanceMode"] = "reference_images"
		}
		if req.Duration > 0 {
			m["duration"] = req.Duration
		}
		m["generateAudio"] = true
		m["watermark"] = false

	case "wan27-image": // 文生图 / 图像编辑
		m["imageAssetIds"] = []string{}
		m["parameters"] = map[string]any{
			"n":         1,
			"watermark": false,
		}
	}

	return m
}

// collectImages 汇总单图 + 多图（去空）。
func collectImages(req *relaycommon.TaskSubmitReq) []string {
	var out []string
	if req.Image != "" {
		out = append(out, req.Image)
	}
	for _, u := range req.Images {
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	setCommonHeaders(req, a.apiKey)
	return nil
}

// setCommonHeaders 统一设置 Bearer + 浏览器 UA + JSON 头。
func setCommonHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", browserUA)
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	var r submitResponse
	if err = common.Unmarshal(responseBody, &r); err != nil {
		taskErr = service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}
	if !r.Ok {
		msg := strings.TrimSpace(r.Message + " " + r.Detail)
		if msg == "" {
			msg = "submit failed"
		}
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("%s", msg), "task_failed", http.StatusBadRequest)
		return
	}
	tid := r.resolveTaskID()
	if tid == "" {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("empty taskId in response"), "task_failed", http.StatusBadRequest)
		return
	}
	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return tid, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	base := strings.TrimRight(baseUrl, "/")
	if base == "" {
		base = vipeakBaseURL
	}
	url := base + "/api/generation-records/" + taskID
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	setCommonHeaders(req, key)
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	// 先整体反序列化为 map，兼容 record 嵌套 / 顶层平铺两种形态。
	var envelope map[string]any
	if err := common.Unmarshal(respBody, &envelope); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}

	if ok, exists := envelope["ok"].(bool); exists && !ok {
		msg, _ := envelope["message"].(string)
		if msg == "" {
			msg = "query failed"
		}
		return nil, fmt.Errorf("%s", msg)
	}

	// 取 record（嵌套优先，否则用顶层）。
	rec := envelope
	if inner, ok := envelope["record"].(map[string]any); ok {
		rec = inner
	}

	info := &relaycommon.TaskInfo{}
	status := strings.ToLower(strings.TrimSpace(asString(rec["status"])))

	switch {
	case isSuccessStatus(status):
		info.Status = model.TaskStatusSuccess
		info.Url = extractResultURL(rec)
		info.CompletionTokens = 1
		info.TotalTokens = 1
	case isFailureStatus(status):
		info.Status = model.TaskStatusFailure
		info.Reason = firstNonEmpty(
			asString(rec["error"]),
			asString(rec["errorMessage"]),
			asString(rec["failReason"]),
			asString(rec["message"]),
		)
		if info.Reason == "" {
			info.Reason = "generation failed"
		}
	case status == "" || isInProgressStatus(status):
		// 用 progress 粗判：有 URL 且进度满则视为成功兜底。
		if u := extractResultURL(rec); u != "" && isComplete(rec) {
			info.Status = model.TaskStatusSuccess
			info.Url = u
			info.CompletionTokens = 1
			info.TotalTokens = 1
		} else {
			info.Status = model.TaskStatusInProgress
		}
	default:
		info.Status = model.TaskStatusInProgress
	}
	return info, nil
}

// ============================ 状态与字段解析辅助 ============================

func isSuccessStatus(s string) bool {
	switch s {
	case "completed", "complete", "success", "succeeded", "done", "finished", "ok":
		return true
	}
	return false
}

func isFailureStatus(s string) bool {
	switch s {
	case "failed", "failure", "error", "cancelled", "canceled", "rejected":
		return true
	}
	return false
}

func isInProgressStatus(s string) bool {
	switch s {
	case "pending", "queued", "queueing", "processing", "running", "in_progress", "inprogress", "generating", "submitted", "init", "waiting":
		return true
	}
	return false
}

// isComplete 依据 progress 字段判断是否 100%。
func isComplete(rec map[string]any) bool {
	switch p := rec["progress"].(type) {
	case float64:
		return p >= 100
	case string:
		return strings.HasPrefix(strings.TrimSpace(p), "100")
	}
	return false
}

// extractResultURL 在 record 中防御式地查找结果 URL（字段名未知，多路兜底）。
func extractResultURL(rec map[string]any) string {
	// 直接的字符串 URL 字段
	for _, k := range []string{
		"resultUrl", "videoUrl", "imageUrl", "outputUrl", "url",
		"resultURL", "videoURL", "imageURL", "downloadUrl", "fileUrl",
	} {
		if v, ok := rec[k].(string); ok && looksLikeURL(v) {
			return v
		}
	}
	// 数组/对象字段：outputs / results / assets / images / videos [0]
	for _, k := range []string{"outputs", "results", "assets", "images", "videos", "resultUrls", "urls"} {
		if u := urlFromAny(rec[k]); u != "" {
			return u
		}
	}
	// result 可能是对象或数组或字符串
	if u := urlFromAny(rec["result"]); u != "" {
		return u
	}
	return ""
}

// urlFromAny 从任意（数组/对象/字符串）结构里挖第一个像 URL 的字符串。
func urlFromAny(v any) string {
	switch t := v.(type) {
	case string:
		if looksLikeURL(t) {
			return t
		}
	case []any:
		for _, item := range t {
			if u := urlFromAny(item); u != "" {
				return u
			}
		}
	case map[string]any:
		for _, k := range []string{"url", "resultUrl", "videoUrl", "imageUrl", "downloadUrl", "fileUrl"} {
			if s, ok := t[k].(string); ok && looksLikeURL(s) {
				return s
			}
		}
	}
	return ""
}

func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%v", t)
	case bool:
		return fmt.Sprintf("%v", t)
	}
	return ""
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func (a *TaskAdaptor) GetModelList() []string {
	out := make([]string, 0, len(modelDefs))
	for m := range modelDefs {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

func (a *TaskAdaptor) GetChannelName() string {
	return "vipeak"
}
