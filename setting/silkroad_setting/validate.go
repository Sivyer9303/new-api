package silkroad_setting

import (
	"errors"
	"fmt"
	"strings"
)

// ValidateSilkRoadSetting checks silkroad_setting invariants before save/use.
func ValidateSilkRoadSetting(s *SilkRoadSetting) error {
	if s == nil {
		return errors.New("silkroad setting is nil")
	}
	if len(s.Profiles) == 0 {
		return errors.New("profiles cannot be empty")
	}
	seenIDs := make(map[string]struct{}, len(s.Profiles))
	for i := range s.Profiles {
		if err := validateProfile(&s.Profiles[i], i); err != nil {
			return err
		}
		id := s.Profiles[i].ID
		if _, dup := seenIDs[id]; dup {
			return fmt.Errorf("profile[%d]: duplicate id %q", i, id)
		}
		seenIDs[id] = struct{}{}
	}
	if err := validateStorage(&s.Storage); err != nil {
		return err
	}
	return nil
}

func validateProfile(p *Profile, idx int) error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("profile[%d]: id is required", idx)
	}
	if strings.TrimSpace(p.Label) == "" {
		return fmt.Errorf("profile[%d]: label is required", idx)
	}
	if len(p.ModelPrefixes) == 0 {
		return fmt.Errorf("profile[%d]: model_prefixes cannot be empty", idx)
	}
	for j, prefix := range p.ModelPrefixes {
		if strings.TrimSpace(prefix) == "" {
			return fmt.Errorf("profile[%d]: model_prefixes[%d] is empty", idx, j)
		}
	}
	if err := validateOptionList(p.Durations, fmt.Sprintf("profile[%d].durations", idx)); err != nil {
		return err
	}
	if err := validateOptionList(p.AspectRatios, fmt.Sprintf("profile[%d].aspect_ratios", idx)); err != nil {
		return err
	}
	if err := validateGenerationTypes(p.GenerationTypes, fmt.Sprintf("profile[%d].generation_types", idx)); err != nil {
		return err
	}
	if err := validateOptionItemsOptional(p.ExtraOptions, fmt.Sprintf("profile[%d].extra_options", idx)); err != nil {
		return err
	}
	return nil
}

func validateOptionList(items []OptionItem, path string) error {
	if err := validateOptionItemsOptional(items, path); err != nil {
		return err
	}
	enabled := 0
	for _, item := range items {
		if item.Enabled {
			enabled++
		}
	}
	if enabled == 0 {
		return fmt.Errorf("%s: at least one enabled item is required", path)
	}
	return nil
}

func validateOptionItemsOptional(items []OptionItem, path string) error {
	seen := make(map[string]struct{})
	for i, item := range items {
		if !item.Enabled {
			continue
		}
		if strings.TrimSpace(item.Label) == "" {
			return fmt.Errorf("%s[%d]: label is required when enabled", path, i)
		}
		if strings.TrimSpace(item.Value) == "" {
			return fmt.Errorf("%s[%d]: value is required when enabled", path, i)
		}
		if strings.TrimSpace(item.UpstreamKey) == "" {
			return fmt.Errorf("%s[%d]: upstream_key is required when enabled", path, i)
		}
		if _, dup := seen[item.Value]; dup {
			return fmt.Errorf("%s[%d]: duplicate value %q", path, i, item.Value)
		}
		seen[item.Value] = struct{}{}
	}
	return nil
}

func validateGenerationTypes(items []GenerationType, path string) error {
	enabled := 0
	seen := make(map[string]struct{})
	for i, item := range items {
		if !item.Enabled {
			continue
		}
		enabled++
		if strings.TrimSpace(item.Label) == "" {
			return fmt.Errorf("%s[%d]: label is required when enabled", path, i)
		}
		if strings.TrimSpace(item.Value) == "" {
			return fmt.Errorf("%s[%d]: value is required when enabled", path, i)
		}
		if _, dup := seen[item.Value]; dup {
			return fmt.Errorf("%s[%d]: duplicate value %q", path, i, item.Value)
		}
		seen[item.Value] = struct{}{}
		if item.MediaRequirements.ImagesMin < 0 || item.MediaRequirements.ImagesMax < 0 {
			return fmt.Errorf("%s[%d]: media_requirements images bounds must be >= 0", path, i)
		}
		if item.MediaRequirements.ImagesMin > item.MediaRequirements.ImagesMax {
			return fmt.Errorf("%s[%d]: media_requirements images_min must be <= images_max", path, i)
		}
		for j, us := range item.UpstreamSets {
			if strings.TrimSpace(us.UpstreamKey) == "" {
				return fmt.Errorf("%s[%d].upstream_sets[%d]: upstream_key is required", path, i, j)
			}
		}
	}
	if enabled == 0 {
		return fmt.Errorf("%s: at least one enabled item is required", path)
	}
	return nil
}

func validateStorage(s *StorageSetting) error {
	if !s.Enabled {
		return nil
	}
	if s.Driver != "local" {
		return fmt.Errorf("storage.driver must be \"local\" when enabled, got %q", s.Driver)
	}
	if strings.TrimSpace(s.LocalDir) == "" {
		return errors.New("storage.local_dir is required when enabled")
	}
	if s.RetentionDays < 1 {
		return errors.New("storage.retention_days must be >= 1 when enabled")
	}
	if s.MaxRetry < 1 {
		return errors.New("storage.max_retry must be >= 1 when enabled")
	}
	return nil
}
