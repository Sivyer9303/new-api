package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
)

func TestInitTaskPersistsSelectedBrioiKeyForPolling(t *testing.T) {
	task := InitTask(constant.TaskPlatform("62"), &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeBrioi,
			ApiKey:      "selected-brioi-key",
		},
	})

	assert.Equal(t, "selected-brioi-key", task.PrivateData.Key)
}

func TestInitTaskPersistsSelectedSilkRoadKeyForPolling(t *testing.T) {
	task := InitTask(constant.TaskPlatform("61"), &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeSilkRoad,
			ApiKey:      "selected-silkroad-key",
		},
	})

	assert.Equal(t, "selected-silkroad-key", task.PrivateData.Key)
}
