package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/video_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoStorageUploadBlocksAtFreeTierSoftLimit(t *testing.T) {
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	resetR2UsageState()

	checkedAt := time.Unix(1_800_000_000, 0)
	recordR2Usage(video_setting.R2SoftLimitBytes()-1, checkedAt)
	blocked, err := VideoStorageUploadBlocked()
	assert.False(t, blocked)
	require.NoError(t, err)

	recordR2Usage(video_setting.R2SoftLimitBytes(), checkedAt)
	blocked, err = VideoStorageUploadBlocked()
	assert.True(t, blocked)
	require.ErrorIs(t, err, ErrVideoStorageQuotaExceeded)
	assert.Contains(t, err.Error(), "contact the administrator")

	snapshot := GetR2UsageSnapshot()
	assert.Equal(t, video_setting.R2SoftLimitBytes(), snapshot.UsageBytes)
	assert.Equal(t, video_setting.R2FreeTierBytes, snapshot.QuotaBytes)
	assert.Equal(t, checkedAt.Unix(), snapshot.CheckedAt)
	assert.True(t, snapshot.Blocked)
}

func TestLocalDriverUploadsAreNeverQuotaBlocked(t *testing.T) {
	withVideoStorageSetting(t, video_setting.StorageSetting{
		Driver:   video_setting.DriverLocal,
		LocalDir: t.TempDir(),
	})
	t.Cleanup(resetR2UsageState)
	recordR2Usage(video_setting.R2FreeTierBytes, time.Now())

	blocked, err := VideoStorageUploadBlocked()
	assert.False(t, blocked)
	require.NoError(t, err)
}

func TestUsageCheckFailureKeepsPreviousUploadGate(t *testing.T) {
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	resetR2UsageState()

	recordR2Usage(video_setting.R2SoftLimitBytes(), time.Unix(1_800_000_000, 0))
	require.True(t, IsR2UploadBlocked())

	recordR2UsageFailure(assert.AnError, time.Unix(1_800_003_600, 0))
	assert.True(t, IsR2UploadBlocked(), "a failed usage check must not lift the block")

	snapshot := GetR2UsageSnapshot()
	assert.Equal(t, int64(1_800_003_600), snapshot.CheckedAt)
	assert.Equal(t, assert.AnError.Error(), snapshot.LastError)

	recordR2Usage(0, time.Unix(1_800_007_200, 0))
	assert.False(t, IsR2UploadBlocked())
	assert.Empty(t, GetR2UsageSnapshot().LastError)
}

func TestIngestRefusesUploadAndKeepsChargeWhenQuotaBlocked(t *testing.T) {
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	recordR2Usage(video_setting.R2FreeTierBytes, time.Now())

	task := &model.Task{
		TaskID: "task_quota_blocked",
		Status: model.TaskStatusStoring,
		PrivateData: model.TaskPrivateData{
			StorageStatus:     "pending",
			UpstreamResultURL: "https://upstream.example/result.mp4",
		},
	}

	err := ingestOne(task, nil)
	require.ErrorIs(t, err, ErrVideoStorageQuotaExceeded)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), task.Status)
	assert.Equal(t, "failed", task.PrivateData.StorageStatus)
	assert.True(t, task.PrivateData.NoAutomaticRefund, "the original charge must be kept for admin review")
	assert.Equal(t, VideoStorageQuotaExceededMessage, task.PrivateData.StorageLastError)
	assert.Contains(t, task.FailReason, task.TaskID)
	assert.NotContains(t, task.FailReason, "upstream.example")
}

func TestParseR2UsageResponseAcceptsNumbersAndStrings(t *testing.T) {
	numeric := []byte(`{"success":true,"result":{"payloadSize":2048,"metadataSize":52}}`)
	usage, err := parseR2UsageResponse(numeric)
	require.NoError(t, err)
	assert.Equal(t, int64(2100), usage)

	stringly := []byte(`{"success":true,"result":{"payloadSize":"2048","metadataSize":"52"}}`)
	usage, err = parseR2UsageResponse(stringly)
	require.NoError(t, err)
	assert.Equal(t, int64(2100), usage)

	missingMetadata := []byte(`{"success":true,"result":{"payloadSize":10}}`)
	usage, err = parseR2UsageResponse(missingMetadata)
	require.NoError(t, err)
	assert.Equal(t, int64(10), usage)
}

func TestParseR2UsageResponseRejectsFailuresAndMissingPayload(t *testing.T) {
	failure := []byte(`{"success":false,"errors":[{"code":10000,"message":"Authentication error"}]}`)
	_, err := parseR2UsageResponse(failure)
	require.ErrorContains(t, err, "Authentication error")

	missing := []byte(`{"success":true,"result":{}}`)
	_, err = parseR2UsageResponse(missing)
	require.ErrorContains(t, err, "payloadSize")
}
