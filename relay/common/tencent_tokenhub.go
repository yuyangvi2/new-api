package common

import (
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

const (
	tencentTokenHubHost                 = "tokenhub.tencentmaas.com"
	tencentTokenHubSearchSourceLite     = "lite"
	tencentTokenHubSearchSourceStandard = "standard"
)

func IsTencentTokenHub(info *RelayInfo) bool {
	if info == nil || info.ChannelMeta == nil {
		return false
	}
	parsed, err := url.Parse(strings.TrimSpace(info.ChannelBaseUrl))
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Hostname(), tencentTokenHubHost)
}

func TencentTokenHubWebSearchSource(info *RelayInfo) (string, bool) {
	if !IsTencentTokenHub(info) {
		return "", false
	}

	source := tencentTokenHubSearchSourceStandard
	switch request := info.Request.(type) {
	case *dto.GeneralOpenAIRequest:
		if request.WebSearchOptions != nil && strings.EqualFold(strings.TrimSpace(request.WebSearchOptions.SearchSource), tencentTokenHubSearchSourceLite) {
			source = tencentTokenHubSearchSourceLite
		}
	case *dto.OpenAIResponsesRequest:
		for _, tool := range request.GetToolsMap() {
			if !strings.EqualFold(strings.TrimSpace(stringValue(tool["type"])), "web_search") {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(stringValue(tool["search_source"])), tencentTokenHubSearchSourceLite) {
				source = tencentTokenHubSearchSourceLite
			}
			break
		}
	}
	return source, true
}

func TencentTokenHubWebSearchRequested(info *RelayInfo) (string, bool) {
	source, ok := TencentTokenHubWebSearchSource(info)
	if !ok {
		return "", false
	}

	switch request := info.Request.(type) {
	case *dto.GeneralOpenAIRequest:
		if request.WebSearchOptions == nil || request.WebSearchOptions.Enable == nil || !*request.WebSearchOptions.Enable {
			return "", false
		}
	case *dto.OpenAIResponsesRequest:
		for _, tool := range request.GetToolsMap() {
			if strings.EqualFold(strings.TrimSpace(stringValue(tool["type"])), "web_search") {
				return source, true
			}
		}
		return "", false
	default:
		return "", false
	}

	return source, true
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
