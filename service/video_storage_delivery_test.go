package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/video_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresignStoredVideoURLOnlyRedirectsForObjectStorage(t *testing.T) {
	task := &model.Task{
		TaskID:      "task_delivery",
		PrivateData: model.TaskPrivateData{StorageObjectKey: "videos/task_delivery"},
	}

	withVideoStorageSetting(t, video_setting.StorageSetting{
		Driver:   video_setting.DriverLocal,
		LocalDir: t.TempDir(),
	})
	signed, redirects, err := PresignStoredVideoURL(context.Background(), task)
	require.NoError(t, err)
	assert.False(t, redirects)
	assert.Empty(t, signed)

	withVideoStorageSetting(t, r2TestStorageSetting())
	signed, redirects, err = PresignStoredVideoURL(context.Background(), task)
	require.NoError(t, err)
	assert.True(t, redirects)
	assert.Contains(t, signed, "https://acct.r2.cloudflarestorage.com/videos/videos/task_delivery")
	assert.Contains(t, signed, "X-Amz-Expires=900")
}

func TestOpenStoredVideoStreamsLocalFilesAndRefusesObjectStorage(t *testing.T) {
	dir := t.TempDir()
	withVideoStorageSetting(t, video_setting.StorageSetting{
		Driver:   video_setting.DriverLocal,
		LocalDir: dir,
	})

	payload := []byte("fake-video-bytes")
	_, _, err := WriteSilkRoadVideoFile("task_open", bytes.NewReader(payload))
	require.NoError(t, err)

	task := &model.Task{
		TaskID:      "task_open",
		PrivateData: model.TaskPrivateData{StorageObjectKey: "task_open"},
	}
	handle, err := OpenStoredVideo(context.Background(), task)
	require.NoError(t, err)
	require.NoError(t, handle.Close())

	withVideoStorageSetting(t, r2TestStorageSetting())
	_, err = OpenStoredVideo(context.Background(), task)
	require.ErrorIs(t, err, ErrVideoStorageStreamUnsupported)
}

func TestVideoIngestNodeGatingDependsOnDriver(t *testing.T) {
	withVideoStorageSetting(t, video_setting.StorageSetting{
		Driver:   video_setting.DriverLocal,
		LocalDir: t.TempDir(),
	})
	assert.False(t, IsVideoIngestNode(), "local storage requires a designated ingest node")

	withVideoStorageSetting(t, r2TestStorageSetting())
	assert.True(t, IsVideoIngestNode(), "r2 storage lets any node ingest")

	r2WithNode := r2TestStorageSetting()
	r2WithNode.IngestNodeName = "node-that-is-not-this-process"
	withVideoStorageSetting(t, r2WithNode)
	assert.False(t, IsVideoIngestNode())
}

func TestLocalStorageRetentionFollowsConfiguredDays(t *testing.T) {
	driver := &LocalVideoStorageDriver{RootDir: t.TempDir(), RetentionDays: 2}

	stored, err := driver.Store(
		context.Background(),
		"task_retention",
		bytes.NewBufferString("video-bytes"),
		VideoObjectMetadata{ContentType: "video/mp4"},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(2*24*3600), stored.ExpiresAt-stored.ReadyAt)

	fallback := &LocalVideoStorageDriver{RootDir: t.TempDir()}
	stored, err = fallback.Store(
		context.Background(),
		"task_retention_default",
		bytes.NewBufferString("video-bytes"),
		VideoObjectMetadata{ContentType: "video/mp4"},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(video_setting.DefaultLocalRetentionDays*24*3600), stored.ExpiresAt-stored.ReadyAt)
}
