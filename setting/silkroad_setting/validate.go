package silkroad_setting

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// ValidateSilkRoadSetting checks the complete legacy module. Section-specific
// option saves use the narrower provider/storage validators below.
func ValidateSilkRoadSetting(s *SilkRoadSetting) error {
	if err := ValidateSilkRoadProviderSetting(s); err != nil {
		return err
	}
	return ValidateSilkRoadStorageSetting(&s.Storage)
}

// ValidateSilkRoadProviderSetting validates only provider capabilities and
// groups, without depending on the deprecated storage section.
func ValidateSilkRoadProviderSetting(s *SilkRoadSetting) error {
	if s == nil {
		return errors.New("silkroad setting is nil")
	}
	if len(s.Profiles) == 0 {
		return errors.New("profiles cannot be empty")
	}
	if err := validateOptionList(s.Common.Durations, "common.durations"); err != nil {
		return err
	}
	if err := validateDurationValues(s.Common.Durations, "common.durations"); err != nil {
		return err
	}
	if err := validateOptionList(s.Common.AspectRatios, "common.aspect_ratios"); err != nil {
		return err
	}
	if err := validateAspectRatioValues(s.Common.AspectRatios, "common.aspect_ratios"); err != nil {
		return err
	}
	seenIDs := make(map[string]struct{}, len(s.Profiles))
	seenExactModels := make(map[string]string)
	seenPrefixes := make(map[string]string)
	for i := range s.Profiles {
		if err := validateProfile(&s.Profiles[i], i); err != nil {
			return err
		}
		id := s.Profiles[i].ID
		if _, dup := seenIDs[id]; dup {
			return fmt.Errorf("profile[%d]: duplicate id %q", i, id)
		}
		seenIDs[id] = struct{}{}
		for _, exact := range s.Profiles[i].ExactModels {
			modelName := strings.TrimSpace(exact)
			if owner, duplicate := seenExactModels[modelName]; duplicate {
				return fmt.Errorf(
					"profile[%d]: duplicate exact model %q already used by profile %q",
					i,
					modelName,
					owner,
				)
			}
			seenExactModels[modelName] = id
		}
		for _, rawPrefix := range s.Profiles[i].ModelPrefixes {
			prefix := strings.TrimSpace(rawPrefix)
			if owner, duplicate := seenPrefixes[prefix]; duplicate {
				return fmt.Errorf(
					"profile[%d]: duplicate model prefix %q already used by profile %q",
					i,
					prefix,
					owner,
				)
			}
			seenPrefixes[prefix] = id
		}
	}
	if strings.TrimSpace(s.DefaultProfileID) == "" {
		return errors.New("default_profile_id is required")
	}
	if _, ok := seenIDs[s.DefaultProfileID]; !ok {
		return fmt.Errorf("default_profile_id %q does not reference an existing profile", s.DefaultProfileID)
	}
	s.VideoToolGroups = NormalizeVideoToolGroups(s.VideoToolGroups)
	return nil
}

func ValidateSilkRoadStorageSetting(s *StorageSetting) error {
	if s == nil {
		return errors.New("silkroad storage setting is nil")
	}
	return validateStorage(s)
}

// NormalizeVideoToolGroups trims, drops empties, and deduplicates group names
// while preserving first-seen order. Empty input yields an empty slice (no keys allowed).
func NormalizeVideoToolGroups(groups []string) []string {
	if len(groups) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(groups))
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		name := strings.TrimSpace(g)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func validateProfile(p *Profile, idx int) error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("profile[%d]: id is required", idx)
	}
	if strings.TrimSpace(p.Label) == "" {
		return fmt.Errorf("profile[%d]: label is required", idx)
	}
	for j, exact := range p.ExactModels {
		if strings.TrimSpace(exact) == "" {
			return fmt.Errorf("profile[%d]: exact_models[%d] is empty", idx, j)
		}
	}
	for j, prefix := range p.ModelPrefixes {
		if strings.TrimSpace(prefix) == "" {
			return fmt.Errorf("profile[%d]: model_prefixes[%d] is empty", idx, j)
		}
	}
	if p.Durations != nil {
		path := fmt.Sprintf("profile[%d].durations", idx)
		if err := validateOptionList(p.Durations, path); err != nil {
			return err
		}
		if err := validateDurationValues(p.Durations, path); err != nil {
			return err
		}
	}
	if p.AspectRatios != nil {
		path := fmt.Sprintf("profile[%d].aspect_ratios", idx)
		if err := validateOptionList(p.AspectRatios, path); err != nil {
			return err
		}
		if err := validateAspectRatioValues(p.AspectRatios, path); err != nil {
			return err
		}
	}
	return nil
}

func validateAspectRatioValues(items []OptionItem, path string) error {
	allowed := make(map[string]struct{})
	for _, item := range defaultAspectRatios() {
		allowed[item.Value] = struct{}{}
	}
	for i, item := range items {
		if !item.Enabled {
			continue
		}
		if _, ok := allowed[item.Value]; !ok {
			return fmt.Errorf("%s[%d]: aspect ratio %q is outside SilkRoad hard capabilities", path, i, item.Value)
		}
	}
	return nil
}

func validateDurationValues(items []OptionItem, path string) error {
	for i, item := range items {
		if !item.Enabled {
			continue
		}
		seconds, err := strconv.Atoi(item.Value)
		if err != nil || seconds < 1 || seconds > relaycommon.MaxTaskDurationSeconds {
			return fmt.Errorf(
				"%s[%d]: duration must be between 1 and %d seconds",
				path,
				i,
				relaycommon.MaxTaskDurationSeconds,
			)
		}
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
	if s.RetentionDays != 7 {
		return errors.New("storage.retention_days is fixed at 7")
	}
	if s.MaxRetry < 1 {
		return errors.New("storage.max_retry must be >= 1 when enabled")
	}
	if strings.TrimSpace(s.IngestNodeName) == "" {
		return errors.New("storage.ingest_node_name is required when enabled")
	}
	if strings.TrimSpace(s.PublicDownloadBaseURL) == "" {
		return errors.New("storage.public_download_base_url is required when enabled")
	}
	return nil
}
