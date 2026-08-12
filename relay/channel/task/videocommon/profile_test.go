package videocommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveProfileUsesExactThenLongestPrefixThenDefault(t *testing.T) {
	common := Capabilities{
		GenerationTypes: []string{"text2video"},
		Durations:       []Option{{Value: "10", UpstreamKey: "seconds"}},
		AspectRatios:    []Option{{Value: "16:9", UpstreamKey: "aspect_ratio"}},
	}
	hard := Capabilities{
		GenerationTypes: []string{"text2video", "image2video"},
		Durations: []Option{
			{Value: "5", UpstreamKey: "seconds"},
			{Value: "10", UpstreamKey: "seconds"},
			{Value: "15", UpstreamKey: "seconds"},
		},
		AspectRatios: []Option{{Value: "16:9", UpstreamKey: "aspect_ratio"}},
	}
	profiles := []Profile{
		{
			ID:            "default",
			ModelPrefixes: []string{"seedance-"},
		},
		{
			ID:            "long-prefix",
			ModelPrefixes: []string{"seedance-2.0-"},
			Overrides: CapabilityOverrides{
				Durations: []Option{{Value: "5", UpstreamKey: "seconds"}},
			},
		},
		{
			ID:          "exact",
			ExactModels: []string{"seedance-2.0-pro"},
			Overrides: CapabilityOverrides{
				Durations: []Option{{Value: "15", UpstreamKey: "seconds"}},
			},
		},
	}

	exact, err := ResolveProfile("seedance-2.0-pro", common, profiles, "default", hard)
	require.NoError(t, err)
	assert.Equal(t, "exact", exact.ProfileID)
	assert.Equal(t, "15", exact.Capabilities.Durations[0].Value)
	assert.False(t, exact.UsedDefault)

	prefix, err := ResolveProfile("seedance-2.0-lite", common, profiles, "default", hard)
	require.NoError(t, err)
	assert.Equal(t, "long-prefix", prefix.ProfileID)
	assert.Equal(t, "5", prefix.Capabilities.Durations[0].Value)
	assert.False(t, prefix.UsedDefault)

	fallback, err := ResolveProfile("unclassified-model", common, profiles, "default", hard)
	require.NoError(t, err)
	assert.Equal(t, "default", fallback.ProfileID)
	assert.Equal(t, common, fallback.Capabilities)
	assert.True(t, fallback.UsedDefault)
}

func TestResolveProfileRejectsCapabilitiesOutsideHardLimits(t *testing.T) {
	common := Capabilities{
		GenerationTypes: []string{"text2video"},
		Durations:       []Option{{Value: "10", UpstreamKey: "seconds"}},
		AspectRatios:    []Option{{Value: "16:9", UpstreamKey: "aspect_ratio"}},
	}
	profiles := []Profile{{
		ID:          "default",
		ExactModels: []string{"model"},
		Overrides: CapabilityOverrides{
			GenerationTypes: []string{"unsupported"},
		},
	}}

	_, err := ResolveProfile("model", common, profiles, "default", common)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generation type")
}

func TestResolveProfileInheritsOmittedMediaOverrideFields(t *testing.T) {
	maxItems := 2
	common := Capabilities{
		Media: MediaCapabilities{
			MinItems:      1,
			MaxItems:      4,
			AcceptedTypes: []VideoMediaType{VideoMediaImage},
			AllowedRoles:  []VideoMediaRole{VideoMediaRoleReference},
			AllowAudio:    true,
		},
	}
	hard := common
	hard.Media.MaxItems = 8
	profiles := []Profile{{
		ID:          "default",
		ExactModels: []string{"model"},
		Overrides: CapabilityOverrides{
			Media: &MediaCapabilityOverrides{MaxItems: &maxItems},
		},
	}}

	resolved, err := ResolveProfile("model", common, profiles, "default", hard)
	require.NoError(t, err)
	assert.Equal(t, 1, resolved.Capabilities.Media.MinItems)
	assert.Equal(t, 2, resolved.Capabilities.Media.MaxItems)
	assert.Equal(t, []VideoMediaType{VideoMediaImage}, resolved.Capabilities.Media.AcceptedTypes)
	assert.Equal(t, []VideoMediaRole{VideoMediaRoleReference}, resolved.Capabilities.Media.AllowedRoles)
	assert.True(t, resolved.Capabilities.Media.AllowAudio)
}

func TestValidateCapabilitiesRejectsMinimumBelowHardLimit(t *testing.T) {
	configured := Capabilities{Media: MediaCapabilities{MinItems: 0, MaxItems: 2}}
	hard := Capabilities{Media: MediaCapabilities{MinItems: 1, MaxItems: 4}}

	err := ValidateCapabilities(configured, hard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "minimum")
}
