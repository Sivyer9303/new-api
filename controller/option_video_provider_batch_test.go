package controller

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
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
		Provider:        setting.VideoProviderBrioi,
		VideoToolGroups: []string{" brioi ", "brioi", ""},
		Profiles:        json.RawMessage(profiles),
	})
	require.NoError(t, err)
	require.Len(t, values, 2)
	assert.JSONEq(t, `["brioi"]`, values["brioi_setting.video_tool_groups"])
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
		Provider:        setting.VideoProviderBrioi,
		VideoToolGroups: []string{"brioi"},
		Profiles:        json.RawMessage(rawProfiles),
	})
	require.Error(t, err)

	after, err := common.Marshal(brioi)
	require.NoError(t, err)
	assert.JSONEq(t, string(before), string(after))
}
