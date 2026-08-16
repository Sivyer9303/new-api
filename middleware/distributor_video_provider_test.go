package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	appi18n "github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDistributeConstrainsVideoProviderAcrossPriorityAndFailure(t *testing.T) {
	require.NoError(t, appi18n.Init())
	gin.SetMode(gin.TestMode)

	previousDB := model.DB
	previousMemoryCache := common.MemoryCacheEnabled
	brioi := brioi_setting.GetBrioiSetting()
	previousBrioi := *brioi
	silkRoad := silkroad_setting.GetSilkRoadSetting()
	previousSilkRoadGroups := append([]string(nil), silkRoad.VideoToolGroups...)
	t.Cleanup(func() {
		model.DB = previousDB
		common.MemoryCacheEnabled = previousMemoryCache
		*brioi = previousBrioi
		silkRoad.VideoToolGroups = previousSilkRoadGroups
	})

	common.MemoryCacheEnabled = false
	*brioi = brioi_setting.DefaultBrioiSetting()
	brioi.VideoToolGroups = []string{"brioi-owned"}
	silkRoad.VideoToolGroups = []string{"silkroad-owned"}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}))
	model.DB = db

	highPriority := int64(100)
	lowPriority := int64(10)
	silkRoadChannel := model.Channel{
		Id:       901,
		Type:     constant.ChannelTypeSilkRoad,
		Name:     "silkroad-same-model",
		Key:      "silkroad-key",
		Status:   common.ChannelStatusEnabled,
		Group:    "brioi-owned",
		Models:   "same-video-model",
		Priority: &highPriority,
	}
	brioiChannel := model.Channel{
		Id:       902,
		Type:     constant.ChannelTypeBrioi,
		Name:     "brioi-same-model",
		Key:      "brioi-key",
		Status:   common.ChannelStatusEnabled,
		Group:    "brioi-owned",
		Models:   "same-video-model",
		Priority: &lowPriority,
	}
	require.NoError(t, db.Create(&[]model.Channel{silkRoadChannel, brioiChannel}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{
			Group:     "brioi-owned",
			Model:     "same-video-model",
			ChannelId: silkRoadChannel.Id,
			Enabled:   true,
			Priority:  &highPriority,
		},
		{
			Group:     "brioi-owned",
			Model:     "same-video-model",
			ChannelId: brioiChannel.Id,
			Enabled:   true,
			Priority:  &lowPriority,
		},
	}).Error)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyUsingGroup, "brioi-owned")
		common.SetContextKey(c, constant.ContextKeyTokenGroup, "brioi-owned")
		c.Next()
	})
	router.Use(Distribute())
	selectedChannelType := 0
	router.POST("/v1/video/generations", func(c *gin.Context) {
		selectedChannelType = common.GetContextKeyInt(c, constant.ContextKeyChannelType)
		c.Status(http.StatusNoContent)
	})

	firstRecorder := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/video/generations",
		bytes.NewBufferString(`{"model":"same-video-model"}`),
	)
	firstRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(firstRecorder, firstRequest)

	require.Equal(t, http.StatusNoContent, firstRecorder.Code)
	assert.Equal(t, constant.ChannelTypeSilkRoad, selectedChannelType)

	require.NoError(t, db.Model(&model.Channel{}).
		Where("id = ?", silkRoadChannel.Id).
		Update("status", common.ChannelStatusManuallyDisabled).Error)

	selectedChannelType = 0
	secondRecorder := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/video/generations",
		bytes.NewBufferString(`{"model":"same-video-model"}`),
	)
	secondRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(secondRecorder, secondRequest)

	require.Equal(t, http.StatusNoContent, secondRecorder.Code)
	assert.Equal(t, constant.ChannelTypeBrioi, selectedChannelType)
}

