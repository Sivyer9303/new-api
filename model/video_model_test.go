package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestResolveVideoRouteFiltersChannelsByRequestPathBeforePriority(t *testing.T) {
	previousDB := DB
	previousMemoryCache := common.MemoryCacheEnabled
	t.Cleanup(func() {
		DB = previousDB
		common.MemoryCacheEnabled = previousMemoryCache
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}))
	DB = db
	common.MemoryCacheEnabled = false

	highPriority := int64(100)
	lowPriority := int64(10)
	require.NoError(t, db.Create(&[]Channel{
		{
			Id:       1101,
			Type:     constant.ChannelTypeBrioi,
			Name:     "brioi-openai-video",
			Key:      "brioi-key",
			Status:   common.ChannelStatusEnabled,
			Group:    "video",
			Models:   "shared-model",
			Priority: &highPriority,
		},
		{
			Id:       1102,
			Type:     constant.ChannelTypeSilkRoad,
			Name:     "silkroad-openai-video",
			Key:      "silkroad-key",
			Status:   common.ChannelStatusEnabled,
			Group:    "video",
			Models:   "shared-model",
			Priority: &lowPriority,
		},
	}).Error)
	require.NoError(t, db.Create(&[]Ability{
		{Group: "video", Model: "shared-model", ChannelId: 1101, Enabled: true, Priority: &highPriority},
		{Group: "video", Model: "shared-model", ChannelId: 1102, Enabled: true, Priority: &lowPriority},
	}).Error)

	decision, err := ResolveVideoRoute([]string{"video"}, "shared-model", "/v1/videos")
	require.NoError(t, err)
	assert.Equal(t, "video", decision.Group)
	assert.Equal(t, 1102, decision.ChannelID)
	assert.Equal(t, constant.ChannelTypeSilkRoad, decision.ChannelType)
	assert.Equal(t, "shared-model", decision.UpstreamModel)
	assert.Equal(t, "/v1/videos", decision.RequestPath)
}
