package model

import (
	"errors"
	"slices"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm/clause"
)

const videoGenerationRequestPath = "/v1/video/generations"

type EligibleVideoModelRoute struct {
	Model          string
	UpstreamModel  string
	ChannelID      int
	ChannelType    int
	InvalidMapping bool
}

// GetEligibleVideoModelRoutes returns enabled group/model/channel bindings for
// one video provider. The query uses only GORM-compatible joins and predicates.
func GetEligibleVideoModelRoutes(
	groups []string,
	channelType int,
) ([]EligibleVideoModelRoute, error) {
	groups = normalizeVideoModelGroups(groups)
	if len(groups) == 0 || channelType <= 0 {
		return []EligibleVideoModelRoute{}, nil
	}
	if !common.ChannelTypeSupportsRequestPath(channelType, videoGenerationRequestPath) ||
		!slices.Contains(
			common.GetEndpointTypesByChannelType(channelType, ""),
			constant.EndpointTypeOpenAIVideo,
		) {
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
		ModelMapping *string
	}
	var rows []routeRow
	err := DB.Table("abilities").
		Select(
			"abilities.model AS model, abilities.channel_id AS channel_id, "+
				"channels.type AS channel_type, channels.model_mapping AS model_mapping",
		).
		Joins("JOIN channels ON channels.id = abilities.channel_id").
		Where(clause.IN{
			Column: clause.Column{Table: "abilities", Name: "group"},
			Values: groupValues,
		}).
		Where("abilities.enabled = ?", true).
		Where("channels.status = ?", common.ChannelStatusEnabled).
		Where("channels.type = ?", channelType).
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
				InvalidMapping: true,
			})
			continue
		}
		routes = append(routes, EligibleVideoModelRoute{
			Model:         modelName,
			UpstreamModel: upstreamModel,
			ChannelID:     row.ChannelID,
			ChannelType:   row.ChannelType,
		})
	}
	sort.SliceStable(routes, func(i, j int) bool {
		if routes[i].Model == routes[j].Model {
			return routes[i].ChannelID < routes[j].ChannelID
		}
		return routes[i].Model < routes[j].Model
	})
	return routes, nil
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
