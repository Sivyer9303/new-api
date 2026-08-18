package compatvideo_setting

import (
	"fmt"
	"sort"
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// ValidateCompatVideoSetting verifies administrator overrides are coherent with
// the built-in profiles before they are persisted. Empty optional fields are
// allowed — they mean "keep the built-in default" — so only explicit values are
// checked here.
func ValidateCompatVideoSetting(setting *CompatVideoSetting) error {
	if setting == nil {
		return fmt.Errorf("compat_video setting is required")
	}
	if err := normalizeCompatVideoSetting(setting); err != nil {
		return err
	}
	seenIDs := make(map[string]struct{}, len(setting.Profiles))
	for _, profile := range setting.Profiles {
		id := profile.ID
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
			if duration <= 0 || duration > relaycommon.MaxTaskDurationSeconds {
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
		if len(profile.GenerationModes) > 0 {
			return fmt.Errorf("generation mode overrides are not supported for compat_video profile %q", id)
		}

		switch profile.Dialect {
		case "", DialectNewAPIGenerations, DialectOpenAIVideos:
		default:
			return fmt.Errorf("compat_video profile %q has an unsupported dialect %q", id, profile.Dialect)
		}
	}
	return nil
}

func normalizeCompatVideoSetting(setting *CompatVideoSetting) error {
	for index := range setting.Profiles {
		profile := &setting.Profiles[index]
		profile.ID = strings.TrimSpace(profile.ID)
		profile.Dialect = Dialect(strings.TrimSpace(string(profile.Dialect)))
		profile.Durations = normalizedDurations(profile.Durations)

		var err error
		profile.Resolutions, err = normalizedStrings(profile.Resolutions, "resolution")
		if err != nil {
			return fmt.Errorf("compat_video profile %q %w", profile.ID, err)
		}
		profile.AspectRatios, err = normalizedStrings(profile.AspectRatios, "aspect ratio")
		if err != nil {
			return fmt.Errorf("compat_video profile %q %w", profile.ID, err)
		}
	}
	return nil
}

func normalizedDurations(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(values))
	normalized := make([]int, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Ints(normalized)
	return normalized
}

func normalizedStrings(values []string, field string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("has an empty %s", field)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func builtInProfileByID(id string) (Profile, bool) {
	for _, profile := range builtInProfiles {
		if profile.ID == id {
			return profile, true
		}
	}
	return Profile{}, false
}
