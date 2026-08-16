package compatvideo_setting

import (
	"fmt"
	"strings"
)

// ValidateCompatVideoSetting verifies administrator overrides are coherent with
// the built-in profiles before they are persisted. Empty optional fields are
// allowed — they mean "keep the built-in default" — so only explicit values are
// checked here.
func ValidateCompatVideoSetting(setting *CompatVideoSetting) error {
	if setting == nil {
		return fmt.Errorf("compat_video setting is required")
	}
	seenIDs := make(map[string]struct{}, len(setting.Profiles))
	for _, profile := range setting.Profiles {
		id := strings.TrimSpace(profile.ID)
		if id == "" {
			return fmt.Errorf("each compat_video profile override must reference a built-in profile ID")
		}
		if _, duplicate := seenIDs[id]; duplicate {
			return fmt.Errorf("compat_video profile %q is configured more than once", id)
		}
		seenIDs[id] = struct{}{}

		builtIn, ok := builtInProfileByID(id)
		if !ok {
			return fmt.Errorf("unknown compat_video profile %q", id)
		}

		for _, duration := range profile.Durations {
			if duration <= 0 {
				return fmt.Errorf("compat_video profile %q has an invalid duration %d", id, duration)
			}
		}
		for _, resolution := range profile.Resolutions {
			if strings.TrimSpace(resolution) == "" {
				return fmt.Errorf("compat_video profile %q has an empty resolution", id)
			}
		}
		for _, aspect := range profile.AspectRatios {
			if strings.TrimSpace(aspect) == "" {
				return fmt.Errorf("compat_video profile %q has an empty aspect ratio", id)
			}
		}

		supportedModes := make(map[string]struct{}, len(builtIn.GenerationModes))
		for _, mode := range builtIn.GenerationModes {
			supportedModes[mode.Value] = struct{}{}
		}
		for _, mode := range profile.GenerationModes {
			if _, ok := supportedModes[mode.Value]; !ok {
				return fmt.Errorf(
					"generation mode %q is not supported by compat_video profile %q",
					mode.Value,
					id,
				)
			}
			if mode.ImagesMax < 0 {
				return fmt.Errorf(
					"generation mode %q of profile %q has a negative images_max",
					mode.Value,
					id,
				)
			}
		}

		switch profile.Dialect {
		case "", DialectNewAPIGenerations, DialectOpenAIVideos:
		default:
			return fmt.Errorf("compat_video profile %q has an unsupported dialect %q", id, profile.Dialect)
		}
	}
	return nil
}

func builtInProfileByID(id string) (Profile, bool) {
	for _, profile := range builtInProfiles {
		if profile.ID == id {
			return profile, true
		}
	}
	return Profile{}, false
}
