package kling

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestAdjustBillingOnCompleteRejectsOversizedBillingUnits(t *testing.T) {
	originalModelRatio := ratio_setting.ModelRatio2JSONString()
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(originalModelRatio)) })
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"kling-v2-master":0.14}`))

	task := &model.Task{Properties: model.Properties{OriginModelName: "kling-v2-master"}}
	result := &relaycommon.TaskInfo{BillingUnits: relaycommon.MaxTaskBillingUnits + 1}
	adaptor := &TaskAdaptor{}

	quota, clamp := adaptor.AdjustBillingOnCompleteWithClamp(task, result)
	assert.Zero(t, quota)
	assert.Nil(t, clamp)
	assert.Zero(t, adaptor.AdjustBillingOnComplete(task, result))
}
