package brioi_setting

import (
	"errors"
	"fmt"
	"strings"
)

func ValidateBrioiSetting(setting *BrioiSetting) error {
	if setting == nil {
		return errors.New("brioi setting is nil")
	}
	setting.VideoToolGroups = NormalizeVideoToolGroups(setting.VideoToolGroups)
	if len(setting.Profiles) != 3 {
		return fmt.Errorf("profiles must contain exactly the three supported Brioi models")
	}

	seenModels := make(map[string]struct{}, len(setting.Profiles))
	for index := range setting.Profiles {
		profile := &setting.Profiles[index]
		profile.Model = strings.TrimSpace(profile.Model)
		if profile.Model == "" {
			return fmt.Errorf("profiles[%d].model is required", index)
		}
		if _, duplicate := seenModels[profile.Model]; duplicate {
			return fmt.Errorf("profiles[%d].model %q is duplicated", index, profile.Model)
		}
		seenModels[profile.Model] = struct{}{}

		hardProfile, ok := hardProfile(profile.Model)
		if !ok {
			return fmt.Errorf("profiles[%d].model %q is not supported by Brioi", index, profile.Model)
		}
		if strings.TrimSpace(profile.Label) == "" {
			return fmt.Errorf("profiles[%d].label is required", index)
		}
		if err := validateIntegerSubset(
			profile.Durations,
			hardProfile.Durations,
			fmt.Sprintf("profiles[%d].durations", index),
			profile.Enabled,
		); err != nil {
			return err
		}
		if err := validateStringSubset(
			profile.Resolutions,
			hardProfile.Resolutions,
			fmt.Sprintf("profiles[%d].resolutions", index),
			profile.Enabled,
		); err != nil {
			return err
		}
		if err := validateStringSubset(
			profile.AspectRatios,
			hardProfile.AspectRatios,
			fmt.Sprintf("profiles[%d].aspect_ratios", index),
			profile.Enabled,
		); err != nil {
			return err
		}
		profile.GenerationModes = softMergeGenerationModes(
			profile.GenerationModes,
			hardProfile.GenerationModes,
		)
		if err := validateGenerationModes(
			profile.GenerationModes,
			hardProfile.GenerationModes,
			fmt.Sprintf("profiles[%d].generation_modes", index),
			profile.Enabled,
		); err != nil {
			return err
		}
	}

	for _, model := range []string{ModelSeedance20Fast, ModelSeedance20, ModelSeedance25} {
		if _, ok := seenModels[model]; !ok {
			return fmt.Errorf("profiles is missing required Brioi model %q", model)
		}
	}
	return nil
}

func NormalizeVideoToolGroups(groups []string) []string {
	if len(groups) == 0 {
		return []string{}
	}
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

func hardProfile(model string) (Profile, bool) {
	switch model {
	case ModelSeedance20Fast, ModelSeedance20, ModelSeedance25:
		return defaultProfile(model), true
	default:
		return Profile{}, false
	}
}

func validateIntegerSubset(values, allowed []int, path string, requireNonEmpty bool) error {
	if requireNonEmpty && len(values) == 0 {
		return fmt.Errorf("%s must contain at least one supported value", path)
	}
	allowedSet := make(map[int]struct{}, len(allowed))
	for _, value := range allowed {
		allowedSet[value] = struct{}{}
	}
	seen := make(map[int]struct{}, len(values))
	for index, value := range values {
		if _, ok := allowedSet[value]; !ok {
			return fmt.Errorf("%s[%d] value %d is outside the Brioi hard capabilities", path, index, value)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("%s[%d] value %d is duplicated", path, index, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateStringSubset(values, allowed []string, path string, requireNonEmpty bool) error {
	if requireNonEmpty && len(values) == 0 {
		return fmt.Errorf("%s must contain at least one supported value", path)
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, value := range allowed {
		allowedSet[value] = struct{}{}
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if _, ok := allowedSet[value]; !ok {
			return fmt.Errorf("%s[%d] value %q is outside the Brioi hard capabilities", path, index, value)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("%s[%d] value %q is duplicated", path, index, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

// softMergeGenerationModes appends newly introduced hard modes (e.g. reference_videos)
// so stored admin configs remain valid after capability upgrades. Empty slices stay
// empty so explicit clearing still fails validation.
func softMergeGenerationModes(
	modes []GenerationModeSetting,
	hardModes []GenerationModeSetting,
) []GenerationModeSetting {
	if len(modes) == 0 {
		return modes
	}
	seen := make(map[string]struct{}, len(modes))
	for _, mode := range modes {
		seen[strings.TrimSpace(mode.Value)] = struct{}{}
	}
	merged := append([]GenerationModeSetting(nil), modes...)
	for _, hard := range hardModes {
		if _, ok := seen[hard.Value]; ok {
			continue
		}
		merged = append(merged, hard)
	}
	return merged
}

func validateGenerationModes(
	modes []GenerationModeSetting,
	hardModes []GenerationModeSetting,
	path string,
	profileEnabled bool,
) error {
	if len(modes) == 0 {
		return fmt.Errorf("%s must contain every supported Brioi generation mode exactly once", path)
	}
	hardByValue := make(map[string]GenerationModeSetting, len(hardModes))
	for _, mode := range hardModes {
		hardByValue[mode.Value] = mode
	}

	seen := make(map[string]struct{}, len(modes))
	enabledCount := 0
	for index, mode := range modes {
		hardMode, ok := hardByValue[mode.Value]
		if !ok {
			return fmt.Errorf("%s[%d].value %q is not supported by Brioi", path, index, mode.Value)
		}
		if _, duplicate := seen[mode.Value]; duplicate {
			return fmt.Errorf("%s[%d].value %q is duplicated", path, index, mode.Value)
		}
		seen[mode.Value] = struct{}{}
		if mode.Enabled {
			enabledCount++
		}

		switch mode.Value {
		case GenerationMultiImage:
			if mode.ImagesMax < 2 || mode.ImagesMax > hardMode.ImagesMax {
				return fmt.Errorf(
					"%s[%d].images_max must be between 2 and %d",
					path,
					index,
					hardMode.ImagesMax,
				)
			}
		case GenerationReferenceVideos:
			// Companion images are optional; 0 disables them.
			if mode.ImagesMax < 0 || mode.ImagesMax > hardMode.ImagesMax {
				return fmt.Errorf(
					"%s[%d].images_max must be between 0 and %d",
					path,
					index,
					hardMode.ImagesMax,
				)
			}
		default:
			if mode.ImagesMax != hardMode.ImagesMax {
				return fmt.Errorf(
					"%s[%d].images_max must equal the Brioi hard value %d",
					path,
					index,
					hardMode.ImagesMax,
				)
			}
		}
	}
	for _, hard := range hardModes {
		if _, ok := seen[hard.Value]; !ok {
			return fmt.Errorf("%s must contain every supported Brioi generation mode exactly once", path)
		}
	}
	if len(modes) != len(hardModes) {
		return fmt.Errorf("%s must contain every supported Brioi generation mode exactly once", path)
	}
	if profileEnabled && enabledCount == 0 {
		return fmt.Errorf("%s must contain at least one enabled mode", path)
	}
	return nil
}
