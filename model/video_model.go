package model

import (
	"errors"
	"slices"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
)

const videoGenerationRequestPath = "/v1/video/generations"

type EligibleVideoModelRoute struct {
	Model          string
	UpstreamModel  string
	Group          string
	ChannelID      int
	ChannelType    int
	Priority       int64
	Weight         uint
	InvalidMapping bool
}

type VideoRouteDecision struct {
	Group         string
	ChannelID     int
	ChannelType   int
	PublicModel   string
	UpstreamModel string
	RequestPath   string
	Provider      setting.VideoProvider
}

// GetEligibleVideoModelRoutes returns enabled group/model/channel bindings for
// video generation channel types. Empty channelTypes uses every video type.
func GetEligibleVideoModelRoutes(
	groups []string,
	channelTypes ...int,
) ([]EligibleVideoModelRoute, error) {
	return GetEligibleVideoModelRoutesForPath(
		groups,
		videoGenerationRequestPath,
		channelTypes...,
	)
}

func GetEligibleVideoModelRoutesForPath(
	groups []string,
	requestPath string,
	channelTypes ...int,
) ([]EligibleVideoModelRoute, error) {
	groups = normalizeVideoModelGroups(groups)
	if len(groups) == 0 {
		return []EligibleVideoModelRoute{}, nil
	}
	requestPath = strings.TrimSpace(requestPath)
	if requestPath == "" {
		return []EligibleVideoModelRoute{}, errors.New("video request path is required")
	}
	if len(channelTypes) == 0 {
		channelTypes = constant.VideoGenerationChannelTypes()
	}
	allowedTypes := make([]interface{}, 0, len(channelTypes))
	for _, channelType := range channelTypes {
		if channelType <= 0 {
			continue
		}
		if !common.ChannelTypeSupportsRequestPath(channelType, requestPath) ||
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
		VideoGroup   string
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
			"abilities.model AS model, abilities."+groupCol+" AS video_group, "+
				"abilities.channel_id AS channel_id, "+
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
				Group:          row.VideoGroup,
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
			Group:         row.VideoGroup,
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

func ResolveVideoRoute(
	groups []string,
	modelName string,
	requestPath string,
) (VideoRouteDecision, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return VideoRouteDecision{}, errors.New("video model is required")
	}
	requestPath = strings.TrimSpace(requestPath)
	if requestPath == "" {
		return VideoRouteDecision{}, errors.New("video request path is required")
	}

	for _, group := range normalizeVideoModelGroups(groups) {
		routes, err := GetEligibleVideoModelRoutesForPath([]string{group}, requestPath)
		if err != nil {
			return VideoRouteDecision{}, err
		}
		hit, ok := PickVideoModelHitByName(routes, modelName)
		if !ok {
			continue
		}
		provider, _ := setting.VideoProviderFromChannelType(hit.ChannelType)
		return VideoRouteDecision{
			Group:         group,
			ChannelID:     hit.ChannelID,
			ChannelType:   hit.ChannelType,
			PublicModel:   hit.Model,
			UpstreamModel: hit.UpstreamModel,
			RequestPath:   requestPath,
			Provider:      provider,
		}, nil
	}
	return VideoRouteDecision{}, errors.New("no eligible video route")
}

// ChannelMatchesVideoRoute verifies that a channel still provides the exact
// route selected during video model discovery. It is intentionally stricter
// than a channel-type check: two channels of the same provider may map one
// public model to different upstream profiles.
func ChannelMatchesVideoRoute(channel *Channel, decision VideoRouteDecision) bool {
	if channel == nil || channel.Status != common.ChannelStatusEnabled {
		return false
	}
	if channel.Id != decision.ChannelID || channel.Type != decision.ChannelType {
		return false
	}
	if strings.TrimSpace(decision.Group) == "" ||
		!slices.Contains(strings.Split(channel.Group, ","), decision.Group) {
		return false
	}
	if strings.TrimSpace(decision.PublicModel) == "" ||
		strings.TrimSpace(decision.UpstreamModel) == "" ||
		strings.TrimSpace(decision.RequestPath) == "" {
		return false
	}
	if !common.ChannelTypeSupportsRequestPath(channel.Type, decision.RequestPath) {
		return false
	}
	upstreamModel, err := resolveMappedModelName(decision.PublicModel, channel.GetModelMapping())
	return err == nil && upstreamModel == decision.UpstreamModel
}

// GetRandomSatisfiedChannelForVideoRoute selects a retry candidate that keeps
// the resolved provider profile. The initial request uses the decision's exact
// channel; retries may use another channel only when it maps to the same
// upstream model in the same group.
func GetRandomSatisfiedChannelForVideoRoute(
	group string,
	modelName string,
	retry int,
	requestPath string,
	decision VideoRouteDecision,
) (*Channel, error) {
	group = strings.TrimSpace(group)
	modelName = strings.TrimSpace(modelName)
	requestPath = strings.TrimSpace(requestPath)
	if group == "" || group != decision.Group || modelName != decision.PublicModel ||
		requestPath == "" || requestPath != decision.RequestPath {
		return nil, nil
	}

	abilities, err := getChannelTypeAbilities(group, modelName, decision.ChannelType)
	if err != nil {
		return nil, err
	}
	if len(abilities) == 0 {
		return nil, nil
	}

	type candidate struct {
		ability Ability
		channel *Channel
	}
	candidates := make([]candidate, 0, len(abilities))
	for _, ability := range abilities {
		channel, err := GetChannelById(ability.ChannelId, true)
		if err != nil {
			return nil, err
		}
		if channel.Type != decision.ChannelType || channel.Status != common.ChannelStatusEnabled ||
			!common.ChannelTypeSupportsRequestPath(channel.Type, requestPath) {
			continue
		}
		upstreamModel, err := resolveMappedModelName(modelName, channel.GetModelMapping())
		if err != nil || upstreamModel != decision.UpstreamModel {
			continue
		}
		candidates = append(candidates, candidate{ability: ability, channel: channel})
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	priorities := make(map[int64]struct{}, len(candidates))
	for _, candidate := range candidates {
		priority := int64(0)
		if candidate.ability.Priority != nil {
			priority = *candidate.ability.Priority
		}
		priorities[priority] = struct{}{}
	}
	orderedPriorities := make([]int64, 0, len(priorities))
	for priority := range priorities {
		orderedPriorities = append(orderedPriorities, priority)
	}
	sort.Slice(orderedPriorities, func(i, j int) bool {
		return orderedPriorities[i] > orderedPriorities[j]
	})
	if retry < 0 {
		retry = 0
	}
	if retry >= len(orderedPriorities) {
		retry = len(orderedPriorities) - 1
	}
	targetPriority := orderedPriorities[retry]

	eligible := make([]candidate, 0, len(candidates))
	totalWeight := 0
	for _, candidate := range candidates {
		priority := int64(0)
		if candidate.ability.Priority != nil {
			priority = *candidate.ability.Priority
		}
		if priority != targetPriority {
			continue
		}
		eligible = append(eligible, candidate)
		totalWeight += int(candidate.ability.Weight) + 10
	}
	if len(eligible) == 0 {
		return nil, nil
	}

	randomWeight := common.GetRandomInt(totalWeight)
	for _, candidate := range eligible {
		randomWeight -= int(candidate.ability.Weight) + 10
		if randomWeight <= 0 {
			return candidate.channel, nil
		}
	}
	return eligible[len(eligible)-1].channel, nil
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
