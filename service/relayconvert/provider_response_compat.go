package relayconvert

import (
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	claudemessages "github.com/QuantumNous/new-api/service/relayconvert/internal/claude_messages"
	geminichat "github.com/QuantumNous/new-api/service/relayconvert/internal/gemini_chat"
	oaichat "github.com/QuantumNous/new-api/service/relayconvert/internal/oai_chat"
)

type ClaudeResponseInfo = claudemessages.ClaudeResponseInfo

func NormalizeCacheCreationSplit(totalTokens int, tokens5m int, tokens1h int) (int, int) {
	return oaichat.NormalizeCacheCreationSplit(totalTokens, tokens5m, tokens1h)
}

func ResponseOpenAI2Claude(openAIResponse *dto.OpenAITextResponse, info *relaycommon.RelayInfo) *dto.ClaudeResponse {
	return oaichat.ResponseOpenAI2Claude(openAIResponse, info)
}

func StreamResponseOpenAI2Claude(openAIResponse *dto.ChatCompletionsStreamResponse, info *relaycommon.RelayInfo) []*dto.ClaudeResponse {
	return oaichat.StreamResponseOpenAI2Claude(openAIResponse, info)
}

func ResponseOpenAI2Gemini(openAIResponse *dto.OpenAITextResponse, info *relaycommon.RelayInfo) *dto.GeminiChatResponse {
	return oaichat.ResponseOpenAI2Gemini(openAIResponse, info)
}

func StreamResponseOpenAI2Gemini(openAIResponse *dto.ChatCompletionsStreamResponse, info *relaycommon.RelayInfo) *dto.GeminiChatResponse {
	return oaichat.StreamResponseOpenAI2Gemini(openAIResponse, info)
}

func StopReasonClaudeToOpenAI(reason string) string {
	return claudemessages.StopReasonClaudeToOpenAI(reason)
}

func StreamResponseClaude2OpenAI(claudeResponse *dto.ClaudeResponse) *dto.ChatCompletionsStreamResponse {
	return claudemessages.StreamResponseClaude2OpenAI(claudeResponse)
}

func ResponseClaude2OpenAI(claudeResponse *dto.ClaudeResponse) *dto.OpenAITextResponse {
	return claudemessages.ResponseClaude2OpenAI(claudeResponse)
}

func UsageFromClaudeAPIUsage(usage *dto.ClaudeUsage) *dto.Usage {
	return claudemessages.UsageFromClaudeAPIUsage(usage)
}

func UsageFromClaudeUsage(usage *dto.Usage) *dto.Usage {
	return claudemessages.UsageFromClaudeUsage(usage)
}

func BuildMessageDeltaPatchUsage(claudeResponse *dto.ClaudeResponse, claudeInfo *ClaudeResponseInfo) *dto.ClaudeUsage {
	return claudemessages.BuildMessageDeltaPatchUsage(claudeResponse, claudeInfo)
}

func PatchClaudeMessageDeltaUsageData(data string, usage *dto.ClaudeUsage) string {
	return claudemessages.PatchClaudeMessageDeltaUsageData(data, usage)
}

func FormatClaudeResponseInfo(claudeResponse *dto.ClaudeResponse, oaiResponse *dto.ChatCompletionsStreamResponse, claudeInfo *ClaudeResponseInfo) bool {
	return claudemessages.FormatClaudeResponseInfo(claudeResponse, oaiResponse, claudeInfo)
}

func UsageFromGeminiMetadata(metadata *dto.GeminiUsageMetadata, fallbackPromptTokens int) *dto.Usage {
	return geminichat.UsageFromGeminiMetadata(metadata, fallbackPromptTokens)
}

func ResponseGeminiChat2OpenAI(id string, created int64, response *dto.GeminiChatResponse) *dto.OpenAITextResponse {
	return geminichat.ResponseGeminiChat2OpenAI(id, created, response)
}

func StreamResponseGeminiChat2OpenAI(geminiResponse *dto.GeminiChatResponse) (*dto.ChatCompletionsStreamResponse, bool) {
	return geminichat.StreamResponseGeminiChat2OpenAI(geminiResponse)
}
