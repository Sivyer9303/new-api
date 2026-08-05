package silkroad_setting

import "strings"

// MatchProfile finds the first profile whose model_prefixes match modelName.
func MatchProfile(modelName string) (*Profile, bool) {
	if modelName == "" {
		return nil, false
	}
	for i := range silkRoadSetting.Profiles {
		p := &silkRoadSetting.Profiles[i]
		for _, prefix := range p.ModelPrefixes {
			if prefix != "" && strings.HasPrefix(modelName, prefix) {
				return p, true
			}
		}
	}
	return nil, false
}

// FindEnabledOption returns the enabled option matching value.
func FindEnabledOption(items []OptionItem, value string) (*OptionItem, bool) {
	for i := range items {
		if !items[i].Enabled {
			continue
		}
		if items[i].Value == value {
			return &items[i], true
		}
	}
	return nil, false
}

// FindGenerationType returns the enabled generation type matching value.
func FindGenerationType(p *Profile, value string) (*GenerationType, bool) {
	if p == nil {
		return nil, false
	}
	for i := range p.GenerationTypes {
		gt := &p.GenerationTypes[i]
		if !gt.Enabled {
			continue
		}
		if gt.Value == value {
			return gt, true
		}
	}
	return nil, false
}
