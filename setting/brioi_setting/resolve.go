package brioi_setting

import "strings"

// ResolveProfile matches only the mapped upstream model name. Brioi has no
// default profile because applying the wrong hard limits is unsafe.
func ResolveProfile(upstreamModel string) (Profile, bool) {
	upstreamModel = strings.TrimSpace(upstreamModel)
	if upstreamModel == "" {
		return Profile{}, false
	}
	setting := GetBrioiSetting()
	if setting == nil {
		return Profile{}, false
	}
	for _, profile := range setting.Profiles {
		if profile.Enabled && strings.TrimSpace(profile.Model) == upstreamModel {
			return cloneProfile(profile), true
		}
	}
	return Profile{}, false
}

func FindGenerationMode(profile Profile, value string) (GenerationModeSetting, bool) {
	value = strings.TrimSpace(value)
	for _, mode := range profile.GenerationModes {
		if mode.Enabled && mode.Value == value {
			return mode, true
		}
	}
	return GenerationModeSetting{}, false
}

func cloneProfile(profile Profile) Profile {
	profile.Durations = append([]int(nil), profile.Durations...)
	profile.Resolutions = append([]string(nil), profile.Resolutions...)
	profile.AspectRatios = append([]string(nil), profile.AspectRatios...)
	profile.GenerationModes = append([]GenerationModeSetting(nil), profile.GenerationModes...)
	return profile
}
