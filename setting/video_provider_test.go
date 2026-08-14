package setting

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveVideoProviderOwnershipNormalizesAndRejectsOverlap(t *testing.T) {
	ownership := resolveVideoProviderOwnership(
		videoProviderGroupSource{
			Provider:    VideoProviderSilkRoad,
			ChannelType: constant.ChannelTypeSilkRoad,
			Groups:      []string{" legacy ", "", "legacy", "shared"},
		},
		videoProviderGroupSource{
			Provider:    VideoProviderBrioi,
			ChannelType: constant.ChannelTypeBrioi,
			Groups:      []string{" brioi ", "brioi", "shared"},
		},
	)

	assert.Equal(t, VideoProviderOwnership{
		Provider:    VideoProviderSilkRoad,
		ChannelType: constant.ChannelTypeSilkRoad,
	}, ownership["legacy"])
	assert.Equal(t, VideoProviderOwnership{
		Provider:    VideoProviderBrioi,
		ChannelType: constant.ChannelTypeBrioi,
	}, ownership["brioi"])
	assert.NotContains(t, ownership, "shared")
	assert.NotContains(t, ownership, "unowned")
}

func TestValidateVideoProviderGroupsReturnsTargetedConflict(t *testing.T) {
	brioi := brioi_setting.GetBrioiSetting()
	previousGroups := append([]string(nil), brioi.VideoToolGroups...)
	t.Cleanup(func() { brioi.VideoToolGroups = previousGroups })
	brioi.VideoToolGroups = []string{" brioi-owned "}

	err := ValidateVideoProviderGroups(
		VideoProviderSilkRoad,
		[]string{"silkroad", " brioi-owned ", "silkroad"},
	)
	require.Error(t, err)
	var conflict *VideoProviderGroupConflictError
	require.True(t, errors.As(err, &conflict))
	assert.Equal(t, "brioi-owned", conflict.Group)
	assert.Equal(t, VideoProviderBrioi, conflict.ExistingProvider)
	assert.Equal(t, VideoProviderSilkRoad, conflict.RequestedProvider)
}

func TestResolveVideoProviderGroupUsesLegacySilkRoadGroups(t *testing.T) {
	silkRoad := silkroad_setting.GetSilkRoadSetting()
	brioi := brioi_setting.GetBrioiSetting()
	previousSilkRoadGroups := append([]string(nil), silkRoad.VideoToolGroups...)
	previousBrioiGroups := append([]string(nil), brioi.VideoToolGroups...)
	t.Cleanup(func() {
		silkRoad.VideoToolGroups = previousSilkRoadGroups
		brioi.VideoToolGroups = previousBrioiGroups
	})

	silkRoad.VideoToolGroups = []string{" legacy-video "}
	brioi.VideoToolGroups = []string{}

	owner, ok := ResolveVideoProviderGroup("legacy-video")
	require.True(t, ok)
	assert.Equal(t, VideoProviderSilkRoad, owner.Provider)
	assert.Equal(t, constant.ChannelTypeSilkRoad, owner.ChannelType)
}

func TestResolveVideoProviderForGroupsRequiresOneProvider(t *testing.T) {
	ownership := resolveVideoProviderOwnership(
		videoProviderGroupSource{
			Provider:    VideoProviderSilkRoad,
			ChannelType: constant.ChannelTypeSilkRoad,
			Groups:      []string{"silkroad"},
		},
		videoProviderGroupSource{
			Provider:    VideoProviderBrioi,
			ChannelType: constant.ChannelTypeBrioi,
			Groups:      []string{"brioi"},
		},
	)
	assert.Len(t, ownership, 2)

	silkRoad := silkroad_setting.GetSilkRoadSetting()
	brioi := brioi_setting.GetBrioiSetting()
	previousSilkRoadGroups := append([]string(nil), silkRoad.VideoToolGroups...)
	previousBrioiGroups := append([]string(nil), brioi.VideoToolGroups...)
	t.Cleanup(func() {
		silkRoad.VideoToolGroups = previousSilkRoadGroups
		brioi.VideoToolGroups = previousBrioiGroups
	})
	silkRoad.VideoToolGroups = []string{"silkroad"}
	brioi.VideoToolGroups = []string{"brioi"}

	_, _, err := ResolveVideoProviderForGroups([]string{"unowned", "silkroad", "brioi"})
	var resolutionError *VideoProviderResolutionError
	require.ErrorAs(t, err, &resolutionError)

	owner, groups, err := ResolveVideoProviderForGroups([]string{"unowned", " brioi ", "brioi"})
	require.NoError(t, err)
	assert.Equal(t, VideoProviderBrioi, owner.Provider)
	assert.Equal(t, []string{"brioi"}, groups)
}

func TestIsVideoGenerationToolEnabledUsesAnyProvider(t *testing.T) {
	tests := []struct {
		name            string
		globalEnabled   bool
		silkRoadEnabled bool
		brioiEnabled    bool
		xTokenEnabled   bool
		expectedEnabled bool
	}{
		{
			name:            "Brioi only",
			globalEnabled:   true,
			brioiEnabled:    true,
			expectedEnabled: true,
		},
		{
			name:            "SilkRoad only",
			globalEnabled:   true,
			silkRoadEnabled: true,
			expectedEnabled: true,
		},
		{
			name:            "all providers disabled",
			globalEnabled:   true,
			expectedEnabled: false,
		},
		{
			name:            "global disabled",
			globalEnabled:   false,
			silkRoadEnabled: true,
			brioiEnabled:    true,
			xTokenEnabled:   true,
			expectedEnabled: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedEnabled, videoGenerationToolEnabled(
				test.globalEnabled,
				test.silkRoadEnabled,
				test.brioiEnabled,
				test.xTokenEnabled,
			))
		})
	}
}
