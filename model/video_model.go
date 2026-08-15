package model

import (
	"errors"
	"slices"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

const videoGenerationRequestPath = "/v1/video/generations"

type EligibleVideoModelRoute struct {
	Model          string
	UpstreamModel  string
	ChannelID      int
	ChannelType    int
	Priority       int64
	Weight         uint
	InvalidMapping bool
}

// GetEligibleVideoModelRoutes returns enabled group/model/channel bindings for
// video generation channel types. Empty channelTypes uses every video type.
func GetEligibleVideoModelRoutes(
	groups []string,
	channelTypes ...int,
) ([]EligibleVideoModelRoute, error) {
	groups = normalizeVideoModelGroups(groups)
	if len(groups) == 0 {
		return []EligibleVideoModelRoute{}, nil
	}
	if len(channelTypes) == 0 {
		channelTypes = constant.VideoGenerationChannelTypes()
	}
	allowedTypes := make([]interface{}, 0, len(channelTypes))
	for _, channelType := range channelTypes {
		if channelType <= 0 {
			continue
		}
		if !common.ChannelTypeSupportsRequestPath(channelType, videoGenerationRequestPath) ||
			!slices.Contains(
				common.GetEndpointTypesByChannelType(channelType, ""),
				constant.EndpointTypeOpenAIVideo,
			) {
			continue
		}
		allowedTypes = append(allowedTypes, channelType)
	}
	if len(allowedTypes) == 0 {
		return []EligibleVideoModelRoute{}, nil
	}
	groupValues := make([]interface{}, len(groups))
	for index, group := range groups {
		groupValues[index] = group
	}

	type routeRow struct {
		Model        string
		ChannelID    int
		ChannelType  int
		Priority     *int64
		Weight       uint
		ModelMapping *string
	}
	groupCol := commonGroupCol
	if groupCol == "" {
		groupCol = "`group`"
	}
	var rows []routeRow
	err := DB.Table("abilities").
		Select(
			"abilities.model AS model, abilities.channel_id AS channel_id, "+
				"channels.type AS channel_type, abilities.priority AS priority, "+
				"abilities.weight AS weight, channels.model_mapping AS model_mapping",
		).
		Joins("JOIN channels ON channels.id = abilities.channel_id").
		Where("abilities."+groupCol+" IN ?", groupValues).
		Where("abilities.enabled = ?", true).
		Where("channels.status = ?", common.ChannelStatusEnabled).
		Where("channels.type IN ?", allowedTypes).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	routes := make([]EligibleVideoModelRoute, 0, len(rows))
	for _, row := range rows {
		modelName := strings.TrimSpace(row.Model)
		if modelName == "" {
			continue
		}
		priority := int64(0)
		if row.Priority != nil {
			priority = *row.Priority
		}
		rawMapping := ""
		if row.ModelMapping != nil {
			rawMapping = *row.ModelMapping
		}
		upstreamModel, mappingErr := resolveMappedModelName(modelName, rawMapping)
		if mappingErr != nil {
			routes = append(routes, EligibleVideoModelRoute{
				Model:          modelName,
				ChannelID:      row.ChannelID,
				ChannelType:    row.ChannelType,
				Priority:       priority,
				Weight:         row.Weight,
				InvalidMapping: true,
			})
			continue
		}
		routes = append(routes, EligibleVideoModelRoute{
			Model:         modelName,
			UpstreamModel: upstreamModel,
			ChannelID:     row.ChannelID,
			ChannelType:   row.ChannelType,
			Priority:      priority,
			Weight:        row.Weight,
		})
	}
	sort.SliceStable(routes, func(i, j int) bool {
		if routes[i].Model != routes[j].Model {
			return routes[i].Model < routes[j].Model
		}
		return videoRouteHitLess(routes[j], routes[i])
	})
	return routes, nil
}

func PickVideoModelHit(routes []EligibleVideoModelRoute) (EligibleVideoModelRoute, bool) {
	var hit EligibleVideoModelRoute
	found := false
	for _, route := range routes {
		if route.InvalidMapping || route.ChannelType <= 0 {
			continue
		}
		if !found || videoRouteHitLess(hit, route) {
			hit = route
			found = true
		}
	}
	return hit, found
}

func PickVideoModelHitByName(
	routes []EligibleVideoModelRoute,
	modelName string,
) (EligibleVideoModelRoute, bool) {
	modelName = strings.TrimSpace(modelName)
	filtered := make([]EligibleVideoModelRoute, 0, len(routes))
	for _, route := range routes {
		if route.Model == modelName {
			filtered = append(filtered, route)
		}
	}
	return PickVideoModelHit(filtered)
}

func ListEnabledVideoToolGroups() ([]string, error) {
	if DB == nil {
		return []string{}, nil
	}
	channelTypes := constant.VideoGenerationChannelTypes()
	typeValues := make([]interface{}, len(channelTypes))
	for index, channelType := range channelTypes {
		typeValues[index] = channelType
	}
	groupCol := commonGroupCol
	if groupCol == "" {
		groupCol = "`group`"
	}
	var groups []string
	err := DB.Table("abilities").
		Select("DISTINCT abilities."+groupCol+" AS video_group").
		Joins("JOIN channels ON channels.id = abilities.channel_id").
		Where("abilities.enabled = ?", true).
		Where("channels.status = ?", common.ChannelStatusEnabled).
		Where("channels.type IN ?", typeValues).
		Pluck("video_group", &groups).Error
	if err != nil {
		return nil, err
	}
	return normalizeVideoModelGroups(groups), nil
}

func videoRouteHitLess(left, right EligibleVideoModelRoute) bool {
	if left.Priority != right.Priority {
		return left.Priority < right.Priority
	}
	if left.Weight != right.Weight {
		return left.Weight < right.Weight
	}
	return left.ChannelID > right.ChannelID
}

func resolveMappedModelName(modelName, rawMapping string) (string, error) {
	modelName = strings.TrimSpace(modelName)
	rawMapping = strings.TrimSpace(rawMapping)
	if rawMapping == "" || rawMapping == "{}" {
		return modelName, nil
	}

	mapping := make(map[string]string)
	if err := common.UnmarshalJsonStr(rawMapping, &mapping); err != nil {
		return "", err
	}
	current := modelName
	visited := map[string]struct{}{current: {}}
	for {
		next, ok := mapping[current]
		next = strings.TrimSpace(next)
		if !ok || next == "" || next == current {
			return current, nil
		}
		if _, cycle := visited[next]; cycle {
			return "", errors.New("model mapping contains cycle")
		}
		visited[next] = struct{}{}
		current = next
	}
}

func normalizeVideoModelGroups(groups []string) []string {
	seen := make(map[string]struct{}, len(groups))
	normalized := make([]string, 0, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		if _, duplicate := seen[group]; duplicate {
			continue
		}
		seen[group] = struct{}{}
		normalized = append(normalized, group)
	}
	return normalized
}
