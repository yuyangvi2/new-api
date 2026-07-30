package seedancem

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
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

func TestChannelNameIsPublicAlias(t *testing.T) {
	assert.Equal(t, "seedance-m", (&TaskAdaptor{}).GetChannelName())
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
