package seedancem

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
)

func TestConvertToRequestPayloadPreservesExplicitZeroValues(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Prompt: "test prompt",
		Model:  "doubao-seedance-2.0",
		Metadata: map[string]any{
			"generate_audio":          false,
			"watermark":               false,
			"camera_fixed":            false,
			"seed":                    float64(0),
			"execution_expires_after": float64(3600),
			"duration":                float64(4),
		},
	}

	payload, err := adaptor.convertToRequestPayload(req)
	require.NoError(t, err)

	data, err := common.Marshal(payload)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, common.Unmarshal(data, &out))

	assert.Equal(t, false, out["generate_audio"])
	assert.Equal(t, false, out["watermark"])
	assert.Equal(t, false, out["camera_fixed"])
	assert.Equal(t, float64(0), out["seed"])
	assert.Equal(t, float64(3600), out["execution_expires_after"])
}

func TestBuildRequestHeaderSetsVideoAndEncryptionHeaders(t *testing.T) {
	adaptor := &TaskAdaptor{
		apiKey:   "maas-key",
		hasVideo: true,
		settings: &dto.SeedanceMSettings{
			ServiceVersion:     "v1",
			EnableVideoEncrypt: true,
			PublicKeyPEM:       "public-key",
			PrivateKeyPEM:      "private-key",
		},
	}
	req, err := http.NewRequest(http.MethodPost, "https://example.test", bytes.NewReader(nil))
	require.NoError(t, err)

	err = adaptor.BuildRequestHeader(nil, req, nil)
	require.NoError(t, err)

	assert.Equal(t, "Bearer maas-key", req.Header.Get("Authorization"))
	assert.Equal(t, "v1", req.Header.Get("service-version"))
	assert.Equal(t, "true", req.Header.Get("Input-Has-Video"))
	assert.Equal(t, "true", req.Header.Get("Enable-TOS-Content-Result-Encryption"))
	assert.Equal(t, "RSA_OAEP_4096_AES_256", req.Header.Get("X-Encryption-Algorithm"))
	assert.NotEmpty(t, req.Header.Get("PK"))
}

func TestConvertToRequestPayloadUsesOfficialContentFields(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model: "doubao-seedance-2.0",
		Metadata: map[string]any{
			"content": []any{
				map[string]any{"type": "text", "text": "Create a product video"},
				map[string]any{
					"type":      "image_url",
					"image_url": map[string]any{"url": "https://example.com/input.png"},
					"role":      "reference_image",
				},
				map[string]any{
					"type":      "video_url",
					"video_url": map[string]any{"url": "https://example.com/input.mp4"},
					"role":      "reference_video",
				},
				map[string]any{
					"type":      "audio_url",
					"audio_url": map[string]any{"url": "https://example.com/input.mp3"},
					"role":      "reference_audio",
				},
			},
			"generate_audio":          false,
			"ratio":                   "16:9",
			"resolution":              "720p",
			"duration":                float64(10),
			"return_last_frame":       true,
			"service_tier":            "default",
			"execution_expires_after": float64(3600),
			"draft":                   false,
			"frames":                  float64(240),
			"seed":                    float64(42),
			"camera_fixed":            true,
			"watermark":               false,
		},
	}

	payload, err := adaptor.convertToRequestPayload(req)
	require.NoError(t, err)

	require.Len(t, payload.Content, 4)
	assert.Equal(t, "Create a product video", payload.Content[0].Text)
	assert.Equal(t, "https://example.com/input.png", payload.Content[1].ImageURL.URL)
	assert.Equal(t, "https://example.com/input.mp4", payload.Content[2].VideoURL.URL)
	assert.Equal(t, "https://example.com/input.mp3", payload.Content[3].AudioURL.URL)
	require.NotNil(t, payload.GenerateAudio)
	assert.False(t, *payload.GenerateAudio)
	assert.Equal(t, "16:9", payload.Ratio)
	assert.Equal(t, "720p", payload.Resolution)
	require.NotNil(t, payload.Duration)
	assert.Equal(t, 10, *payload.Duration)
	require.NotNil(t, payload.ReturnLastFrame)
	assert.True(t, *payload.ReturnLastFrame)
	assert.Equal(t, "default", payload.ServiceTier)
	require.NotNil(t, payload.ExecutionExpiresAfter)
	assert.Equal(t, 3600, *payload.ExecutionExpiresAfter)
	require.NotNil(t, payload.Draft)
	assert.False(t, *payload.Draft)
	require.NotNil(t, payload.Frames)
	assert.Equal(t, 240, *payload.Frames)
	require.NotNil(t, payload.Seed)
	assert.Equal(t, int64(42), *payload.Seed)
	require.NotNil(t, payload.CameraFixed)
	assert.True(t, *payload.CameraFixed)
	require.NotNil(t, payload.Watermark)
	assert.False(t, *payload.Watermark)
}

func TestConvertToRequestPayloadPreservesTopLevelNegativeOneDuration(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Prompt:   "test prompt",
		Model:    "doubao-seedance-2.0",
		Duration: -1,
	}

	payload, err := adaptor.convertToRequestPayload(req)
	require.NoError(t, err)
	require.NotNil(t, payload.Duration)
	assert.Equal(t, -1, *payload.Duration)
}

func TestChannelNameIsPublicAlias(t *testing.T) {
	assert.Equal(t, "seedance-m", (&TaskAdaptor{}).GetChannelName())
}

func TestSeedanceRAURLUsesGatewayHost(t *testing.T) {
	assert.Equal(t,
		"https://zhenze-huhehaote.cmecloud.cn/v1/security/token",
		seedanceRAURL("https://zhenze-huhehaote.cmecloud.cn/api/v3"),
	)
}

func TestValidateSeedanceMRequestBounds(t *testing.T) {
	tests := []struct {
		name string
		req  relaycommon.TaskSubmitReq
	}{
		{
			name: "duration too large",
			req: relaycommon.TaskSubmitReq{
				Prompt:   "prompt",
				Duration: 16,
			},
		},
		{
			name: "expiry too small",
			req: relaycommon.TaskSubmitReq{
				Prompt: "prompt",
				Metadata: map[string]any{
					"execution_expires_after": float64(3599),
				},
			},
		},
		{
			name: "seed too large",
			req: relaycommon.TaskSubmitReq{
				Prompt: "prompt",
				Metadata: map[string]any{
					"seed": float64(4294967296),
				},
			},
		},
		{
			name: "4k unsupported",
			req: relaycommon.TaskSubmitReq{
				Prompt: "prompt",
				Metadata: map[string]any{
					"resolution": "4k",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, validateSeedanceMRequest(tt.req))
		})
	}

	assert.NoError(t, validateSeedanceMRequest(relaycommon.TaskSubmitReq{
		Prompt:   "prompt",
		Duration: -1,
		Metadata: map[string]any{
			"execution_expires_after": float64(3600),
			"seed":                    float64(0),
		},
	}))
}

func TestValidateRequestAcceptsOfficialTopLevelContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := strings.NewReader(`{
		"model": "doubao-seedance-2.0",
		"content": [
			{"type": "text", "text": "生成一段城市夜景视频"}
		],
		"generate_audio": true,
		"ratio": "16:9",
		"resolution": "720p",
		"duration": 10,
		"draft": false,
		"watermark": false
	}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/video/generations", body)
	req.Header.Set("Content-Type", "application/json")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info)
	require.Nil(t, taskErr)

	storedReq, err := relaycommon.GetTaskRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, constant.TaskActionGenerate, info.Action)
	assert.Equal(t, "生成一段城市夜景视频", storedReq.Prompt)
	assert.Equal(t, true, storedReq.Metadata["generate_audio"])
	assert.Equal(t, "16:9", storedReq.Metadata["ratio"])
	assert.Equal(t, "720p", storedReq.Metadata["resolution"])
	assert.Equal(t, false, storedReq.Metadata["draft"])
	assert.Equal(t, false, storedReq.Metadata["watermark"])
	require.Contains(t, storedReq.Metadata, "content")
}

func TestEstimateBillingUsesVolcengineBasePriceForUserCharge(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	originalExchangeRate := operation_setting.USDExchangeRate
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
		operation_setting.USDExchangeRate = originalExchangeRate
	})
	common.QuotaPerUnit = 1_000_000
	operation_setting.USDExchangeRate = 1

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("task_request", relaycommon.TaskSubmitReq{
		Model:    "doubao-seedance-2.0",
		Duration: 5,
		Size:     "16:9",
		Metadata: map[string]any{"resolution": "720p"},
	})

	ratios := (&TaskAdaptor{}).EstimateBilling(ctx, &relaycommon.RelayInfo{
		OriginModelName: "doubao-seedance-2.0",
		ChannelMeta:     &relaycommon.ChannelMeta{},
		PriceData: types.PriceData{
			Quota:          4_968_000,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	})

	assert.Empty(t, ratios)
}

func TestParseTaskResultMapsStatusesAndLastFrame(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"id":"task-1",
		"status":"succeeded",
		"content":{"video_url":"https://cdn.example/video.mp4","last_frame_url":"https://cdn.example/frame.png"},
		"usage":{"completion_tokens":12,"total_tokens":34}
	}`)

	info, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)

	assert.Equal(t, "SUCCESS", string(info.Status))
	assert.Equal(t, "100%", info.Progress)
	assert.Equal(t, "https://cdn.example/video.mp4", info.Url)
	assert.Equal(t, 12, info.CompletionTokens)
	assert.Equal(t, 34, info.TotalTokens)

	failed, err := adaptor.ParseTaskResult([]byte(`{"id":"task-2","status":"expired","error":{"message":"expired"}}`))
	require.NoError(t, err)
	assert.Equal(t, "FAILURE", string(failed.Status))
	assert.Equal(t, "expired", failed.Reason)
}
