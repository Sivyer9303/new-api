package controller

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/QuantumNous/new-api/setting/compatvideo_setting"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoProviderOptionValuesBuildsOneNormalizedBrioiRevision(t *testing.T) {
	silkRoad := silkroad_setting.GetSilkRoadSetting()
	brioi := brioi_setting.GetBrioiSetting()
	previousSilkRoad := *silkRoad
	previousBrioi := *brioi
	t.Cleanup(func() {
		*silkRoad = previousSilkRoad
		*brioi = previousBrioi
	})
	silkRoad.VideoToolGroups = nil
	*brioi = brioi_setting.DefaultBrioiSetting()

	profiles, err := common.Marshal(brioi.Profiles)
	require.NoError(t, err)
	values, err := videoProviderOptionValues(VideoProviderOptionUpdateRequest{
		Provider: setting.VideoProviderBrioi,
		Profiles: json.RawMessage(profiles),
	})
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.JSONEq(t, string(profiles), values["brioi_setting.profiles"])
}

func TestVideoProviderOptionValuesDoesNotMutateLiveConfigOnValidationFailure(t *testing.T) {
	brioi := brioi_setting.GetBrioiSetting()
	previous := *brioi
	t.Cleanup(func() { *brioi = previous })
	*brioi = brioi_setting.DefaultBrioiSetting()
	before, err := common.Marshal(brioi)
	require.NoError(t, err)

	invalidProfiles := append([]brioi_setting.Profile(nil), brioi.Profiles...)
	invalidProfiles[0].Durations = []int{999}
	rawProfiles, err := common.Marshal(invalidProfiles)
	require.NoError(t, err)

	_, err = videoProviderOptionValues(VideoProviderOptionUpdateRequest{
		Provider: setting.VideoProviderBrioi,
		Profiles: json.RawMessage(rawProfiles),
	})
	require.Error(t, err)

	after, err := common.Marshal(brioi)
	require.NoError(t, err)
	assert.JSONEq(t, string(before), string(after))
}

func TestVideoProviderOptionValuesAcceptsEmptyCompatVideoOverrides(t *testing.T) {
	values, err := videoProviderOptionValues(VideoProviderOptionUpdateRequest{
		Provider: setting.VideoProviderCompatVideo,
		Profiles: json.RawMessage(`[]`),
	})
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.JSONEq(t, `[]`, values["compatvideo_setting.profiles"])
}

func TestVideoProviderOptionValuesPersistsCompatVideoOverrides(t *testing.T) {
	overrides, err := common.Marshal([]compatvideo_setting.Profile{
		{
			ID:          compatvideo_setting.ProfileSeedance2,
			Durations:   []int{5, 10},
			Resolutions: []string{"1080p"},
			Dialect:     compatvideo_setting.DialectNewAPIGenerations,
		},
	})
	require.NoError(t, err)

	values, err := videoProviderOptionValues(VideoProviderOptionUpdateRequest{
		Provider: setting.VideoProviderCompatVideo,
		Profiles: json.RawMessage(overrides),
	})
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.JSONEq(t, string(overrides), values["compatvideo_setting.profiles"])
}

func TestVideoProviderOptionValuesRejectsUnknownCompatVideoProfile(t *testing.T) {
	overrides, err := common.Marshal([]compatvideo_setting.Profile{
		{ID: "not-a-profile"},
	})
	require.NoError(t, err)

	_, err = videoProviderOptionValues(VideoProviderOptionUpdateRequest{
		Provider: setting.VideoProviderCompatVideo,
		Profiles: json.RawMessage(overrides),
	})
	require.ErrorContains(t, err, "unknown compat_video profile")
}
