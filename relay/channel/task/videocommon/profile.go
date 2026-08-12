package videocommon

import (
	"fmt"
	"strings"
)

// ResolveProfile applies exact, longest-prefix, then selected-default
// precedence and merges sparse profile overrides onto common capabilities.
func ResolveProfile(
	model string,
	common Capabilities,
	profiles []Profile,
	defaultProfileID string,
	hard Capabilities,
) (ProfileResolution, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return ProfileResolution{}, fmt.Errorf("model is required")
	}
	if err := ValidateCapabilities(common, hard); err != nil {
		return ProfileResolution{}, fmt.Errorf("common capabilities: %w", err)
	}

	defaultProfile, err := findDefaultProfile(profiles, defaultProfileID)
	if err != nil {
		return ProfileResolution{}, err
	}

	var exact *Profile
	for i := range profiles {
		profile := &profiles[i]
		if !profileEnabled(profile) {
			continue
		}
		for _, exactModel := range profile.ExactModels {
			if strings.TrimSpace(exactModel) != model {
				continue
			}
			if exact != nil && exact.ID != profile.ID {
				return ProfileResolution{}, fmt.Errorf("model %q matches multiple exact profiles", model)
			}
			exact = profile
		}
	}
	if exact != nil {
		return buildResolution(*exact, ProfileMatchExact, false, common, hard)
	}

	var prefixProfile *Profile
	longestPrefix := -1
	for i := range profiles {
		profile := &profiles[i]
		if !profileEnabled(profile) {
			continue
		}
		for _, rawPrefix := range profile.ModelPrefixes {
			prefix := strings.TrimSpace(rawPrefix)
			if prefix == "" || !strings.HasPrefix(model, prefix) {
				continue
			}
			switch {
			case len(prefix) > longestPrefix:
				prefixProfile = profile
				longestPrefix = len(prefix)
			case len(prefix) == longestPrefix && prefixProfile != nil && prefixProfile.ID != profile.ID:
				return ProfileResolution{}, fmt.Errorf("model %q has ambiguous longest-prefix profiles", model)
			}
		}
	}
	if prefixProfile != nil {
		return buildResolution(*prefixProfile, ProfileMatchPrefix, false, common, hard)
	}

	return buildResolution(*defaultProfile, ProfileMatchDefault, true, common, hard)
}

// ValidateCapabilities rejects configured values outside code-defined limits.
func ValidateCapabilities(configured, hard Capabilities) error {
	if err := validateStringsSubset("generation type", configured.GenerationTypes, hard.GenerationTypes); err != nil {
		return err
	}
	if err := validateOptionsSubset("duration", configured.Durations, hard.Durations); err != nil {
		return err
	}
	if err := validateOptionsSubset("aspect ratio", configured.AspectRatios, hard.AspectRatios); err != nil {
		return err
	}
	if configured.Media.MinItems < 0 {
		return fmt.Errorf("media minimum cannot be negative")
	}
	if configured.Media.MaxItems < configured.Media.MinItems {
		return fmt.Errorf("media maximum cannot be below minimum")
	}
	if configured.Media.MinItems < hard.Media.MinItems {
		return fmt.Errorf(
			"media minimum %d is below hard limit %d",
			configured.Media.MinItems,
			hard.Media.MinItems,
		)
	}
	if hard.Media.MaxItems > 0 && configured.Media.MaxItems > hard.Media.MaxItems {
		return fmt.Errorf("media maximum %d exceeds hard limit %d", configured.Media.MaxItems, hard.Media.MaxItems)
	}
	if configured.Media.AllowAudio && !hard.Media.AllowAudio {
		return fmt.Errorf("audio media is outside provider hard limits")
	}
	if err := validateMediaTypesSubset(configured.Media.AcceptedTypes, hard.Media.AcceptedTypes); err != nil {
		return err
	}
	if err := validateMediaRolesSubset(configured.Media.AllowedRoles, hard.Media.AllowedRoles); err != nil {
		return err
	}
	return nil
}

func findDefaultProfile(profiles []Profile, defaultProfileID string) (*Profile, error) {
	defaultProfileID = strings.TrimSpace(defaultProfileID)
	if defaultProfileID == "" {
		return nil, fmt.Errorf("default profile is required")
	}
	for i := range profiles {
		if profiles[i].ID == defaultProfileID {
			if !profileEnabled(&profiles[i]) {
				return nil, fmt.Errorf("default profile %q is disabled", defaultProfileID)
			}
			return &profiles[i], nil
		}
	}
	return nil, fmt.Errorf("default profile %q does not exist", defaultProfileID)
}

func profileEnabled(profile *Profile) bool {
	return profile.Enabled == nil || *profile.Enabled
}

func buildResolution(
	profile Profile,
	matchKind ProfileMatchKind,
	usedDefault bool,
	common Capabilities,
	hard Capabilities,
) (ProfileResolution, error) {
	resolved := cloneCapabilities(common)
	if profile.Overrides.GenerationTypes != nil {
		resolved.GenerationTypes = append([]string(nil), profile.Overrides.GenerationTypes...)
	}
	if profile.Overrides.Durations != nil {
		resolved.Durations = append([]Option(nil), profile.Overrides.Durations...)
	}
	if profile.Overrides.AspectRatios != nil {
		resolved.AspectRatios = append([]Option(nil), profile.Overrides.AspectRatios...)
	}
	if profile.Overrides.Media != nil {
		if profile.Overrides.Media.MinItems != nil {
			resolved.Media.MinItems = *profile.Overrides.Media.MinItems
		}
		if profile.Overrides.Media.MaxItems != nil {
			resolved.Media.MaxItems = *profile.Overrides.Media.MaxItems
		}
		if profile.Overrides.Media.AcceptedTypes != nil {
			resolved.Media.AcceptedTypes = append([]VideoMediaType(nil), profile.Overrides.Media.AcceptedTypes...)
		}
		if profile.Overrides.Media.AllowedRoles != nil {
			resolved.Media.AllowedRoles = append([]VideoMediaRole(nil), profile.Overrides.Media.AllowedRoles...)
		}
		if profile.Overrides.Media.AllowAudio != nil {
			resolved.Media.AllowAudio = *profile.Overrides.Media.AllowAudio
		}
	}
	if err := ValidateCapabilities(resolved, hard); err != nil {
		return ProfileResolution{}, fmt.Errorf("profile %q: %w", profile.ID, err)
	}
	return ProfileResolution{
		ProfileID:    profile.ID,
		ProfileLabel: profile.Label,
		MatchKind:    matchKind,
		UsedDefault:  usedDefault,
		Capabilities: resolved,
	}, nil
}

func cloneCapabilities(capabilities Capabilities) Capabilities {
	capabilities.GenerationTypes = append([]string(nil), capabilities.GenerationTypes...)
	capabilities.Durations = append([]Option(nil), capabilities.Durations...)
	capabilities.AspectRatios = append([]Option(nil), capabilities.AspectRatios...)
	capabilities.Media = cloneMediaCapabilities(capabilities.Media)
	return capabilities
}

func cloneMediaCapabilities(capabilities MediaCapabilities) MediaCapabilities {
	capabilities.AcceptedTypes = append([]VideoMediaType(nil), capabilities.AcceptedTypes...)
	capabilities.AllowedRoles = append([]VideoMediaRole(nil), capabilities.AllowedRoles...)
	return capabilities
}

func validateStringsSubset(name string, configured, hard []string) error {
	allowed := make(map[string]struct{}, len(hard))
	for _, value := range hard {
		allowed[value] = struct{}{}
	}
	for _, value := range configured {
		if _, ok := allowed[value]; !ok {
			return fmt.Errorf("%s %q is outside provider hard limits", name, value)
		}
	}
	return nil
}

func validateOptionsSubset(name string, configured, hard []Option) error {
	allowed := make(map[string]struct{}, len(hard))
	for _, option := range hard {
		allowed[option.Value+"\x00"+option.UpstreamKey] = struct{}{}
	}
	for _, option := range configured {
		if _, ok := allowed[option.Value+"\x00"+option.UpstreamKey]; !ok {
			return fmt.Errorf("%s %q is outside provider hard limits", name, option.Value)
		}
	}
	return nil
}

func validateMediaTypesSubset(configured, hard []VideoMediaType) error {
	allowed := make(map[VideoMediaType]struct{}, len(hard))
	for _, mediaType := range hard {
		allowed[mediaType] = struct{}{}
	}
	for _, mediaType := range configured {
		if _, ok := allowed[mediaType]; !ok {
			return fmt.Errorf("media type %q is outside provider hard limits", mediaType)
		}
	}
	return nil
}

func validateMediaRolesSubset(configured, hard []VideoMediaRole) error {
	allowed := make(map[VideoMediaRole]struct{}, len(hard))
	for _, role := range hard {
		allowed[role] = struct{}{}
	}
	for _, role := range configured {
		if _, ok := allowed[role]; !ok {
			return fmt.Errorf("media role %q is outside provider hard limits", role)
		}
	}
	return nil
}
