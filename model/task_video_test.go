package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitTaskPersistsSelectedBrioiKeyForPolling(t *testing.T) {
	task := InitTask(constant.TaskPlatform("62"), &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeBrioi,
			ApiKey:      "selected-brioi-key",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			RequestSnapshot: &relaycommon.TaskRequestSnapshot{
				Model:          "seedance-2-0",
				Prompt:         "保持 @图片1",
				GenerationType: "reference_videos",
				Duration:       10,
				Resolution:     "720p",
				AspectRatio:    "16:9",
				Media: []relaycommon.TaskMediaSnapshot{
					{Type: "image", Role: "reference"},
					{Type: "video", Role: "reference"},
				},
			},
		},
	})

	assert.Equal(t, "selected-brioi-key", task.PrivateData.Key)
	require.NotNil(t, task.Properties.Request)
	assert.Equal(t, "reference_videos", task.Properties.Request.GenerationType)
	assert.Equal(t, "保持 @图片1", task.Properties.Request.Prompt)
	assert.Equal(t, []relaycommon.TaskMediaSnapshot{
		{Type: "image", Role: "reference"},
		{Type: "video", Role: "reference"},
	}, task.Properties.Request.Media)
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
