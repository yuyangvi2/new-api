package service

import (
	"fmt"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service/relayconvert"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func init() {
	relayconvert.SetMediaResolver(relayconvert.MediaResolver{
		GetBase64Data:        GetBase64Data,
		DecodeBase64FileData: DecodeBase64FileData,
	})
}

func ConvertRequest(c *gin.Context, info *relaycommon.RelayInfo, target types.RelayFormat, request any) (*relayconvert.RequestResult, error) {
	return relayconvert.ConvertRequest(c, info, target, request)
}

func ConvertRequestByID(c *gin.Context, info *relaycommon.RelayInfo, converter string, request any) (*relayconvert.RequestResult, error) {
	return relayconvert.ConvertRequestByID(c, info, converter, request)
}

func ConvertRequestVia(c *gin.Context, info *relaycommon.RelayInfo, request any, path ...types.RelayFormat) (*relayconvert.RequestResult, error) {
	return relayconvert.ConvertRequestVia(c, info, request, path...)
}

func ConvertResponse(c *gin.Context, info *relaycommon.RelayInfo, target types.RelayFormat, response any) (*relayconvert.ResponseResult, error) {
	return relayconvert.ConvertResponse(c, info, target, response)
}

func ConvertResponseByID(c *gin.Context, info *relaycommon.RelayInfo, converter string, response any) (*relayconvert.ResponseResult, error) {
	return relayconvert.ConvertResponseByID(c, info, converter, response)
}

func ConvertStreamResponse(c *gin.Context, info *relaycommon.RelayInfo, target types.RelayFormat, response any) (*relayconvert.ResponseResult, error) {
	return relayconvert.ConvertStreamResponse(c, info, target, response)
}

func NewResponseStreamState(from types.RelayFormat, target types.RelayFormat, options relayconvert.ResponseStreamOptions) (*relayconvert.ResponseStreamState, error) {
	return relayconvert.NewResponseStreamState(from, target, options)
}

func NewResponseStreamStateByID(converter string, options relayconvert.ResponseStreamOptions) (*relayconvert.ResponseStreamState, error) {
	return relayconvert.NewResponseStreamStateByID(converter, options)
}

func ConvertStreamResponseChunk(c *gin.Context, info *relaycommon.RelayInfo, state *relayconvert.ResponseStreamState, response any) ([]relayconvert.ResponseResult, error) {
	return relayconvert.ConvertStreamResponseChunk(c, info, state, response)
}

func FinalizeStreamResponse(c *gin.Context, info *relaycommon.RelayInfo, state *relayconvert.ResponseStreamState) ([]relayconvert.ResponseResult, error) {
	return relayconvert.FinalizeStreamResponse(c, info, state)
}

func ConvertClaudeRequestToOpenAIChat(claudeRequest dto.ClaudeRequest, info *relaycommon.RelayInfo) (*dto.GeneralOpenAIRequest, error) {
	result, err := ConvertRequest(nil, info, types.RelayFormatOpenAI, &claudeRequest)
	if err != nil {
		return nil, err
	}
	openAIRequest, ok := result.Value.(*dto.GeneralOpenAIRequest)
	if !ok {
		return nil, fmt.Errorf("expected OpenAI chat completions request, got %T", result.Value)
	}
	return openAIRequest, nil
}

func ConvertGeminiRequestToOpenAIChat(geminiRequest *dto.GeminiChatRequest, info *relaycommon.RelayInfo) (*dto.GeneralOpenAIRequest, error) {
	result, err := ConvertRequest(nil, info, types.RelayFormatOpenAI, geminiRequest)
	if err != nil {
		return nil, err
	}
	openAIRequest, ok := result.Value.(*dto.GeneralOpenAIRequest)
	if !ok {
		return nil, fmt.Errorf("expected OpenAI chat completions request, got %T", result.Value)
	}
	return openAIRequest, nil
}
