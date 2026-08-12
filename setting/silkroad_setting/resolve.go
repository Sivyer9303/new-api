package silkroad_setting

import "strings"

type ProfileMatchKind string

const (
	ProfileMatchExact   ProfileMatchKind = "exact"
	ProfileMatchPrefix  ProfileMatchKind = "prefix"
	ProfileMatchDefault ProfileMatchKind = "default"
)

type ProfileResolution struct {
	Profile   Profile
	MatchKind ProfileMatchKind
	Rule      string
}

func MatchProfile(modelName string) (*Profile, bool) {
	resolution, ok := ResolveProfile(modelName)
	if !ok {
		return nil, false
	}
	return &resolution.Profile, true
}

func ResolveProfile(modelName string) (ProfileResolution, bool) {
	return resolveProfileFromSetting(&silkRoadSetting, strings.TrimSpace(modelName))
}

func resolveProfileFromSetting(setting *SilkRoadSetting, modelName string) (ProfileResolution, bool) {
	if setting == nil || len(setting.Profiles) == 0 {
		return ProfileResolution{}, false
	}
	for i := range setting.Profiles {
		for _, exact := range setting.Profiles[i].ExactModels {
			if modelName == strings.TrimSpace(exact) {
				return buildProfileResolution(setting, i, ProfileMatchExact, modelName), true
			}
		}
	}

	matchIndex := -1
	matchPrefix := ""
	for i := range setting.Profiles {
		for _, rawPrefix := range setting.Profiles[i].ModelPrefixes {
			prefix := strings.TrimSpace(rawPrefix)
			if prefix != "" && strings.HasPrefix(modelName, prefix) && len(prefix) > len(matchPrefix) {
				matchIndex = i
				matchPrefix = prefix
			}
		}
	}
	if matchIndex >= 0 {
		return buildProfileResolution(setting, matchIndex, ProfileMatchPrefix, matchPrefix), true
	}

	defaultIndex := effectiveDefaultProfileIndex(setting)
	return buildProfileResolution(
		setting,
		defaultIndex,
		ProfileMatchDefault,
		setting.Profiles[defaultIndex].ID,
	), true
}

func effectiveDefaultProfileIndex(setting *SilkRoadSetting) int {
	for i := range setting.Profiles {
		if setting.Profiles[i].ID == setting.DefaultProfileID {
			return i
		}
	}
	return 0
}

func buildProfileResolution(
	setting *SilkRoadSetting,
	profileIndex int,
	matchKind ProfileMatchKind,
	rule string,
) ProfileResolution {
	profile := setting.Profiles[profileIndex]
	profile.ExactModels = append([]string(nil), profile.ExactModels...)
	profile.ModelPrefixes = append([]string(nil), profile.ModelPrefixes...)
	if profile.Durations == nil {
		profile.Durations = append([]OptionItem(nil), setting.Common.Durations...)
	} else {
		profile.Durations = append([]OptionItem(nil), profile.Durations...)
	}
	if profile.AspectRatios == nil {
		profile.AspectRatios = append([]OptionItem(nil), setting.Common.AspectRatios...)
	} else {
		profile.AspectRatios = append([]OptionItem(nil), profile.AspectRatios...)
	}
	return ProfileResolution{Profile: profile, MatchKind: matchKind, Rule: rule}
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
