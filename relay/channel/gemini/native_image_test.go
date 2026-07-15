package gemini

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertImageRequestSupportsNativeGeminiImageModel(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-3.1-flash-image",
		},
	}
	request := dto.ImageRequest{
		Model:   "gemini-3.1-flash-image",
		Prompt:  "generate a puppy photo",
		N:       common.GetPointer(uint(2)),
		Size:    "1792x1024",
		Quality: "hd",
	}

	converted, err := (&Adaptor{}).ConvertImageRequest(nil, info, request)
	require.NoError(t, err)

	geminiRequest, ok := converted.(*dto.GeminiChatRequest)
	require.True(t, ok)
	require.Len(t, geminiRequest.Contents, 1)
	require.Len(t, geminiRequest.Contents[0].Parts, 1)
	assert.Equal(t, "generate a puppy photo", geminiRequest.Contents[0].Parts[0].Text)
	assert.Equal(t, []string{"TEXT", "IMAGE"}, geminiRequest.GenerationConfig.ResponseModalities)
	require.NotNil(t, geminiRequest.GenerationConfig.CandidateCount)
	assert.Equal(t, 2, *geminiRequest.GenerationConfig.CandidateCount)

	var imageConfig map[string]string
	require.NoError(t, common.Unmarshal(geminiRequest.GenerationConfig.ImageConfig, &imageConfig))
	assert.Equal(t, "16:9", imageConfig["aspectRatio"])
	assert.Equal(t, "2K", imageConfig["imageSize"])
}

func TestGeminiNativeImageHandlerConvertsInlineImageToOpenAIImageResponse(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-3-pro-image",
		},
	}

	payload := dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{
			{
				Content: dto.GeminiChatContent{
					Role: "model",
					Parts: []dto.GeminiPart{
						{Text: "A puppy photo."},
						{
							InlineData: &dto.GeminiInlineData{
								MimeType: "image/png",
								Data:     "base64-image-data",
							},
						},
					},
				},
			},
		},
		UsageMetadata: dto.GeminiUsageMetadata{
			PromptTokenCount:     12,
			CandidatesTokenCount: 34,
			TotalTokenCount:      46,
		},
	}

	body, err := common.Marshal(payload)
	require.NoError(t, err)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}

	usage, newAPIError := GeminiNativeImageHandler(c, info, resp)
	require.Nil(t, newAPIError)
	require.NotNil(t, usage)
	assert.Equal(t, 12, usage.PromptTokens)
	assert.Equal(t, 34, usage.CompletionTokens)
	assert.Equal(t, 46, usage.TotalTokens)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var imageResponse dto.ImageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &imageResponse))
	require.Len(t, imageResponse.Data, 1)
	assert.Equal(t, "base64-image-data", imageResponse.Data[0].B64Json)
	assert.Equal(t, "A puppy photo.", imageResponse.Data[0].RevisedPrompt)
}
