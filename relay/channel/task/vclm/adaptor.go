// Package vclm 实现腾讯云 VCLM（视频创作大模型）可灵图生视频的任务适配器。
// 上游：vclm.tencentcloudapi.com，TC3-HMAC-SHA256 签名，异步 submit/query。
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
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
)

const (
	vclmHost     = "vclm.tencentcloudapi.com"
	vclmService  = "vclm"
	vclmVersion  = "2024-05-23"
	vclmRegion   = "ap-guangzhou"
	actionSubmit = "SubmitImageToVideoJob"
	actionQuery  = "DescribeImageToVideoJob"

	ctxPayloadKey = "vclm_payload"
)

// ============================ 请求 / 响应结构 ============================

type tcImage struct {
	Url    string `json:"Url,omitempty"`
	Base64 string `json:"Base64,omitempty"`
}

// submitPayload 对应 SubmitImageToVideoJob 入参（常用字段；高级字段经 metadata 透传）
type submitPayload struct {
	Model          string   `json:"Model,omitempty"`
	Image          *tcImage `json:"Image,omitempty"`
	ImageTail      *tcImage `json:"ImageTail,omitempty"`
	Prompt         string   `json:"Prompt,omitempty"`
	NegativePrompt string   `json:"NegativePrompt,omitempty"`
	Duration       string   `json:"Duration,omitempty"`
	Mode           string   `json:"Mode,omitempty"`
	CfgScale       *float64 `json:"CfgScale,omitempty"`
	Sound          string   `json:"Sound,omitempty"`
	CallbackUrl    string   `json:"CallbackUrl,omitempty"`
	ExternalTaskId string   `json:"ExternalTaskId,omitempty"`
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

// new-api 模型名 → VCLM Model 版本码
var modelMap = map[string]string{
	"kling-v1":          "v1.0",
	"kling-v1-5":        "v1.5",
	"kling-v1-6":        "v1.6",
	"kling-v2-master":   "v2.0",
	"kling-v2-1":        "v2.1",
	"kling-v2-1-master": "v2.1m",
	"kling-v2-5-turbo":  "v2.5",
	"kling-v2-6":        "v2.6",
	"kling-v3":          "v3.0",
}

// ============================ 适配器实现 ============================

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

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
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

	p := submitPayload{
		Prompt:   req.Prompt,
		Mode:     taskcommon.DefaultString(req.Mode, "std"),
		Duration: fmt.Sprintf("%d", taskcommon.DefaultInt(req.Duration, 5)),
		Model:    mapModel(info.UpstreamModelName),
	}
	if req.Image != "" {
		p.Image = &tcImage{Url: req.Image}
	}
	// 透传高级字段（NegativePrompt/ImageTail/CfgScale/Sound/CameraControl 等）
	if err := taskcommon.UnmarshalMetadata(req.Metadata, &p); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}

	data, err := common.Marshal(p)
	if err != nil {
		return nil, err
	}
	c.Set(ctxPayloadKey, data) // 供 BuildRequestHeader 做 TC3 签名
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	payload, _ := c.Get(ctxPayloadKey)
	body, _ := payload.([]byte)
	secretId, secretKey, err := splitKey(a.apiKey)
	if err != nil {
		return err
	}
	for k, v := range signTC3(secretId, secretKey, actionSubmit, body, time.Now().Unix()) {
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
	for k, v := range signTC3(secretId, secretKey, actionQuery, payload, time.Now().Unix()) {
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
	info := &relaycommon.TaskInfo{TaskID: ""}
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
	return []string{
		"kling-v1", "kling-v1-5", "kling-v1-6",
		"kling-v2-master", "kling-v2-1", "kling-v2-1-master",
		"kling-v2-5-turbo", "kling-v2-6", "kling-v3",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "vclm"
}

// ============================ 辅助 ============================

func mapModel(name string) string {
	if v, ok := modelMap[name]; ok {
		return v
	}
	return name // 已是 VCLM 版本码（如 v1.6）则直接用
}

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

// signTC3 生成腾讯云 TC3-HMAC-SHA256 鉴权头。
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
