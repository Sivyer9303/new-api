package controller

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/QuantumNous/new-api/setting/video_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderOptionValidationIsIndependentFromUnrelatedStorage(t *testing.T) {
	brioi := brioi_setting.GetBrioiSetting()
	silkRoad := silkroad_setting.GetSilkRoadSetting()
	video := video_setting.GetVideoSetting()
	previousBrioi := *brioi
	previousSilkRoad := *silkRoad
	previousVideo := *video
	t.Cleanup(func() {
		*brioi = previousBrioi
		*silkRoad = previousSilkRoad
		*video = previousVideo
	})

	*brioi = brioi_setting.DefaultBrioiSetting()
	brioi.VideoToolGroups = []string{"brioi"}
	silkRoad.VideoToolGroups = []string{"silkroad"}
	video.Storage.Driver = "incomplete-storage"

	profiles, err := common.Marshal(brioi.Profiles)
	require.NoError(t, err)
	require.NoError(t, validateBrioiSettingOption(
		"brioi_setting.profiles",
		string(profiles),
	))

	silkRoad.Storage.Enabled = true
	silkRoad.Storage.Driver = "incomplete-storage"
	silkRoadProfiles, err := common.Marshal(silkRoad.Profiles)
	require.NoError(t, err)
	require.NoError(t, validateSilkRoadSettingOption(
		"silkroad_setting.profiles",
		string(silkRoadProfiles),
	))

	require.NoError(t, validateVideoSettingOption("video_setting.enabled", "true"))
}

func TestProviderOptionValidationReportsGroupAndOwner(t *testing.T) {
	brioi := brioi_setting.GetBrioiSetting()
	silkRoad := silkroad_setting.GetSilkRoadSetting()
	video := video_setting.GetVideoSetting()
	previousBrioi := *brioi
	previousSilkRoad := *silkRoad
	previousVideo := *video
	t.Cleanup(func() {
		*brioi = previousBrioi
		*silkRoad = previousSilkRoad
		*video = previousVideo
	})

	*brioi = brioi_setting.DefaultBrioiSetting()
	silkRoad.VideoToolGroups = []string{" shared "}
	video.VideoToolGroups = []string{" shared "}

	groups, err := common.Marshal([]string{"brioi", " shared ", "brioi"})
	require.NoError(t, err)
	err = validateBrioiSettingOption(
		"brioi_setting.video_tool_groups",
		string(groups),
	)
	require.Error(t, err)

	var conflict *setting.VideoProviderGroupConflictError
	require.True(t, errors.As(err, &conflict))
	assert.Equal(t, "shared", conflict.Group)
	assert.Equal(t, setting.VideoProviderSilkRoad, conflict.ExistingProvider)
	assert.Equal(t, setting.VideoProviderBrioi, conflict.RequestedProvider)
}

func TestVideoStorageOptionStillValidatesItsOwnSection(t *testing.T) {
	storage, err := common.Marshal(video_setting.StorageSetting{Driver: "unsupported"})
	require.NoError(t, err)

	err = validateVideoSettingOption("video_setting.storage", string(storage))
	require.ErrorContains(t, err, "driver")
}
