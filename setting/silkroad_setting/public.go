package silkroad_setting

import "sort"

// PublicOption is a user-facing selectable option (no internal-only fields beyond what the tool needs).
type PublicOption struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	UpstreamKey string `json:"upstream_key"`
	Sort        int    `json:"sort"`
}

// PublicGenerationType is a user-facing generation mode.
type PublicGenerationType struct {
	Label           string            `json:"label"`
	Value           string            `json:"value"`
	Sort            int               `json:"sort"`
	RequireRefModel bool              `json:"require_ref_model"`
	ImagesMin       int               `json:"images_min"`
	ImagesMax       int               `json:"images_max"`
}

// PublicProfile is a sanitized profile for the logged-in video tool UI.
type PublicProfile struct {
	ID              string                `json:"id"`
	Label           string                `json:"label"`
	ModelPrefixes   []string              `json:"model_prefixes"`
	Durations       []PublicOption        `json:"durations"`
	AspectRatios    []PublicOption        `json:"aspect_ratios"`
	GenerationTypes []PublicGenerationType `json:"generation_types"`
}

// PublicVideoToolConfig is returned to logged-in users for the Seedance-style tool page.
type PublicVideoToolConfig struct {
	Enabled  bool           `json:"enabled"`
	Profiles []PublicProfile `json:"profiles"`
}

// GetPublicVideoToolConfig returns enabled profiles/options only (no storage secrets).
func GetPublicVideoToolConfig() PublicVideoToolConfig {
	s := GetSilkRoadSetting()
	out := PublicVideoToolConfig{Enabled: true, Profiles: make([]PublicProfile, 0)}
	if s == nil || len(s.Profiles) == 0 {
		out.Enabled = false
		return out
	}
	for _, p := range s.Profiles {
		pub := PublicProfile{
			ID:            p.ID,
			Label:         p.Label,
			ModelPrefixes: append([]string(nil), p.ModelPrefixes...),
			Durations:     publicOptions(p.Durations),
			AspectRatios:  publicOptions(p.AspectRatios),
			GenerationTypes: publicGenerationTypes(p.GenerationTypes),
		}
		if len(pub.Durations) == 0 || len(pub.AspectRatios) == 0 || len(pub.GenerationTypes) == 0 {
			continue
		}
		out.Profiles = append(out.Profiles, pub)
	}
	if len(out.Profiles) == 0 {
		out.Enabled = false
	}
	return out
}

func publicOptions(items []OptionItem) []PublicOption {
	out := make([]PublicOption, 0, len(items))
	for _, it := range items {
		if !it.Enabled {
			continue
		}
		out = append(out, PublicOption{
			Label:       it.Label,
			Value:       it.Value,
			UpstreamKey: it.UpstreamKey,
			Sort:        it.Sort,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sort < out[j].Sort })
	return out
}

func publicGenerationTypes(items []GenerationType) []PublicGenerationType {
	out := make([]PublicGenerationType, 0, len(items))
	for _, it := range items {
		if !it.Enabled {
			continue
		}
		out = append(out, PublicGenerationType{
			Label:           it.Label,
			Value:           it.Value,
			Sort:            it.Sort,
			RequireRefModel: it.RequireRefModel,
			ImagesMin:       it.MediaRequirements.ImagesMin,
			ImagesMax:       it.MediaRequirements.ImagesMax,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sort < out[j].Sort })
	return out
}
