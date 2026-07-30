package seedancem

import (
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func baseURL(raw string) string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		base = defaultBaseURL
	}
	for _, suffix := range []string{createTaskPath, "/api/v3"} {
		if strings.HasSuffix(base, suffix) {
			base = strings.TrimSuffix(base, suffix)
			if suffix == "/api/v3" {
				base += "/api/v3"
			}
		}
	}
	return strings.TrimRight(base, "/")
}

func setBearerHeaders(req *http.Request, apiKey string, settings *dto.SeedanceMSettings) {
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if settings != nil && strings.TrimSpace(settings.ServiceVersion) != "" {
		req.Header.Set("service-version", strings.TrimSpace(settings.ServiceVersion))
	}
}

func base64PublicKey(publicKeyPEM string) string {
	return base64.StdEncoding.EncodeToString([]byte(strings.TrimSpace(publicKeyPEM)))
}

func contentHasText(items []ContentItem) bool {
	for _, item := range items {
		if item.Type == "text" || strings.TrimSpace(item.Text) != "" {
			return true
		}
	}
	return false
}

func payloadHasVideo(payload *requestPayload) bool {
	if payload == nil {
		return false
	}
	for _, item := range payload.Content {
		if item.Type == "video_url" || item.VideoURL != nil {
			return true
		}
	}
	return false
}

func hasVideoInput(req relaycommon.TaskSubmitReq) bool {
	if strings.TrimSpace(req.InputReference) != "" {
		return true
	}
	content, ok := req.Metadata["content"].([]interface{})
	if ok {
		for _, item := range content {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if m["type"] == "video_url" {
				return true
			}
			if _, has := m["video_url"]; has {
				return true
			}
		}
	}
	return hasNonEmptyList(req.Metadata["reference_video_urls"]) ||
		hasNonEmptyList(req.Metadata["referenceVideoUrls"]) ||
		hasNonEmptyList(req.Metadata["video_files"])
}

func hasNonEmptyList(value any) bool {
	switch v := value.(type) {
	case []string:
		return len(v) > 0
	case []any:
		return len(v) > 0
	default:
		return false
	}
}

func seedanceMBillingModel(modelName string) string {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case "", "doubao-seedance-2.0":
		return "doubao-seedance-2-0-260128"
	default:
		return modelName
	}
}

func isSupportedModel(modelName string) bool {
	if strings.TrimSpace(modelName) == "" {
		return true
	}
	for _, item := range ModelList {
		if strings.EqualFold(strings.TrimSpace(modelName), item) {
			return true
		}
	}
	return false
}

func requestDuration(req relaycommon.TaskSubmitReq) (int, bool) {
	if req.Duration != 0 {
		return req.Duration, true
	}
	if strings.TrimSpace(req.Seconds) != "" {
		if v, err := strconv.Atoi(strings.TrimSpace(req.Seconds)); err == nil {
			return v, true
		}
	}
	return intFromAny(req.Metadata["duration"])
}

func requestResolution(req relaycommon.TaskSubmitReq) string {
	resolution := strings.ToLower(strings.TrimSpace(asString(req.Metadata["resolution"])))
	if resolution == "" && strings.HasSuffix(strings.ToLower(strings.TrimSpace(req.Size)), "p") {
		resolution = strings.ToLower(strings.TrimSpace(req.Size))
	}
	return resolution
}

func intFromAny(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		maxInt := int64(^uint(0) >> 1)
		minInt := -maxInt - 1
		if v > maxInt || v < minInt {
			return 0, false
		}
		return int(v), true
	case float64:
		maxInt := int64(^uint(0) >> 1)
		minInt := -maxInt - 1
		if math.Trunc(v) != v || v > float64(maxInt) || v < float64(minInt) {
			return 0, false
		}
		return int(v), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(v))
		return i, err == nil
	default:
		return 0, false
	}
}

func int64FromAny(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		if math.Trunc(v) != v || v > float64(int64(^uint64(0)>>1)) || v < float64(-int64(^uint64(0)>>1)-1) {
			return 0, false
		}
		return int64(v), true
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
