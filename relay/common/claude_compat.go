package common

import (
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var unsupportedClaudeSamplingParamPaths = []string{
	"temperature",
	"top_p",
	"top_k",
}

func RemoveUnsupportedClaudeSamplingParams(jsonData []byte, info *RelayInfo) ([]byte, error) {
	model, ok := unsupportedClaudeSamplingParamsModel(info, jsonData)
	if !ok || !model_setting.ShouldOmitClaudeSamplingParams(model) {
		return jsonData, nil
	}

	result := jsonData
	for _, path := range unsupportedClaudeSamplingParamPaths {
		if !gjson.GetBytes(result, path).Exists() {
			continue
		}
		next, err := sjson.DeleteBytes(result, path)
		if err != nil {
			return nil, err
		}
		result = next
	}
	return result, nil
}

func unsupportedClaudeSamplingParamsModel(info *RelayInfo, jsonData []byte) (string, bool) {
	if info == nil {
		return "", false
	}

	model := ""
	if info.ChannelMeta != nil {
		model = strings.TrimSpace(info.ChannelMeta.UpstreamModelName)
	}
	if model == "" {
		model = strings.TrimSpace(gjson.GetBytes(jsonData, "model").String())
	}
	if model == "" {
		return "", false
	}

	if info.GetFinalRequestRelayFormat() == types.RelayFormatClaude {
		return model, true
	}
	if info.ChannelMeta == nil {
		return "", false
	}
	switch info.ChannelMeta.ApiType {
	case constant.APITypeAnthropic, constant.APITypeAws, constant.APITypeVertexAi:
		return model, true
	default:
		return "", false
	}
}
