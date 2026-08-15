package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type videoToolModel struct {
	ID                     string                  `json:"id"`
	Object                 string                  `json:"object"`
	Created                int                     `json:"created"`
	OwnedBy                string                  `json:"owned_by"`
	ProfileModel           string                  `json:"profile_model"`
	ProviderID             setting.VideoProvider   `json:"provider_id"`
	ChannelType            int                     `json:"channel_type"`
	Profile                any                     `json:"profile,omitempty"`
	GenerationTypes        any                     `json:"generation_types,omitempty"`
	SupportedEndpointTypes []constant.EndpointType `json:"supported_endpoint_types"`
}

type videoToolModelsResponse struct {
	TokenID        int                   `json:"token_id"`
	Group          string                `json:"group"`
	ResolvedGroups []string              `json:"resolved_groups"`
	Provider       setting.VideoProvider `json:"provider,omitempty"`
	Models         []videoToolModel      `json:"models"`
	Reason         string                `json:"reason,omitempty"`
}

// GetVideoToolModels discovers Video Generation models for one API key owned by
// the authenticated dashboard user. It never accepts a client-supplied group or
// provider override.
func GetVideoToolModels(c *gin.Context) {
	tokenID, err := strconv.Atoi(strings.TrimSpace(c.Query("token_id")))
	if err != nil || tokenID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "token_id must be a positive integer",
		})
		return
	}

	userID := c.GetInt("id")
	token, err := model.GetTokenByIds(tokenID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "API key not found",
			})
			return
		}
		common.ApiError(c, err)
		return
	}
	if token.Status != common.TokenStatusEnabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "API key is not enabled",
		})
		return
	}

	group, candidateGroups, err := videoTokenGroups(
		token,
		strings.TrimSpace(c.GetString("user_group")),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	response := videoToolModelsResponse{
		TokenID:        token.Id,
		Group:          group,
		ResolvedGroups: []string{},
		Models:         []videoToolModel{},
	}
	if len(candidateGroups) == 0 {
		response.Reason = "token_group_unavailable"
		common.ApiSuccess(c, response)
		return
	}
	response.ResolvedGroups = candidateGroups

	routes, err := model.GetEligibleVideoModelRoutes(candidateGroups)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(routes) == 0 {
		response.Reason = "no_eligible_video_channels"
		common.ApiSuccess(c, response)
		return
	}

	type modelCapability struct {
		hit     model.EligibleVideoModelRoute
		invalid bool
		seen    bool
	}
	capabilities := make(map[string]modelCapability, len(routes))
	modelOrder := make([]string, 0, len(routes))
	for _, route := range routes {
		capability := capabilities[route.Model]
		if !capability.seen {
			modelOrder = append(modelOrder, route.Model)
			capability.seen = true
		}
		if route.InvalidMapping {
			capability.invalid = true
			capabilities[route.Model] = capability
			continue
		}
		if !capability.invalid {
			if hit, ok := model.PickVideoModelHitByName(routes, route.Model); ok {
				capability.hit = hit
			}
		}
		capabilities[route.Model] = capability
	}

	modelLimits := token.GetModelLimitsMap()
	providers := make(map[setting.VideoProvider]struct{})
	for _, modelName := range modelOrder {
		capability := capabilities[modelName]
		if capability.invalid || capability.hit.ChannelType <= 0 || capability.hit.UpstreamModel == "" {
			continue
		}
		if token.ModelLimitsEnabled {
			matchingName := ratio_setting.FormatMatchingModelName(modelName)
			if !modelLimits[modelName] && !modelLimits[matchingName] {
				continue
			}
		}
		modelPrice, priced := ratio_setting.GetModelPrice(modelName, false)
		if !priced || modelPrice <= 0 {
			continue
		}
		if billing_setting.GetBillingMode(modelName) == billing_setting.BillingModeTieredExpr {
			continue
		}

		provider, profile, generationTypes, ok := attachVideoToolCapabilities(
			capability.hit.ChannelType,
			capability.hit.UpstreamModel,
		)
		if !ok {
			continue
		}

		baseModel := buildOpenAIModel(modelName, nil)
		response.Models = append(response.Models, videoToolModel{
			ID:                     baseModel.Id,
			Object:                 baseModel.Object,
			Created:                baseModel.Created,
			OwnedBy:                string(provider),
			ProfileModel:           capability.hit.UpstreamModel,
			ProviderID:             provider,
			ChannelType:            capability.hit.ChannelType,
			Profile:                profile,
			GenerationTypes:        generationTypes,
			SupportedEndpointTypes: baseModel.SupportedEndpointTypes,
		})
		providers[provider] = struct{}{}
	}
	if len(response.Models) == 0 {
		response.Reason = "no_eligible_video_models"
	} else if len(providers) == 1 {
		for provider := range providers {
			response.Provider = provider
		}
	}
	common.ApiSuccess(c, response)
}

func videoTokenGroups(
	token *model.Token,
	userGroup string,
) (string, []string, error) {
	if userGroup == "" {
		var err error
		userGroup, err = model.GetUserGroup(token.UserId, false)
		if err != nil {
			return "", nil, err
		}
	}
	group := strings.TrimSpace(token.Group)
	if group != "" && group != "auto" {
		if !service.IsUserSelectableGroup(userGroup, group) {
			return group, []string{}, nil
		}
		return group, []string{group}, nil
	}

	if group == "" {
		return userGroup, []string{userGroup}, nil
	}

	if token.AutoGroups == "" {
		return group, service.GetUserAutoGroup(userGroup), nil
	}
	tokenGroups, err := token.GetAutoGroups()
	if err != nil {
		return "", nil, err
	}
	if len(tokenGroups) == 0 {
		return group, service.GetUserAutoGroup(userGroup), nil
	}
	return group, service.FilterUserTokenAutoGroups(userGroup, tokenGroups), nil
}
