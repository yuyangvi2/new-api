// Package vclm 实现腾讯云 VCLM（视频创作大模型）任务适配器（派发式）。
//
// 上游：vclm.tencentcloudapi.com，TC3-HMAC-SHA256 签名，异步 submit/query。
// 一个渠道类型(9001) 内按「模型族 + 有无图」自动派发到对应 VCLM Action：
//   - Kling      : 有图→SubmitImageToVideoJob / 无图→SubmitTextToVideoJob
//   - Vidu       : 有图→SubmitImageToVideoViduJob / 无图→SubmitTextToVideoViduJob
//   - Hunyuan    : SubmitHunyuanToVideoJob
//   - General    : SubmitImageToVideoGeneralJob
// 其它能力（换脸/数字人/模板/视频编辑等）可后续往 modelDefs / buildRequest 里加。
//
// task.Action 存「提交 Action 名」，轮询时 FetchTask 据此推出对应 Describe Action。
// 各 Action 特有参数可由调用方通过 metadata（PascalCase 键）透传/覆盖。
// apiKey 格式：SecretId|SecretKey
package vclm

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
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
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

const (
	vclmHost    = "vclm.tencentcloudapi.com"
	vclmService = "vclm"
	vclmVersion = "2024-05-23"
	vclmRegion  = "ap-guangzhou"

	ctxPayloadKey = "vclm_payload"
	ctxActionKey  = "vclm_action"
)

// modelDef 定义一个对外模型名属于哪个能力族、以及映射到 VCLM 的 Model 参数值。
type modelDef struct {
	family    string // kling | vidu | hunyuan | general
	vclmModel string // VCLM Model 参数值；为空表示该 Action 不接受 Model
}

var modelDefs = map[string]modelDef{
	// Kling（Model 用版本码）
	"kling-v1":          {"kling", "v1.0"},
	"kling-v1-5":        {"kling", "v1.5"},
	"kling-v1-6":        {"kling", "v1.6"},
	"kling-v2-master":   {"kling", "v2.0"},
	"kling-v2-1":        {"kling", "v2.1"},
	"kling-v2-1-master": {"kling", "v2.1m"},
	"kling-v2-5-turbo":  {"kling", "v2.5"},
	"kling-v2-6":        {"kling", "v2.6"},
	"kling-v3":          {"kling", "v3.0"},
	// Vidu
	"vidu-q1":  {"vidu", "viduq1"},
	"vidu-q2":  {"vidu", "viduq2"},
	"vidu-2.0": {"vidu", "vidu2.0"},
	"vidu-1.5": {"vidu", "vidu1.5"},
	// 混元生视频
	"hunyuan-video": {"hunyuan", ""},
	// 通用图生视频
	"general-i2v": {"general", ""},
}

// resolveSubmitAction 按模型族 + 是否有图，选出 VCLM 提交 Action。
func resolveSubmitAction(modelName string, hasImage bool) (string, bool) {
	d, ok := modelDefs[modelName]
	if !ok {
		return "", false
	}
	switch d.family {
	case "kling":
		if hasImage {
			return "SubmitImageToVideoJob", true
		}
		return "SubmitTextToVideoJob", true
	case "vidu":
		if hasImage {
			return "SubmitImageToVideoViduJob", true
		}
		return "SubmitTextToVideoViduJob", true
	case "hunyuan":
		return "SubmitHunyuanToVideoJob", true
	case "general":
		return "SubmitImageToVideoGeneralJob", true
	}
	return "", false
}

// ============================ 响应结构（各 video Action 通用）============================

type tcImage struct {
	Url    string `json:"Url,omitempty"`
	Base64 string `json:"Base64,omitempty"`
}

type tcError struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

type submitResponse struct {
	Response struct {
		JobId          string   `json:"JobId"`
		ExternalTaskId string   `json:"ExternalTaskId"`
		RequestId      string   `json:"RequestId"`
		Error          *tcError `json:"Error,omitempty"`
	} `json:"Response"`
}

type queryResponse struct {
	Response struct {
		Status             string   `json:"Status"`
		ErrorCode          string   `json:"ErrorCode"`
		ErrorMessage       string   `json:"ErrorMessage"`
		ResultVideoUrl     string   `json:"ResultVideoUrl"`
		VideoId            string   `json:"VideoId"`
		Duration           string   `json:"Duration"`
		FinalUnitDeduction string   `json:"FinalUnitDeduction"`
		RequestId          string   `json:"RequestId"`
		Error              *tcError `json:"Error,omitempty"`
	} `json:"Response"`
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

// ValidateRequestAndSetAction 先窥探模型+是否有图，确定 VCLM 提交 Action，
// 把它作为 task.Action 存下（轮询时据此推 Describe）。
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	var peek relaycommon.TaskSubmitReq
	_ = common.UnmarshalBodyReusable(c, &peek)
	hasImage := strings.TrimSpace(peek.Image) != "" || len(peek.Images) > 0
	action, ok := resolveSubmitAction(peek.Model, hasImage)
	if !ok {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("unsupported model: %s", peek.Model), "invalid_model", http.StatusBadRequest)
	}
	return relaycommon.ValidateBasicTaskRequest(c, info, action)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return "https://" + vclmHost + "/", nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req := v.(relaycommon.TaskSubmitReq)
	action := info.Action

	body := buildRequest(action, &req, info.UpstreamModelName)

	// metadata 透传/覆盖（PascalCase 键，供各 Action 特有参数）
	if err := taskcommon.UnmarshalMetadata(req.Metadata, &body); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}

	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	c.Set(ctxPayloadKey, data)
	c.Set(ctxActionKey, action)
	return bytes.NewReader(data), nil
}

// buildRequest 按 Action 构造 VCLM 请求体（只放该 Action 接受的常用字段）。
func buildRequest(action string, req *relaycommon.TaskSubmitReq, upstreamModel string) map[string]any {
	m := map[string]any{}
	if req.Prompt != "" {
		m["Prompt"] = req.Prompt
	}
	vclmModel := modelDefs[upstreamModel].vclmModel
	firstImage := req.Image
	if firstImage == "" && len(req.Images) > 0 {
		firstImage = req.Images[0]
	}
	dur := taskcommon.DefaultInt(req.Duration, 5)
	mode := taskcommon.DefaultString(req.Mode, "std")

	switch action {
	case "SubmitImageToVideoJob": // Kling i2v
		m["Model"] = vclmModel
		m["Image"] = tcImage{Url: firstImage}
		m["Duration"] = strconv.Itoa(dur)
		m["Mode"] = mode
	case "SubmitTextToVideoJob": // Kling t2v
		m["Model"] = vclmModel
		m["Duration"] = strconv.Itoa(dur)
		m["Mode"] = mode
	case "SubmitImageToVideoGeneralJob": // 通用 i2v（无 Model）
		m["Image"] = tcImage{Url: firstImage}
	case "SubmitHunyuanToVideoJob": // 混元
		if firstImage != "" {
			m["Image"] = tcImage{Url: firstImage}
		}
	case "SubmitImageToVideoViduJob": // Vidu i2v
		m["Model"] = vclmModel
		imgs := req.Images
		if len(imgs) == 0 && firstImage != "" {
			imgs = []string{firstImage}
		}
		m["Images"] = imgs
		m["Duration"] = dur
	case "SubmitTextToVideoViduJob": // Vidu t2v
		m["Model"] = vclmModel
		m["Duration"] = dur
	}
	return m
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	payload, _ := c.Get(ctxPayloadKey)
	body, _ := payload.([]byte)
	action, _ := c.Get(ctxActionKey)
	actStr, _ := action.(string)
	if actStr == "" {
		actStr = info.Action
	}
	secretId, secretKey, err := splitKey(a.apiKey)
	if err != nil {
		return err
	}
	for k, v := range signTC3(secretId, secretKey, actStr, body, time.Now().Unix()) {
		req.Header.Set(k, v)
	}
	return nil
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
	if r.Response.Error != nil {
		taskErr = service.TaskErrorWrapperLocal(
			fmt.Errorf("%s: %s", r.Response.Error.Code, r.Response.Error.Message),
			"task_failed", http.StatusBadRequest)
		return
	}
	if r.Response.JobId == "" {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("empty JobId"), "task_failed", http.StatusBadRequest)
		return
	}
	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return r.Response.JobId, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}
	// 提交 Action 存在 task.Action 里；据此推出 Describe Action。
	submitAction, _ := body["action"].(string)
	queryAction := strings.Replace(submitAction, "Submit", "Describe", 1)
	if queryAction == "" || !strings.HasPrefix(queryAction, "Describe") {
		queryAction = "DescribeImageToVideoJob" // 兜底
	}
	secretId, secretKey, err := splitKey(key)
	if err != nil {
		return nil, err
	}
	payload, err := common.Marshal(map[string]string{"JobId": taskID})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://"+vclmHost+"/", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	for k, v := range signTC3(secretId, secretKey, queryAction, payload, time.Now().Unix()) {
		req.Header.Set(k, v)
	}
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var r queryResponse
	if err := common.Unmarshal(respBody, &r); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}
	if r.Response.Error != nil {
		return nil, fmt.Errorf("%s: %s", r.Response.Error.Code, r.Response.Error.Message)
	}
	info := &relaycommon.TaskInfo{}
	switch r.Response.Status {
	case "WAIT":
		info.Status = model.TaskStatusSubmitted
	case "RUN":
		info.Status = model.TaskStatusInProgress
	case "DONE":
		info.Status = model.TaskStatusSuccess
		info.Url = r.Response.ResultVideoUrl
		if d, err := strconv.ParseFloat(r.Response.FinalUnitDeduction, 64); err == nil {
			if rounded := int(math.Ceil(d)); rounded > 0 {
				info.CompletionTokens = rounded
				info.TotalTokens = rounded
			}
		}
	case "FAIL":
		info.Status = model.TaskStatusFailure
		info.Reason = strings.TrimSpace(r.Response.ErrorCode + " " + r.Response.ErrorMessage)
	default:
		return nil, fmt.Errorf("unknown task status: %s", r.Response.Status)
	}
	return info, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	out := make([]string, 0, len(modelDefs))
	for m := range modelDefs {
		out = append(out, m)
	}
	return out
}

func (a *TaskAdaptor) GetChannelName() string {
	return "vclm"
}

// AdjustBillingOnComplete 按 VCLM 实际计费单元(FinalUnitDeduction)精确结算。
// 实际额度 = 计费单元数 × 模型倍率 × 组倍率 × QuotaPerUnit。
// 此处模型倍率语义 = 「每个计费单元的美元价」（如 ¥1/单元 → 倍率≈0.139）。
// 这样预扣(modelRatio/2 × QuotaPerUnit ≈ 半美元级)不会爆炸，完成时再按真实用量补/退。
// 未配倍率则返回 0（回退到预扣额度不变）。
func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int {
	if taskResult == nil || taskResult.TotalTokens <= 0 {
		return 0
	}
	modelName := task.Properties.OriginModelName
	if bc := task.PrivateData.BillingContext; bc != nil && bc.OriginModelName != "" {
		modelName = bc.OriginModelName
	}
	modelRatio, ok, _ := ratio_setting.GetModelRatio(modelName)
	if !ok || modelRatio <= 0 {
		return 0
	}
	groupRatio := 1.0
	if task.Group != "" {
		groupRatio = ratio_setting.GetGroupRatio(task.Group)
	}
	return int(float64(taskResult.TotalTokens) * modelRatio * groupRatio * common.QuotaPerUnit)
}

// ============================ TC3 签名 ============================

func splitKey(apiKey string) (secretId, secretKey string, err error) {
	parts := strings.Split(apiKey, "|")
	if len(parts) != 2 {
		return "", "", errors.New("invalid api_key, required format is SecretId|SecretKey")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func sha256hex(s []byte) string {
	h := sha256.Sum256(s)
	return hex.EncodeToString(h[:])
}

func hmacSha256(s, key []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(s)
	return m.Sum(nil)
}

func signTC3(secretId, secretKey, action string, payload []byte, timestamp int64) map[string]string {
	const algorithm = "TC3-HMAC-SHA256"
	const contentType = "application/json; charset=utf-8"

	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-tc-action:%s\n",
		contentType, vclmHost, strings.ToLower(action))
	signedHeaders := "content-type;host;x-tc-action"
	canonicalRequest := strings.Join([]string{
		"POST", "/", "", canonicalHeaders, signedHeaders, sha256hex(payload),
	}, "\n")

	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, vclmService)
	stringToSign := strings.Join([]string{
		algorithm, strconv.FormatInt(timestamp, 10), credentialScope,
		sha256hex([]byte(canonicalRequest)),
	}, "\n")

	secretDate := hmacSha256([]byte(date), []byte("TC3"+secretKey))
	secretService := hmacSha256([]byte(vclmService), secretDate)
	secretSigning := hmacSha256([]byte("tc3_request"), secretService)
	signature := hex.EncodeToString(hmacSha256([]byte(stringToSign), secretSigning))

	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, secretId, credentialScope, signedHeaders, signature)

	return map[string]string{
		"Authorization":  authorization,
		"Content-Type":   contentType,
		"Host":           vclmHost,
		"X-TC-Action":    action,
		"X-TC-Timestamp": strconv.FormatInt(timestamp, 10),
		"X-TC-Version":   vclmVersion,
		"X-TC-Region":    vclmRegion,
	}
}
