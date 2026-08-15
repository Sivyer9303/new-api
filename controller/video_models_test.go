package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type videoModelsAPIResponse struct {
	Success bool                    `json:"success"`
	Data    videoToolModelsResponse `json:"data"`
}

func configureVideoModelsTestSettings(t *testing.T, brioiGroups []string) {
	t.Helper()

	brioi := brioi_setting.GetBrioiSetting()
	previousBrioi := *brioi
	silkRoad := silkroad_setting.GetSilkRoadSetting()
	previousSilkRoadGroups := append([]string(nil), silkRoad.VideoToolGroups...)
	previousPrices := ratio_setting.ModelPrice2JSONString()
	previousModelRatios := ratio_setting.ModelRatio2JSONString()
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		*brioi = previousBrioi
		silkRoad.VideoToolGroups = previousSilkRoadGroups
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(previousPrices))
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(previousModelRatios))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
	})

	*brioi = brioi_setting.DefaultBrioiSetting()
	brioi.VideoToolGroups = append([]string(nil), brioiGroups...)
	silkRoad.VideoToolGroups = []string{"silkroad-only"}

	prices, err := common.Marshal(map[string]float64{
		"shared-video":       0.2,
		"mapped-public":      0.3,
		"token-limited-away": 0.4,
		"disabled-only":      0.5,
		"chat-only":          0.6,
	})
	require.NoError(t, err)
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(string(prices)))
	groupRatios := map[string]float64{
		"default":       1,
		"brioi-group":   1,
		"brioi-empty":   1,
		"silkroad-only": 1,
	}
	for _, group := range brioiGroups {
		groupRatios[group] = 1
	}
	groupRatioJSON, err := common.Marshal(groupRatios)
	require.NoError(t, err)
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(string(groupRatioJSON)))
}

func createVideoModelsChannel(
	t *testing.T,
	channelID int,
	channelType int,
	status int,
	group string,
	models []string,
	mapping map[string]string,
) {
	t.Helper()

	mappingJSON, err := common.Marshal(mapping)
	require.NoError(t, err)
	createVideoModelsChannelWithRawMapping(
		t,
		channelID,
		channelType,
		status,
		group,
		models,
		string(mappingJSON),
	)
}

func createVideoModelsChannelWithRawMapping(
	t *testing.T,
	channelID int,
	channelType int,
	status int,
	group string,
	models []string,
	rawMapping string,
) {
	t.Helper()

	channel := &model.Channel{
		Id:           channelID,
		Type:         channelType,
		Name:         fmt.Sprintf("channel-%d", channelID),
		Key:          "test-key",
		Status:       status,
		Group:        group,
		Models:       strings.Join(models, ","),
		ModelMapping: &rawMapping,
	}
	require.NoError(t, model.DB.Create(channel).Error)

	abilities := make([]model.Ability, 0, len(models))
	for _, modelName := range models {
		abilities = append(abilities, model.Ability{
			Group:     group,
			Model:     modelName,
			ChannelId: channelID,
			Enabled:   true,
		})
	}
	require.NoError(t, model.DB.Create(&abilities).Error)
}

func callVideoModelsEndpoint(t *testing.T, userID, tokenID int) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/video/models?token_id=%d", tokenID),
		nil,
	)
	context.Set("id", userID)
	var token model.Token
	if err := model.DB.Where("id = ?", tokenID).First(&token).Error; err == nil {
		context.Set("user_group", token.Group)
	}
	GetVideoToolModels(context)
	return recorder
}

func decodeVideoModelsResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) videoModelsAPIResponse {
	t.Helper()

	var response videoModelsAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestGetVideoToolModelsAppliesProviderTokenBillingAndChannelEligibility(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}))
	configureVideoModelsTestSettings(t, []string{"brioi-group"})

	require.NoError(t, db.Create(&model.Token{
		Id:                 701,
		UserId:             101,
		Key:                "video-model-token",
		Status:             common.TokenStatusEnabled,
		Group:              "brioi-group",
		ModelLimitsEnabled: true,
		ModelLimits:        "shared-video,mapped-public,unpriced-video",
	}).Error)

	createVideoModelsChannel(
		t,
		801,
		constant.ChannelTypeBrioi,
		common.ChannelStatusEnabled,
		"brioi-group",
		[]string{"shared-video", "mapped-public", "token-limited-away", "unpriced-video"},
		map[string]string{
			"shared-video":       brioi_setting.ModelSeedance20,
			"mapped-public":      brioi_setting.ModelSeedance25,
			"token-limited-away": brioi_setting.ModelSeedance20Fast,
			"unpriced-video":     brioi_setting.ModelSeedance20,
		},
	)
	createVideoModelsChannel(
		t,
		802,
		constant.ChannelTypeSilkRoad,
		common.ChannelStatusEnabled,
		"brioi-group",
		[]string{"shared-video"},
		map[string]string{"shared-video": "silkroad-upstream"},
	)
	createVideoModelsChannel(
		t,
		803,
		constant.ChannelTypeBrioi,
		common.ChannelStatusManuallyDisabled,
		"brioi-group",
		[]string{"disabled-only"},
		map[string]string{"disabled-only": brioi_setting.ModelSeedance20},
	)
	createVideoModelsChannel(
		t,
		804,
		constant.ChannelTypeOpenAI,
		common.ChannelStatusEnabled,
		"brioi-group",
		[]string{"chat-only"},
		map[string]string{"chat-only": brioi_setting.ModelSeedance20},
	)

	recorder := callVideoModelsEndpoint(t, 101, 701)
	require.Equal(t, http.StatusOK, recorder.Code)
	response := decodeVideoModelsResponse(t, recorder)
	require.True(t, response.Success)
	assert.Equal(t, "brioi-group", response.Data.Group)
	assert.Equal(t, []string{"brioi-group"}, response.Data.ResolvedGroups)
	assert.Equal(t, "brioi", string(response.Data.Provider))
	assert.Empty(t, response.Data.Reason)

	modelIDs := make([]string, 0, len(response.Data.Models))
	for _, publicModel := range response.Data.Models {
		modelIDs = append(modelIDs, publicModel.ID)
		assert.Equal(t, "brioi", publicModel.OwnedBy)
		assert.Equal(t, "brioi", string(publicModel.ProviderID))
	}
	assert.ElementsMatch(t, []string{"shared-video", "mapped-public"}, modelIDs)
	profileByModel := make(map[string]string, len(response.Data.Models))
	for _, publicModel := range response.Data.Models {
		profileByModel[publicModel.ID] = publicModel.ProfileModel
	}
	assert.Equal(t, brioi_setting.ModelSeedance20, profileByModel["shared-video"])
	assert.Equal(t, brioi_setting.ModelSeedance25, profileByModel["mapped-public"])
}

func TestGetVideoToolModelsListsEveryVideoChannelInTheKeyGroup(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}))
	configureVideoModelsTestSettings(t, []string{"brioi-empty"})
	require.NoError(t, db.Create(&model.Token{
		Id:     702,
		UserId: 102,
		Key:    "video-empty-token",
		Status: common.TokenStatusEnabled,
		Group:  "brioi-empty",
	}).Error)
	createVideoModelsChannel(
		t,
		805,
		constant.ChannelTypeSilkRoad,
		common.ChannelStatusEnabled,
		"brioi-empty",
		[]string{"shared-video"},
		map[string]string{"shared-video": "silkroad-upstream"},
	)

	response := decodeVideoModelsResponse(t, callVideoModelsEndpoint(t, 102, 702))
	require.True(t, response.Success)
	assert.Equal(t, "silkroad", string(response.Data.Provider))
	require.Len(t, response.Data.Models, 1)
	assert.Equal(t, "shared-video", response.Data.Models[0].ID)
	assert.Equal(t, "silkroad", string(response.Data.Models[0].ProviderID))
	assert.Empty(t, response.Data.Reason)
}

func TestGetVideoToolModelsRequiresTokenOwnership(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}))
	require.NoError(t, db.Create(&model.Token{
		Id:     703,
		UserId: 103,
		Key:    "other-users-token",
		Status: common.TokenStatusEnabled,
		Group:  "default",
	}).Error)

	recorder := callVideoModelsEndpoint(t, 104, 703)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	response := decodeVideoModelsResponse(t, recorder)
	assert.False(t, response.Success)
}

func TestGetVideoToolModelsRequiresEnabledToken(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}))

	for i, status := range []int{
		common.TokenStatusDisabled,
		common.TokenStatusExpired,
		common.TokenStatusExhausted,
	} {
		tokenID := 710 + i
		require.NoError(t, db.Create(&model.Token{
			Id:     tokenID,
			UserId: 110,
			Key:    fmt.Sprintf("unavailable-video-token-%d", status),
			Status: status,
			Group:  "default",
		}).Error)

		recorder := callVideoModelsEndpoint(t, 110, tokenID)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		response := decodeVideoModelsResponse(t, recorder)
		assert.False(t, response.Success)
	}
}

func TestGetVideoToolModelsRejectsStaleUnauthorizedTokenGroup(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}))
	configureVideoModelsTestSettings(t, []string{"brioi-group"})
	require.NoError(t, db.Create(&model.Token{
		Id:     704,
		UserId: 105,
		Key:    "stale-group-token",
		Status: common.TokenStatusEnabled,
		Group:  "brioi-group",
	}).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/video/models?token_id=704",
		nil,
	)
	context.Set("id", 105)
	context.Set("user_group", "default")
	GetVideoToolModels(context)

	response := decodeVideoModelsResponse(t, recorder)
	require.True(t, response.Success)
	assert.Empty(t, response.Data.Models)
	assert.Equal(t, "token_group_unavailable", response.Data.Reason)
}

func TestGetVideoToolModelsFailsClosedWhenAnyRouteHasInvalidMapping(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}))
	configureVideoModelsTestSettings(t, []string{"brioi-group"})
	require.NoError(t, db.Create(&model.Token{
		Id:     705,
		UserId: 106,
		Key:    "invalid-mapping-token",
		Status: common.TokenStatusEnabled,
		Group:  "brioi-group",
	}).Error)

	prices, err := common.Marshal(map[string]float64{"mapping-conflict": 0.2})
	require.NoError(t, err)
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(string(prices)))
	createVideoModelsChannel(
		t,
		806,
		constant.ChannelTypeBrioi,
		common.ChannelStatusEnabled,
		"brioi-group",
		[]string{"mapping-conflict"},
		map[string]string{"mapping-conflict": brioi_setting.ModelSeedance20},
	)
	createVideoModelsChannelWithRawMapping(
		t,
		807,
		constant.ChannelTypeBrioi,
		common.ChannelStatusEnabled,
		"brioi-group",
		[]string{"mapping-conflict"},
		"{",
	)

	response := decodeVideoModelsResponse(t, callVideoModelsEndpoint(t, 106, 705))
	require.True(t, response.Success)
	assert.Empty(t, response.Data.Models)
	assert.Equal(t, "no_eligible_video_models", response.Data.Reason)
}

func TestGetVideoToolModelsRequiresPerCallModelPrice(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}))
	configureVideoModelsTestSettings(t, []string{"brioi-group"})
	require.NoError(t, db.Create(&model.Token{
		Id:     706,
		UserId: 107,
		Key:    "ratio-only-token",
		Status: common.TokenStatusEnabled,
		Group:  "brioi-group",
	}).Error)

	require.NoError(
		t,
		ratio_setting.UpdateModelRatioByJSONString(`{"ratio-only-video":1}`),
	)
	createVideoModelsChannel(
		t,
		808,
		constant.ChannelTypeBrioi,
		common.ChannelStatusEnabled,
		"brioi-group",
		[]string{"ratio-only-video"},
		map[string]string{"ratio-only-video": brioi_setting.ModelSeedance20},
	)

	response := decodeVideoModelsResponse(t, callVideoModelsEndpoint(t, 107, 706))
	require.True(t, response.Success)
	assert.Empty(t, response.Data.Models)
	assert.Equal(t, "no_eligible_video_models", response.Data.Reason)
}

func TestVideoTokenGroupsTreatsStoredEmptyAutoListAsInherited(t *testing.T) {
	previousAutoGroups := setting.AutoGroups2JsonString()
	previousGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(previousAutoGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousGroupRatios))
	})
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["default"]`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1}`))

	group, groups, err := videoTokenGroups(&model.Token{
		UserId:     108,
		Group:      "auto",
		AutoGroups: "[]",
	}, "default")
	require.NoError(t, err)
	assert.Equal(t, "auto", group)
	assert.Equal(t, []string{"default"}, groups)
}
