package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
)

func TestInitTaskKeepsVODAIGCSubmissionKey(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeVODAIGC,
			ApiKey:      "selected-secret-id|selected-secret-key",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{SubAppID: 123456},
	}

	task := InitTask(constant.TaskPlatform("9007"), relayInfo)

	assert.Equal(t, "selected-secret-id|selected-secret-key", task.PrivateData.Key)
	assert.Equal(t, int64(123456), task.PrivateData.SubAppID)
}
