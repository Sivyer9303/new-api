package service

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCacheGetRandomSatisfiedChannelKeepsResolvedVideoProfileOnRetry(t *testing.T) {
	previousDB := model.DB
	previousMemoryCache := common.MemoryCacheEnabled
	t.Cleanup(func() {
		model.DB = previousDB
		common.MemoryCacheEnabled = previousMemoryCache
	})

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}))
	model.DB = db
	common.MemoryCacheEnabled = false

	higherPriority := int64(100)
	lowerPriority := int64(10)
	selectedMapping := `{"public-video-model":"selected-upstream-model"}`
	otherMapping := `{"public-video-model":"other-upstream-model"}`
	require.NoError(t, db.Create(&[]model.Channel{
		{Id: 931, Type: constant.ChannelTypeSilkRoad, Key: "selected-key", Status: common.ChannelStatusEnabled, Name: "selected", Group: "video", Models: "public-video-model", ModelMapping: &selectedMapping},
		{Id: 932, Type: constant.ChannelTypeSilkRoad, Key: "other-key", Status: common.ChannelStatusEnabled, Name: "other", Group: "video", Models: "public-video-model", ModelMapping: &otherMapping},
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "video", Model: "public-video-model", ChannelId: 931, Enabled: true, Priority: &lowerPriority},
		{Group: "video", Model: "public-video-model", ChannelId: 932, Enabled: true, Priority: &higherPriority},
	}).Error)

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeyVideoRouteDecision, model.VideoRouteDecision{
		Group:         "video",
		ChannelID:     931,
		ChannelType:   constant.ChannelTypeSilkRoad,
		PublicModel:   "public-video-model",
		UpstreamModel: "selected-upstream-model",
		RequestPath:   "/v1/videos",
	})

	retry := 1
	channel, selectedGroup, err := CacheGetRandomSatisfiedChannel(&RetryParam{
		Ctx:         ctx,
		TokenGroup:  "video",
		ModelName:   "public-video-model",
		RequestPath: "/v1/videos",
		Retry:       &retry,
	})

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, "video", selectedGroup)
	assert.Equal(t, 931, channel.Id)
}

func TestCacheGetRandomSatisfiedChannelAdvancesAutoGroupAfterResolvedRouteIsExhausted(t *testing.T) {
	previousDB := model.DB
	previousMemoryCache := common.MemoryCacheEnabled
	previousUsableGroups := setting.UserUsableGroups2JSONString()
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		model.DB = previousDB
		common.MemoryCacheEnabled = previousMemoryCache
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(previousUsableGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
	})
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","first":"First","second":"Second"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"first":1,"second":1}`))

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}))
	model.DB = db
	common.MemoryCacheEnabled = false

	priority := int64(10)
	firstMapping := `{"public-video-model":"first-upstream-model"}`
	secondMapping := `{"public-video-model":"second-upstream-model"}`
	require.NoError(t, db.Create(&[]model.Channel{
		{Id: 941, Type: constant.ChannelTypeSilkRoad, Key: "disabled-key", Status: common.ChannelStatusManuallyDisabled, Name: "disabled", Group: "first", Models: "public-video-model", ModelMapping: &firstMapping},
		{Id: 942, Type: constant.ChannelTypeSilkRoad, Key: "second-key", Status: common.ChannelStatusEnabled, Name: "second", Group: "second", Models: "public-video-model", ModelMapping: &secondMapping},
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "first", Model: "public-video-model", ChannelId: 941, Enabled: true, Priority: &priority},
		{Group: "second", Model: "public-video-model", ChannelId: 942, Enabled: true, Priority: &priority},
	}).Error)

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyTokenAutoGroups, []string{"first", "second"})
	common.SetContextKey(ctx, constant.ContextKeyAutoGroupIndex, 0)
	common.SetContextKey(ctx, constant.ContextKeyVideoRouteDecision, model.VideoRouteDecision{
		Group:         "first",
		ChannelID:     941,
		ChannelType:   constant.ChannelTypeSilkRoad,
		PublicModel:   "public-video-model",
		UpstreamModel: "first-upstream-model",
		RequestPath:   "/v1/videos",
	})

	retry := 0
	channel, selectedGroup, err := CacheGetRandomSatisfiedChannel(&RetryParam{
		Ctx:         ctx,
		TokenGroup:  "auto",
		ModelName:   "public-video-model",
		RequestPath: "/v1/videos",
		Retry:       &retry,
	})

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, 942, channel.Id)
	assert.Equal(t, "second", selectedGroup)
	decision, ok := common.GetContextKey(ctx, constant.ContextKeyVideoRouteDecision)
	require.True(t, ok)
	assert.Equal(t, "second", decision.(model.VideoRouteDecision).Group)
}
