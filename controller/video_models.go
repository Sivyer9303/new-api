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

	owner, ownedGroups, err := setting.ResolveVideoProviderForGroups(candidateGroups)
	if err != nil {
		response.Reason = "ambiguous_video_provider"
		common.ApiSuccess(c, response)
		return
	}
	response.ResolvedGroups = ownedGroups
	if owner.Provider == "" {
		response.Reason = "video_provider_not_configured"
		common.ApiSuccess(c, response)
		return
	}
	response.Provider = owner.Provider

	routes, err := model.GetEligibleVideoModelRoutes(ownedGroups, owner.ChannelType)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(routes) == 0 {
		response.Reason = "no_eligible_provider_channel"
		common.ApiSuccess(c, response)
		return
	}

	type modelCapability struct {
		profileModel string
		invalid      bool
	}
	capabilities := make(map[string]modelCapability, len(routes))
	modelOrder := make([]string, 0, len(routes))
	for _, route := range routes {
		capability, seen := capabilities[route.Model]
		if !seen {
			modelOrder = append(modelOrder, route.Model)
		}
		if route.InvalidMapping {
			capability.invalid = true
			capabilities[route.Model] = capability
			continue
		}
		if !setting.VideoProviderSupportsUpstreamModel(owner.Provider, route.UpstreamModel) {
			capability.invalid = true
			capabilities[route.Model] = capability
			continue
		}
		if capability.profileModel != "" && capability.profileModel != route.UpstreamModel {
			capability.invalid = true
		}
		capability.profileModel = route.UpstreamModel
		capabilities[route.Model] = capability
	}

	modelLimits := token.GetModelLimitsMap()
	for _, modelName := range modelOrder {
		capability := capabilities[modelName]
		if capability.invalid || capability.profileModel == "" {
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

		baseModel := buildOpenAIModel(modelName, nil)
		response.Models = append(response.Models, videoToolModel{
			ID:                     baseModel.Id,
			Object:                 baseModel.Object,
			Created:                baseModel.Created,
			OwnedBy:                string(owner.Provider),
			ProfileModel:           capability.profileModel,
			ProviderID:             owner.Provider,
			SupportedEndpointTypes: baseModel.SupportedEndpointTypes,
		})
	}
	if len(response.Models) == 0 {
		response.Reason = "no_eligible_video_models"
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
