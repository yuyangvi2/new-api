package vodaigc

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}

func TestBuildRequestBodyMapsSupportedGeminiModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		model   string
		version string
	}{
		{model: "gemini-2.5-flash-image", version: "2.5"},
		{model: "gemini-3-pro-image", version: "3.0"},
		{model: "gemini-3.1-flash-image", version: "3.1"},
		{model: "gemini-3.1-flash-lite-image", version: "3.1-lite"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			body := `{"model":"` + tt.model + `","prompt":"draw a cat","images":["https://example.com/ref.png"],"aspect_ratio":"16:9","resolution":"2K"}`
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/tasks", strings.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Set("model_mapping", `{"`+tt.model+`":"GG:`+tt.version+`"}`)

			info := &relaycommon.RelayInfo{
				OriginModelName: tt.model,
				TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
				ChannelMeta: &relaycommon.ChannelMeta{
					UpstreamModelName: tt.version,
				},
			}
			adaptor := &TaskAdaptor{}
			require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

			reader, err := adaptor.BuildRequestBody(ctx, info)
			require.NoError(t, err)
			payload, err := io.ReadAll(reader)
			require.NoError(t, err)

			var request createRequest
			require.NoError(t, common.Unmarshal(payload, &request))
			assert.Equal(t, defaultSubAppID, request.SubAppID)
			assert.Equal(t, "GG", request.ModelName)
			assert.Equal(t, tt.version, request.ModelVersion)
			assert.Equal(t, "Mainland", request.InputRegion)
			assert.Equal(t, "Temporary", request.OutputConfig.StorageMode)
			assert.Equal(t, "Disabled", request.OutputConfig.LogoAdd)
			assert.Equal(t, "16:9", request.OutputConfig.AspectRatio)
			assert.Equal(t, "2K", request.OutputConfig.Resolution)
			assert.Equal(t, []inputFileInfo{{Type: "Url", URL: "https://example.com/ref.png"}}, request.FileInfos)

			var raw map[string]any
			require.NoError(t, common.Unmarshal(payload, &raw))
			assert.NotContains(t, raw, "SceneType")
			assert.NotContains(t, raw, "OutputImageCount")
		})
	}
}

func TestValidateRequestRejectsUnsupportedMappedModelFamily(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/tasks", strings.NewReader(`{"model":"gemini-3-pro-image","prompt":"cat"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("model_mapping", `{"gemini-3-pro-image":"Unknown:3.0"}`)

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}})
	require.NotNil(t, taskErr)
	assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	assert.Equal(t, "invalid_model_mapping", taskErr.Code)
}

func TestBuildRequestBodySupportsConfiguredViduModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/tasks", strings.NewReader(`{"model":"vidu-q2","prompt":"draw a cat","images":["https://example.com/ref.png"]}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("model_mapping", `{"vidu-q2":"Vidu:q2"}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "vidu-q2",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "secret-id|secret-key",
			UpstreamModelName: "Vidu:q2",
			ChannelOtherSettings: dto.ChannelOtherSettings{VODAIGC: &dto.VODAIGCSettings{
				SubAppID:         123456,
				InputRegion:      "Oversea",
				DefaultModelName: "GG",
			}},
		},
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	reader, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	payload, err := io.ReadAll(reader)
	require.NoError(t, err)
	var request createRequest
	require.NoError(t, common.Unmarshal(payload, &request))
	assert.Equal(t, int64(123456), request.SubAppID)
	assert.Equal(t, "Vidu", request.ModelName)
	assert.Equal(t, "q2", request.ModelVersion)
	assert.Equal(t, "Oversea", request.InputRegion)
	assert.Equal(t, "1080p", request.OutputConfig.Resolution)
	assert.Equal(t, int64(123456), info.TaskRelayInfo.SubAppID)
}

func TestValidateRequestAppliesViduReferenceImageLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	images := strings.Repeat(`"https://example.com/ref.png",`, 7) + `"https://example.com/ref.png"`
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/tasks", strings.NewReader(`{"model":"vidu-q2","images":[`+images+`]}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("model_mapping", `{"vidu-q2":"Vidu:q2"}`)

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}})
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_image_count", taskErr.Code)
}

func TestUnknownModelProfileAcceptsPromptOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/tasks", strings.NewReader(`{"model":"future-model","prompt":"cat"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("model_mapping", `{"future-model":"GG:future"}`)

	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "GG:future",
		},
	}
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	reader, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	payload, err := io.ReadAll(reader)
	require.NoError(t, err)
	var request createRequest
	require.NoError(t, common.Unmarshal(payload, &request))
	assert.Empty(t, request.FileInfos)
	assert.Empty(t, request.OutputConfig.AspectRatio)
	assert.Empty(t, request.OutputConfig.Resolution)
}

func TestUnknownModelProfileRejectsUnverifiedCapabilities(t *testing.T) {
	tests := []struct {
		name string
		body string
		code string
	}{
		{name: "reference image", body: `{"model":"future-model","prompt":"cat","images":["https://example.com/ref.png"]}`, code: "invalid_image_count"},
		{name: "aspect ratio", body: `{"model":"future-model","prompt":"cat","aspect_ratio":"1:1"}`, code: "invalid_output_config"},
		{name: "resolution", body: `{"model":"future-model","prompt":"cat","resolution":"1K"}`, code: "invalid_output_config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/tasks", strings.NewReader(tt.body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Set("model_mapping", `{"future-model":"GG:future"}`)

			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}})
			require.NotNil(t, taskErr)
			assert.Equal(t, tt.code, taskErr.Code)
		})
	}
}

func TestValidateRequestAllowsReferenceImageWithoutPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/tasks", strings.NewReader(`{"model":"gemini-2.5-flash-image","images":["https://example.com/ref.png"]}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("model_mapping", `{"gemini-2.5-flash-image":"2.5"}`)
	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}

	require.Nil(t, (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, info))
	req, err := relaycommon.GetTaskRequest(ctx)
	require.NoError(t, err)
	assert.Empty(t, req.Prompt)
	assert.Equal(t, []string{"https://example.com/ref.png"}, req.Images)
}

func TestBuildInputFileInfosSupportsTencentBase64Input(t *testing.T) {
	const pngBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB"

	fileInfos, err := buildInputFileInfos([]string{
		"data:image/png;base64," + pngBase64,
		"https://example.com/reference.webp",
	})

	require.NoError(t, err)
	assert.Equal(t, []inputFileInfo{
		{Type: "Base64", Base64: pngBase64},
		{Type: "Url", URL: "https://example.com/reference.webp"},
	}, fileInfos)
}

func TestBuildInputFileInfosRejectsUnsupportedImageFormats(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
	}{
		{name: "GIF", content: []byte("GIF89a")},
		{name: "BMP", content: []byte("BM")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildInputFileInfos([]string{
				base64.StdEncoding.EncodeToString(tt.content),
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported reference image type")
		})
	}
}

func TestValidateRequestRejectsInvalidReferenceImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/tasks", strings.NewReader(`{"model":"gemini-2.5-flash-image","prompt":"cat","images":["not-base64"]}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("model_mapping", `{"gemini-2.5-flash-image":"2.5"}`)

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(ctx, &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}})
	require.NotNil(t, taskErr)
	assert.Equal(t, "invalid_reference_image", taskErr.Code)
}

func TestValidateOutputConfigUsesTencentResolutionValues(t *testing.T) {
	assert.NoError(t, validateOutputConfig(upstreamModel{Name: "GG", Version: "3.1"}, relaycommon.TaskSubmitReq{Resolution: "720p"}))
	assert.NoError(t, validateOutputConfig(upstreamModel{Name: "GG", Version: "3.1-lite"}, relaycommon.TaskSubmitReq{Resolution: "720P"}))
	assert.Error(t, validateOutputConfig(upstreamModel{Name: "GG", Version: "3.1"}, relaycommon.TaskSubmitReq{Resolution: "512"}))
	assert.Error(t, validateOutputConfig(upstreamModel{Name: "GG", Version: "3.0"}, relaycommon.TaskSubmitReq{Resolution: "720P"}))
	assert.NoError(t, validateOutputConfig(upstreamModel{Name: "Vidu", Version: "q2"}, relaycommon.TaskSubmitReq{Resolution: "1080P"}))
	assert.Error(t, validateOutputConfig(upstreamModel{Name: "Vidu", Version: "q2"}, relaycommon.TaskSubmitReq{Resolution: "720P"}))
}

func TestSignTC3UsesVODServiceAndVersion(t *testing.T) {
	headers := signTC3("AKIDEXAMPLE", "secret", createAction, vodHost, "/", []byte(`{"SubAppId":1480226150}`), 1700000000)

	assert.Equal(t, createAction, headers["X-TC-Action"])
	assert.Equal(t, vodVersion, headers["X-TC-Version"])
	assert.Equal(t, vodHost, headers["Host"])
	assert.NotContains(t, headers, "X-TC-Region")
	assert.Equal(t,
		"TC3-HMAC-SHA256 Credential=AKIDEXAMPLE/2023-11-14/vod/tc3_request, SignedHeaders=content-type;host;x-tc-action, Signature=6c18133f9e4554a9878d09b9752d32c7784cb145da956cf7df55d36aa0e0218c",
		headers["Authorization"],
	)
}

func TestFetchTaskUsesConfiguredBaseURLAndSubAppID(t *testing.T) {
	service.InitHttpClient()
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, common.DecodeJson(r.Body, &received))
		assert.Equal(t, describeAction, r.Header.Get("X-TC-Action"))
		assert.NotEmpty(t, r.Host)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":{"Status":"WAITING"}}`))
	}))
	defer server.Close()

	resp, err := (&TaskAdaptor{}).FetchTask(server.URL, "secret-id|secret-key", map[string]any{
		"task_id":    "upstream-task",
		"sub_app_id": int64(123456),
	}, "")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, float64(123456), received["SubAppId"])
	assert.Equal(t, "upstream-task", received["TaskId"])
}

func TestDoResponseReturnsPublicImageTaskID(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	bodyReader := &trackingReadCloser{Reader: strings.NewReader(`{"Response":{"TaskId":"upstream-task","RequestId":"request-id"}}`)}
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       bodyReader,
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "gemini-3-pro-image",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task-public",
		},
	}

	taskID, _, taskErr := (&TaskAdaptor{}).DoResponse(ctx, response, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "upstream-task", taskID)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var body dto.ImageTaskSubmitResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "task-public", body.TaskID)
	assert.Equal(t, "image_generation_task", body.Object)
	assert.True(t, bodyReader.closed)
}

func TestDoResponseClassifiesTencentErrors(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		statusCode int
		localError bool
	}{
		{name: "invalid parameter", code: "InvalidParameterValue", statusCode: http.StatusBadRequest, localError: true},
		{name: "authentication", code: "AuthFailure.SignatureFailure", statusCode: http.StatusUnauthorized},
		{name: "rate limit", code: "RequestLimitExceeded", statusCode: http.StatusTooManyRequests},
		{name: "resource shortage", code: "ResourceInsufficient", statusCode: http.StatusServiceUnavailable},
		{name: "unknown upstream failure", code: "InternalError", statusCode: http.StatusBadGateway},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			response := &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"Response":{"Error":{"Code":"` + tt.code + `","Message":"upstream error"}}}`,
				)),
			}

			_, _, taskErr := (&TaskAdaptor{}).DoResponse(ctx, response, &relaycommon.RelayInfo{})
			require.NotNil(t, taskErr)
			assert.Equal(t, tt.statusCode, taskErr.StatusCode)
			assert.Equal(t, tt.localError, taskErr.LocalError)
		})
	}
}

func TestDoResponseTreatsEmptyTaskIDAsUpstreamFailure(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"Response":{"RequestId":"request-id"}}`)),
	}

	_, _, taskErr := (&TaskAdaptor{}).DoResponse(ctx, response, &relaycommon.RelayInfo{})
	require.NotNil(t, taskErr)
	assert.Equal(t, http.StatusBadGateway, taskErr.StatusCode)
	assert.False(t, taskErr.LocalError)
}

func TestParseTaskResult(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		status     model.TaskStatus
		progress   string
		url        string
		reasonPart string
		wantErr    bool
	}{
		{
			name:     "waiting",
			body:     `{"Response":{"Status":"WAITING","AigcImageTask":{"TaskId":"up-1","Status":"WAITING","Progress":0}}}`,
			status:   model.TaskStatusSubmitted,
			progress: "0%",
		},
		{
			name:     "processing",
			body:     `{"Response":{"Status":"PROCESSING","AigcImageTask":{"TaskId":"up-1","Status":"PROCESSING","Progress":42}}}`,
			status:   model.TaskStatusInProgress,
			progress: "42%",
		},
		{
			name:     "success",
			body:     `{"Response":{"Status":"FINISH","AigcImageTask":{"TaskId":"up-1","Status":"FINISH","ErrCode":0,"ErrCodeExt":"","Progress":100,"Output":{"FileInfos":[{"FileUrl":"https://example.com/result.png"}]}}}}`,
			status:   model.TaskStatusSuccess,
			progress: "100%",
			url:      "https://example.com/result.png",
		},
		{
			name:       "finished without output URL",
			body:       `{"Response":{"Status":"FINISH","AigcImageTask":{"TaskId":"up-1","Status":"FINISH","ErrCode":0,"ErrCodeExt":"","Progress":100,"Output":{"FileInfos":[]}}}}`,
			status:     model.TaskStatusFailure,
			progress:   "100%",
			reasonPart: "no output image URL",
		},
		{
			name:     "uses first non-empty output URL",
			body:     `{"Response":{"Status":"FINISH","AigcImageTask":{"TaskId":"up-1","Status":"FINISH","ErrCode":0,"ErrCodeExt":"","Progress":100,"Output":{"FileInfos":[{"FileUrl":""},{"FileUrl":"https://example.com/result-2.png"}]}}}}`,
			status:   model.TaskStatusSuccess,
			progress: "100%",
			url:      "https://example.com/result-2.png",
		},
		{
			name:       "task failure",
			body:       `{"Response":{"Status":"FINISH","AigcImageTask":{"TaskId":"up-1","Status":"FINISH","ErrCode":1,"ErrCodeExt":"InvalidPrompt","Message":"blocked","Progress":100}}}`,
			status:     model.TaskStatusFailure,
			progress:   "100%",
			reasonPart: "InvalidPrompt blocked",
		},
		{
			name:    "outer error",
			body:    `{"Response":{"Error":{"Code":"AuthFailure","Message":"bad signature"}}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := (&TaskAdaptor{}).ParseTaskResult([]byte(tt.body))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, tt.status, model.TaskStatus(info.Status))
			assert.Equal(t, tt.progress, info.Progress)
			assert.Equal(t, tt.url, info.Url)
			assert.Contains(t, info.Reason, tt.reasonPart)
		})
	}
}
