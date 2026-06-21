// Package aiart 实现腾讯云 AIART（大模型图像创作引擎）任务适配器（派发式）。
//
// 上游：aiart.tencentcloudapi.com，TC3-HMAC-SHA256 签名，异步 submit/query。
// 一个渠道类型(9002) 内按模型名自动派发到对应 AIART Action：
//   - image-gi      : SubmitContentToImageGIJob / DescribeContentToImageGIJob (Nano Banana Pro)
//   - image-gi2     : SubmitContentToImageGIJob / DescribeContentToImageGIJob (Nano Banana 2)
//   - hunyuan-image : SubmitTextToImageJob / QueryTextToImageJob (混元生图 3.0)
//
// apiKey 格式：SecretId|SecretKey
package aiart

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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
)

const (
	aiartHost    = "aiart.tencentcloudapi.com"
	aiartService = "aiart"
	aiartVersion = "2022-12-29"
	aiartRegion  = "ap-guangzhou"

	ctxPayloadKey = "aiart_payload"
	ctxActionKey  = "aiart_action"
)

// modelDef 定义一个对外模型名的 AIART 参数映射。
type modelDef struct {
	aiartModel  string // 上游 Model 参数值；为空表示该 Action 不接受 Model
	submit      string // 提交 Action
	query       string // 查询 Action
	needModel   bool   // 请求体是否需要 Model 字段
}

var modelDefs = map[string]modelDef{
	"image-gi": {
		aiartModel: "Image-GI",
		submit:     "SubmitContentToImageGIJob",
		query:      "DescribeContentToImageGIJob",
		needModel:  true,
	},
	"image-gi2": {
		aiartModel: "Image-GI2",
		submit:     "SubmitContentToImageGIJob",
		query:      "DescribeContentToImageGIJob",
		needModel:  true,
	},
	"hunyuan-image": {
		aiartModel: "",
		submit:     "SubmitTextToImageJob",
		query:      "QueryTextToImageJob",
		needModel:  false,
	},
}

// submitToQuery 从 modelDefs 构造 submitAction → queryAction 反查表。
var submitToQuery = func() map[string]string {
	m := make(map[string]string, len(modelDefs))
	for _, def := range modelDefs {
		m[def.submit] = def.query
	}
	return m
}()

// ============================ 响应结构 ============================

type tcError struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

type tcImage struct {
	Url    string `json:"Url,omitempty"`
	Base64 string `json:"Base64,omitempty"`
}

type submitResponse struct {
	Response struct {
		JobId     string   `json:"JobId"`
		RequestId string   `json:"RequestId"`
		Error     *tcError `json:"Error,omitempty"`
	} `json:"Response"`
}

type queryResponse struct {
	Response struct {
		JobStatusCode string   `json:"JobStatusCode"`
		JobStatusMsg  string   `json:"JobStatusMsg"`
		JobErrorCode  string   `json:"JobErrorCode"`
		JobErrorMsg   string   `json:"JobErrorMsg"`
		ResultImage   []string `json:"ResultImage"`
		RevisedPrompt any      `json:"RevisedPrompt"`
		RequestId     string   `json:"RequestId"`
		Error         *tcError `json:"Error,omitempty"`
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
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("unsupported model: %s (supported: %s)", peek.Model, strings.Join(supported, ", ")),
			"invalid_model", http.StatusBadRequest)
	}
	c.Set(ctxActionKey, def.submit)
	return relaycommon.ValidateBasicTaskRequest(c, info, def.submit)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return "https://" + aiartHost + "/", nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req := v.(relaycommon.TaskSubmitReq)

	body := buildRequest(&req, info.UpstreamModelName)

	// metadata 透传/覆盖（PascalCase 键，供特有参数如 AspectRatio, Resolution 等）
	if err := taskcommon.UnmarshalMetadata(req.Metadata, &body); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}

	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	c.Set(ctxPayloadKey, data)
	return bytes.NewReader(data), nil
}

// buildRequest 构造 AIART 请求体，按模型族区分字段。
func buildRequest(req *relaycommon.TaskSubmitReq, upstreamModel string) map[string]any {
	m := map[string]any{}
	def := modelDefs[strings.ToLower(upstreamModel)]

	// Prompt (all models)
	if req.Prompt != "" {
		m["Prompt"] = req.Prompt
	}

	// Model field (only for Image-GI)
	if def.needModel && def.aiartModel != "" {
		m["Model"] = def.aiartModel
	}

	// Images / reference images
	if def.needModel {
		// Image-GI: Images as array of {Url, Base64} objects
		var images []tcImage
		if req.Image != "" {
			images = append(images, buildTcImage(req.Image))
		}
		for _, img := range req.Images {
			images = append(images, buildTcImage(img))
		}
		if len(images) > 0 {
			m["Images"] = images
		}
	} else {
		// 混元生图: Images as array of strings (URL or base64)
		var images []string
		if req.Image != "" {
			images = append(images, req.Image)
		}
		images = append(images, req.Images...)
		if len(images) > 0 {
			m["Images"] = images
		}
	}

	// Size → Resolution or AspectRatio depending on model
	if req.Size != "" {
		if def.needModel {
			// Image-GI uses AspectRatio (e.g. "1:1")
			m["AspectRatio"] = sizeToAspectRatio(req.Size)
		} else {
			// 混元生图 uses Resolution (e.g. "1024:1024")
			m["Resolution"] = sizeToResolution(req.Size)
		}
	}

	// LogoAdd: default 0 (no watermark) for API usage
	m["LogoAdd"] = 0

	return m
}

// buildTcImage creates an AIART Image object from a URL or base64 string.
func buildTcImage(s string) tcImage {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return tcImage{Url: s}
	}
	return tcImage{Base64: s}
}

// sizeToAspectRatio converts common size formats to AIART aspect ratio strings.
func sizeToAspectRatio(size string) string {
	// If already in ratio format (e.g. "1:1", "16:9"), pass through
	if strings.Contains(size, ":") {
		return size
	}
	// Try to parse WxH format
	parts := strings.SplitN(strings.ToLower(size), "x", 2)
	if len(parts) == 2 {
		w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
		h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
		if errW == nil && errH == nil && w > 0 && h > 0 {
			g := gcd(w, h)
			return fmt.Sprintf("%d:%d", w/g, h/g)
		}
	}
	return size // fallback: pass as-is
}

// sizeToResolution converts size formats to 混元生图 Resolution (e.g. "1024:1024").
func sizeToResolution(size string) string {
	// Already in colon format (e.g. "1024:1024")
	if strings.Contains(size, ":") {
		return size
	}
	// Convert WxH to W:H
	if strings.Contains(strings.ToLower(size), "x") {
		return strings.Replace(strings.ToLower(size), "x", ":", 1)
	}
	return size
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	payload, _ := c.Get(ctxPayloadKey)
	body, _ := payload.([]byte)
	action, _ := c.Get(ctxActionKey)
	actionStr, _ := action.(string)
	if actionStr == "" {
		return fmt.Errorf("missing action in context")
	}
	secretId, secretKey, err := splitKey(a.apiKey)
	if err != nil {
		return err
	}
	for k, v := range signTC3(secretId, secretKey, actionStr, body, time.Now().Unix()) {
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
	// 从提交 Action 反查对应的查询 Action
	submitAction, _ := body["action"].(string)
	qAction, ok := submitToQuery[submitAction]
	if !ok || qAction == "" {
		qAction = "DescribeContentToImageGIJob" // 兜底
	}
	secretId, secretKey, err := splitKey(key)
	if err != nil {
		return nil, err
	}
	payload, err := common.Marshal(map[string]string{"JobId": taskID})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://"+aiartHost+"/", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	for k, v := range signTC3(secretId, secretKey, qAction, payload, time.Now().Unix()) {
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
	code := r.Response.JobStatusCode
	switch {
	case code == "INIT" || code == "WAIT" || code == "1":
		info.Status = model.TaskStatusSubmitted
	case code == "RUN" || code == "2" || code == "3" || code == "4":
		info.Status = model.TaskStatusInProgress
	case code == "DONE" || code == "5":
		info.Status = model.TaskStatusSuccess
		if len(r.Response.ResultImage) > 0 {
			info.Url = r.Response.ResultImage[0]
		}
		// 每次成功生成按 1 个计费单元
		info.CompletionTokens = 1
		info.TotalTokens = 1
	case code == "FAIL" || code == "-1":
		info.Status = model.TaskStatusFailure
		info.Reason = strings.TrimSpace(r.Response.JobErrorCode + " " + r.Response.JobErrorMsg)
		if info.Reason == "" {
			info.Reason = r.Response.JobStatusMsg
		}
	default:
		return nil, fmt.Errorf("unknown task status: %s", code)
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
	return "aiart"
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
		contentType, aiartHost, strings.ToLower(action))
	signedHeaders := "content-type;host;x-tc-action"
	canonicalRequest := strings.Join([]string{
		"POST", "/", "", canonicalHeaders, signedHeaders, sha256hex(payload),
	}, "\n")

	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, aiartService)
	stringToSign := strings.Join([]string{
		algorithm, strconv.FormatInt(timestamp, 10), credentialScope,
		sha256hex([]byte(canonicalRequest)),
	}, "\n")

	secretDate := hmacSha256([]byte(date), []byte("TC3"+secretKey))
	secretService := hmacSha256([]byte(aiartService), secretDate)
	secretSigning := hmacSha256([]byte("tc3_request"), secretService)
	signature := hex.EncodeToString(hmacSha256([]byte(stringToSign), secretSigning))

	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, secretId, credentialScope, signedHeaders, signature)

	return map[string]string{
		"Authorization":  authorization,
		"Content-Type":   contentType,
		"Host":           aiartHost,
		"X-TC-Action":    action,
		"X-TC-Timestamp": strconv.FormatInt(timestamp, 10),
		"X-TC-Version":   aiartVersion,
		"X-TC-Region":    aiartRegion,
	}
}
