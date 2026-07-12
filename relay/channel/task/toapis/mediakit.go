package toapis

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
)

const (
	mediaKitBaseURL          = "https://mediakit.cn-beijing.volces.com"
	mediaKitEnhanceVideoPath = "/api/v1/tools/enhance-video"
	mediaKitTaskPathPrefix   = "/api/v1/tasks/"
	mediaKitStatusCompleted  = "completed"
	mediaKitStatusFailed     = "failed"
	videoSR480p              = "480p"
	videoSR720p              = "720p"
	videoSR1080p             = "1080p"
)

type mediaKitSubmitResponse struct {
	Success   bool   `json:"success"`
	TaskID    string `json:"task_id"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

type mediaKitTaskResponse struct {
	Success bool   `json:"success"`
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  struct {
		Duration    float64 `json:"duration"`
		FPS         float64 `json:"fps"`
		Resolution  string  `json:"resolution"`
		ToolVersion string  `json:"tool_version"`
		VideoURL    string  `json:"video_url"`
	} `json:"result"`
}

func applyVideoSuperResolutionRewrite(req *relaycommon.TaskSubmitReq, payload map[string]any, enabled bool) *relaycommon.TaskVideoSuperResolution {
	if !enabled || req == nil || payload == nil {
		return nil
	}
	videoSR := requestedVideoSuperResolution(req, payload)
	if videoSR == nil {
		return nil
	}
	sourceResolution := sourceResolutionForVideoSR(videoSR)
	if sourceResolution == "" {
		return nil
	}
	videoSR.SourceResolution = sourceResolution
	payload["resolution"] = sourceResolution
	delete(payload, "resolution_limit")
	return videoSR
}

func requestedVideoSuperResolution(req *relaycommon.TaskSubmitReq, payload map[string]any) *relaycommon.TaskVideoSuperResolution {
	if limit, ok := positiveInt(req.Metadata["resolution_limit"]); ok && limit > 480 {
		return &relaycommon.TaskVideoSuperResolution{ResolutionLimit: clampResolutionLimit(limit)}
	}
	if sr := videoSRFromResolutionString(asString(payload["resolution"])); sr != nil {
		return sr
	}
	if sr := videoSRFromResolutionString(req.Size); sr != nil {
		return sr
	}
	return nil
}

func videoSRFromResolutionString(value string) *relaycommon.TaskVideoSuperResolution {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	if normalized == "" || strings.Contains(normalized, ":") {
		return nil
	}
	switch normalized {
	case "4k", "uhd":
		return &relaycommon.TaskVideoSuperResolution{TargetResolution: "4k"}
	case "2k":
		return &relaycommon.TaskVideoSuperResolution{TargetResolution: "2k"}
	}
	if strings.HasSuffix(normalized, "p") {
		n, err := strconv.Atoi(strings.TrimSuffix(normalized, "p"))
		if err != nil || n <= 480 {
			return nil
		}
		if n >= 2160 {
			return &relaycommon.TaskVideoSuperResolution{TargetResolution: "4k"}
		}
		return &relaycommon.TaskVideoSuperResolution{ResolutionLimit: clampResolutionLimit(n)}
	}
	if width, height, ok := parseResolutionDimensions(normalized); ok {
		shortSide := width
		if height < shortSide {
			shortSide = height
		}
		longSide := width
		if height > longSide {
			longSide = height
		}
		if shortSide > 2160 || longSide >= 3840 {
			return &relaycommon.TaskVideoSuperResolution{TargetResolution: "4k"}
		}
		if shortSide > 480 {
			return &relaycommon.TaskVideoSuperResolution{ResolutionLimit: clampResolutionLimit(shortSide)}
		}
		if longSide > 1920 {
			return &relaycommon.TaskVideoSuperResolution{TargetResolution: "2k"}
		}
	}
	return nil
}

func sourceResolutionForVideoSR(videoSR *relaycommon.TaskVideoSuperResolution) string {
	if videoSR == nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(videoSR.TargetResolution)) {
	case "4k", "uhd", "2k":
		return videoSR1080p
	}
	switch {
	case videoSR.ResolutionLimit > 1080:
		return videoSR1080p
	case videoSR.ResolutionLimit > 720:
		return videoSR720p
	case videoSR.ResolutionLimit > 480:
		return videoSR480p
	default:
		return ""
	}
}

func parseResolutionDimensions(value string) (int, int, bool) {
	parts := strings.Split(value, "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, errW := strconv.Atoi(parts[0])
	height, errH := strconv.Atoi(parts[1])
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func positiveInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, v > 0
	case int64:
		if v > int64(^uint(0)>>1) {
			return int(^uint(0) >> 1), true
		}
		return int(v), v > 0
	case float64:
		if v <= 0 {
			return 0, false
		}
		if v > float64(^uint(0)>>1) {
			return int(^uint(0) >> 1), true
		}
		return int(v), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		return parsed, err == nil && parsed > 0
	default:
		return 0, false
	}
}

func clampResolutionLimit(value int) int {
	if value < 64 {
		return 64
	}
	if value > 2160 {
		return 2160
	}
	return value
}

func (a *TaskAdaptor) PostProcessTaskResult(ctx context.Context, ch *model.Channel, task *model.Task, taskResult *relaycommon.TaskInfo) (*relaycommon.TaskInfo, error) {
	if task == nil || taskResult == nil || taskResult.Status != model.TaskStatusSuccess {
		return taskResult, nil
	}
	videoSR := task.PrivateData.VideoSuperResolution
	if videoSR == nil {
		if taskResult.Url != "" {
			taskResult.Url = maybeTransferToTOS(taskResult.Url)
		}
		return taskResult, nil
	}
	apiKey := strings.TrimSpace(common.VideoSuperResolutionMediaKitKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("AI_MEDIAKIT_API_KEY"))
	}
	if apiKey == "" {
		return relaycommon.FailTaskInfo("AI MediaKit API key is not configured"), nil
	}
	proxy := ""
	if ch != nil {
		proxy = ch.GetSetting().Proxy
	}
	if strings.TrimSpace(videoSR.MediaKitTaskID) == "" {
		sourceURL := strings.TrimSpace(taskResult.Url)
		if sourceURL == "" || strings.HasPrefix(sourceURL, "data:") || !taskcommon.IsHTTPURL(sourceURL) {
			return relaycommon.FailTaskInfo("AI MediaKit requires an HTTP video URL for super resolution"), nil
		}
		taskID, err := submitMediaKitEnhance(ctx, apiKey, sourceURL, videoSR, proxy)
		if err != nil {
			return relaycommon.FailTaskInfo("AI MediaKit submit failed: " + err.Error()), nil
		}
		videoSR.MediaKitTaskID = taskID
		videoSR.Status = "submitted"
		videoSR.SubmittedAt = time.Now().Unix()
		logger.LogInfo(ctx, fmt.Sprintf("Task %s submitted AI MediaKit super resolution task %s", task.TaskID, taskID))
		return &relaycommon.TaskInfo{
			TaskID:   taskResult.TaskID,
			Status:   model.TaskStatusInProgress,
			Progress: "95%",
		}, nil
	}

	mediaTask, err := fetchMediaKitTask(ctx, apiKey, videoSR.MediaKitTaskID, proxy)
	if err != nil {
		return nil, err
	}
	status := strings.ToLower(strings.TrimSpace(mediaTask.Status))
	videoSR.Status = status
	switch status {
	case mediaKitStatusCompleted:
		outputURL := strings.TrimSpace(mediaTask.Result.VideoURL)
		if outputURL == "" {
			return relaycommon.FailTaskInfo("AI MediaKit completed without video_url"), nil
		}
		videoSR.FinishedAt = time.Now().Unix()
		videoSR.ResultURL = maybeTransferToTOS(outputURL)
		taskResult.Url = videoSR.ResultURL
		return taskResult, nil
	case mediaKitStatusFailed, "failure", "error", "cancelled", "canceled":
		videoSR.Error = firstNonEmpty(mediaTask.Message, "AI MediaKit super resolution failed")
		videoSR.FinishedAt = time.Now().Unix()
		return relaycommon.FailTaskInfo(videoSR.Error), nil
	default:
		return &relaycommon.TaskInfo{
			TaskID:   taskResult.TaskID,
			Status:   model.TaskStatusInProgress,
			Progress: "95%",
		}, nil
	}
}

func submitMediaKitEnhance(ctx context.Context, apiKey string, sourceURL string, videoSR *relaycommon.TaskVideoSuperResolution, proxy string) (string, error) {
	payload := map[string]any{
		"video_url":    sourceURL,
		"scene":        "aigc",
		"tool_version": "standard",
	}
	if videoSR.ResolutionLimit > 0 {
		payload["resolution_limit"] = videoSR.ResolutionLimit
	} else if videoSR.TargetResolution != "" {
		payload["resolution"] = videoSR.TargetResolution
	}
	body, err := common.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mediaKitBaseURL+mediaKitEnhanceVideoPath, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	setMediaKitHeaders(req, apiKey)
	resp, err := doMediaKitRequest(req, proxy)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var data mediaKitSubmitResponse
	if err = common.Unmarshal(respBody, &data); err != nil {
		return "", err
	}
	if !data.Success {
		return "", fmt.Errorf("%s", firstNonEmpty(data.Message, "submit returned success=false"))
	}
	if strings.TrimSpace(data.TaskID) == "" {
		return "", fmt.Errorf("submit response missing task_id")
	}
	return strings.TrimSpace(data.TaskID), nil
}

func fetchMediaKitTask(ctx context.Context, apiKey string, taskID string, proxy string) (*mediaKitTaskResponse, error) {
	uri := mediaKitBaseURL + mediaKitTaskPathPrefix + url.PathEscape(strings.TrimSpace(taskID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	setMediaKitHeaders(req, apiKey)
	resp, err := doMediaKitRequest(req, proxy)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("AI MediaKit status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var data mediaKitTaskResponse
	if err = common.Unmarshal(respBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func setMediaKitHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

func doMediaKitRequest(req *http.Request, proxy string) (*http.Response, error) {
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}
