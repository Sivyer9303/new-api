package brioi_setting

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type profileWire struct {
	ID              string                   `json:"id"`
	Model           string                   `json:"model"`
	Label           string                   `json:"label"`
	Enabled         *bool                    `json:"enabled"`
	ExactModels     []string                 `json:"exact_models"`
	ModelPrefixes   []string                 `json:"model_prefixes"`
	Durations       json.RawMessage          `json:"durations"`
	Resolutions     json.RawMessage          `json:"resolutions"`
	AspectRatios    json.RawMessage          `json:"aspect_ratios"`
	GenerationModes *[]GenerationModeSetting `json:"generation_modes"`
	GenerationTypes []string                 `json:"generation_types"`
	MaxImages       int                      `json:"max_images"`
	Overrides       *profileOverridesWire    `json:"overrides"`
}

type profileOverridesWire struct {
	Durations       json.RawMessage  `json:"durations"`
	Resolutions     json.RawMessage  `json:"resolutions"`
	AspectRatios    json.RawMessage  `json:"aspect_ratios"`
	GenerationTypes []string         `json:"generation_types"`
	Media           profileMediaWire `json:"media"`
}

type profileMediaWire struct {
	MaxItems  int `json:"max_items"`
	MaxImages int `json:"max_images"`
}

type profileOptionWire struct {
	Value   string `json:"value"`
	Enabled *bool  `json:"enabled"`
}

// UnmarshalJSON accepts the canonical backend representation and the
// option-item/overrides representation used by the administrator settings UI.
func (profile *Profile) UnmarshalJSON(data []byte) error {
	var wire profileWire
	if err := common.Unmarshal(data, &wire); err != nil {
		return err
	}

	modelName := strings.TrimSpace(wire.Model)
	if modelName == "" && len(wire.ExactModels) == 1 {
		modelName = strings.TrimSpace(wire.ExactModels[0])
	}
	if modelName == "" {
		modelName = strings.TrimSpace(wire.ID)
	}
	if wire.ID != "" && strings.TrimSpace(wire.ID) != modelName {
		return fmt.Errorf("id %q must match exact Brioi model %q", wire.ID, modelName)
	}
	if len(wire.ExactModels) > 0 {
		if len(wire.ExactModels) != 1 || strings.TrimSpace(wire.ExactModels[0]) != modelName {
			return fmt.Errorf("exact_models must contain only %q", modelName)
		}
	}
	if len(wire.ModelPrefixes) > 0 {
		return fmt.Errorf("model_prefixes are not supported by Brioi exact profiles")
	}
	enabled := true
	if wire.Enabled != nil {
		enabled = *wire.Enabled
	}

	durationsRaw := wire.Durations
	resolutionsRaw := wire.Resolutions
	aspectRatiosRaw := wire.AspectRatios
	generationTypes := wire.GenerationTypes
	maxImages := wire.MaxImages
	if wire.Overrides != nil {
		if len(wire.Overrides.Durations) > 0 {
			durationsRaw = wire.Overrides.Durations
		}
		if len(wire.Overrides.Resolutions) > 0 {
			resolutionsRaw = wire.Overrides.Resolutions
		}
		if len(wire.Overrides.AspectRatios) > 0 {
			aspectRatiosRaw = wire.Overrides.AspectRatios
		}
		if wire.Overrides.GenerationTypes != nil {
			generationTypes = wire.Overrides.GenerationTypes
		}
		if wire.Overrides.Media.MaxItems > 0 {
			maxImages = wire.Overrides.Media.MaxItems
		} else if wire.Overrides.Media.MaxImages > 0 {
			maxImages = wire.Overrides.Media.MaxImages
		}
	}

	hard, supported := hardProfile(modelName)
	if supported {
		if err := validateProfileWireIntegerOptions(durationsRaw, hard.Durations); err != nil {
			return fmt.Errorf("durations: %w", err)
		}
		if err := validateProfileWireStringOptions(resolutionsRaw, hard.Resolutions); err != nil {
			return fmt.Errorf("resolutions: %w", err)
		}
		if err := validateProfileWireStringOptions(aspectRatiosRaw, hard.AspectRatios); err != nil {
			return fmt.Errorf("aspect_ratios: %w", err)
		}
		if generationTypes != nil {
			allowedTypes := make(map[string]struct{}, len(hard.GenerationModes))
			for _, mode := range hard.GenerationModes {
				allowedTypes[mode.Value] = struct{}{}
			}
			seenTypes := make(map[string]struct{}, len(generationTypes))
			for _, generationType := range generationTypes {
				generationType = strings.TrimSpace(generationType)
				if _, ok := allowedTypes[generationType]; !ok {
					return fmt.Errorf("generation type %q is outside the Brioi hard capabilities", generationType)
				}
				if _, duplicate := seenTypes[generationType]; duplicate {
					return fmt.Errorf("generation type %q is duplicated", generationType)
				}
				seenTypes[generationType] = struct{}{}
			}
		}
	}

	durations, err := parseProfileIntegerOptions(durationsRaw)
	if err != nil {
		return fmt.Errorf("durations: %w", err)
	}
	resolutions, err := parseProfileStringOptions(resolutionsRaw)
	if err != nil {
		return fmt.Errorf("resolutions: %w", err)
	}
	aspectRatios, err := parseProfileStringOptions(aspectRatiosRaw)
	if err != nil {
		return fmt.Errorf("aspect_ratios: %w", err)
	}

	var generationModes []GenerationModeSetting
	if wire.GenerationModes != nil {
		generationModes = append(
			[]GenerationModeSetting(nil),
			(*wire.GenerationModes)...,
		)
	} else {
		if supported {
			generationModes = append([]GenerationModeSetting(nil), hard.GenerationModes...)
		} else {
			generationModes = []GenerationModeSetting{}
		}
		if generationTypes != nil {
			enabledTypes := make(map[string]struct{}, len(generationTypes))
			for _, generationType := range generationTypes {
				enabledTypes[strings.TrimSpace(generationType)] = struct{}{}
			}
			for index := range generationModes {
				_, generationModes[index].Enabled = enabledTypes[generationModes[index].Value]
			}
		}
		if maxImages > 0 {
			for index := range generationModes {
				if generationModes[index].Value == GenerationMultiImage {
					generationModes[index].ImagesMax = maxImages
				}
			}
		}
	}

	*profile = Profile{
		Model:           modelName,
		Label:           strings.TrimSpace(wire.Label),
		Enabled:         enabled,
		Durations:       durations,
		Resolutions:     resolutions,
		AspectRatios:    aspectRatios,
		GenerationModes: generationModes,
	}
	return nil
}

func parseProfileIntegerOptions(raw json.RawMessage) ([]int, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var integers []int
	if err := common.Unmarshal(raw, &integers); err == nil {
		return integers, nil
	}
	var stringsValues []string
	if err := common.Unmarshal(raw, &stringsValues); err == nil {
		integers = make([]int, 0, len(stringsValues))
		for _, value := range stringsValues {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("value %q must be an integer", value)
			}
			integers = append(integers, parsed)
		}
		return integers, nil
	}
	var options []profileOptionWire
	if err := common.Unmarshal(raw, &options); err != nil {
		return nil, err
	}
	integers = make([]int, 0, len(options))
	for _, option := range options {
		if option.Enabled != nil && !*option.Enabled {
			continue
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(option.Value))
		if err != nil {
			return nil, fmt.Errorf("value %q must be an integer", option.Value)
		}
		integers = append(integers, parsed)
	}
	return integers, nil
}

func parseProfileStringOptions(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var values []string
	if err := common.Unmarshal(raw, &values); err == nil {
		return values, nil
	}
	var options []profileOptionWire
	if err := common.Unmarshal(raw, &options); err != nil {
		return nil, err
	}
	values = make([]string, 0, len(options))
	for _, option := range options {
		if option.Enabled != nil && !*option.Enabled {
			continue
		}
		values = append(values, strings.TrimSpace(option.Value))
	}
	return values, nil
}

func validateProfileWireIntegerOptions(raw json.RawMessage, allowed []int) error {
	if len(raw) == 0 {
		return nil
	}
	var options []profileOptionWire
	if err := common.Unmarshal(raw, &options); err != nil {
		return nil
	}
	allowedSet := make(map[int]struct{}, len(allowed))
	for _, value := range allowed {
		allowedSet[value] = struct{}{}
	}
	for _, option := range options {
		value, err := strconv.Atoi(strings.TrimSpace(option.Value))
		if err != nil {
			return fmt.Errorf("value %q must be an integer", option.Value)
		}
		if _, ok := allowedSet[value]; !ok {
			return fmt.Errorf("value %d is outside the Brioi hard capabilities", value)
		}
	}
	return nil
}

func validateProfileWireStringOptions(raw json.RawMessage, allowed []string) error {
	if len(raw) == 0 {
		return nil
	}
	var options []profileOptionWire
	if err := common.Unmarshal(raw, &options); err != nil {
		return nil
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, value := range allowed {
		allowedSet[value] = struct{}{}
	}
	for _, option := range options {
		value := strings.TrimSpace(option.Value)
		if _, ok := allowedSet[value]; !ok {
			return fmt.Errorf("value %q is outside the Brioi hard capabilities", value)
		}
	}
	return nil
}
