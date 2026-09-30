package helper

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/shopspring/decimal"
)

func applyTencentTokenHubSearchPreConsume(info *relaycommon.RelayInfo, priceData *types.PriceData) {
	if info == nil || priceData == nil {
		return
	}

	source, requested := relaycommon.TencentTokenHubWebSearchRequested(info)
	if !requested {
		return
	}

	toolName := constant.ToolNameTencentWebSearchStandard
	if source == "lite" {
		toolName = constant.ToolNameTencentWebSearchLite
	}
	pricePer1K := operation_setting.GetToolPriceForModel(toolName, info.OriginModelName)
	if pricePer1K <= 0 || priceData.GroupRatioInfo.GroupRatio <= 0 {
		return
	}

	surcharge := decimal.NewFromFloat(pricePer1K).
		Div(decimal.NewFromInt(1000)).
		Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
		Mul(decimal.NewFromFloat(priceData.GroupRatioInfo.GroupRatio))
	if info.TieredBillingSnapshot == nil {
		surcharge = priceData.ApplyOtherRatiosToDecimal(surcharge)
	}

	reservedQuota, clamp := common.QuotaFromDecimalChecked(surcharge)
	if clamp != nil && info.QuotaClamp == nil {
		info.QuotaClamp = clamp
	}
	if reservedQuota <= 0 {
		return
	}

	totalQuota, clamp := common.QuotaFromDecimalChecked(
		decimal.NewFromInt(int64(priceData.QuotaToPreConsume)).Add(decimal.NewFromInt(int64(reservedQuota))),
	)
	if clamp != nil && info.QuotaClamp == nil {
		info.QuotaClamp = clamp
	}
	priceData.QuotaToPreConsume = totalQuota
	priceData.FreeModel = false
	info.ToolCallPreConsumedQuota = reservedQuota
}
