package aistarslab_setting

import (
	"fmt"
	"strings"
)

func ValidateAIStarsLabSetting(setting *AIStarsLabSetting) error {
	if setting == nil {
		return fmt.Errorf("aistarslab setting is required")
	}
	if err := normalizeAIStarsLabSetting(setting); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(setting.Profiles))
	for _, profile := range setting.Profiles {
		if profile.Model == "" {
			return fmt.Errorf("each AIStarsLab model override must include a model name")
		}
		if _, duplicate := seen[profile.Model]; duplicate {
			return fmt.Errorf("AIStarsLab model %q is configured more than once", profile.Model)
		}
		seen[profile.Model] = struct{}{}
		if len(profile.Resolutions) == 0 {
			return fmt.Errorf("AIStarsLab model %q must include at least one resolution", profile.Model)
		}
	}
	return nil
}

func normalizeAIStarsLabSetting(setting *AIStarsLabSetting) error {
	if setting.Profiles == nil {
		setting.Profiles = []ModelOverride{}
		return nil
	}
	normalized := make([]ModelOverride, 0, len(setting.Profiles))
	for _, profile := range setting.Profiles {
		model := strings.TrimSpace(profile.Model)
		resolutions, err := uniqueStringsPreserveOrder(profile.Resolutions)
		if err != nil {
			if model == "" {
				return err
			}
			return fmt.Errorf("AIStarsLab model %q %w", model, err)
		}
		normalized = append(normalized, ModelOverride{
			Model:       model,
			Resolutions: resolutions,
		})
	}
	setting.Profiles = normalized
	return nil
}

func uniqueStringsPreserveOrder(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("has an empty resolution")
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}
