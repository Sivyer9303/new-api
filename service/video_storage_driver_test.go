package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingVideoReader struct {
	read bool
}

func (r *failingVideoReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, errors.New("source interrupted")
	}
	r.read = true
	copy(p, "partial")
	return len("partial"), nil
}

func TestLocalVideoStorageDriverStoresOpensAndDeletesAtomically(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	driver := &LocalVideoStorageDriver{
		RootDir: t.TempDir(),
		Now:     func() time.Time { return now },
	}

	stored, err := driver.Store(
		context.Background(),
		"task_public_1",
		bytes.NewBufferString("video-bytes"),
		VideoObjectMetadata{ContentType: "video/mp4"},
	)
	require.NoError(t, err)
	assert.Equal(t, "task_public_1", stored.ObjectKey)
	assert.Equal(t, int64(len("video-bytes")), stored.Size)
	assert.Equal(t, "video/mp4", stored.ContentType)
	assert.Equal(t, now.Unix(), stored.ReadyAt)
	assert.Equal(t, now.Add(7*24*time.Hour).Unix(), stored.ExpiresAt)

	handle, err := driver.Open(context.Background(), stored.ObjectKey)
	require.NoError(t, err)
	body, err := io.ReadAll(handle)
	require.NoError(t, err)
	require.NoError(t, handle.Close())
	assert.Equal(t, "video-bytes", string(body))

	require.NoError(t, driver.Delete(context.Background(), stored.ObjectKey))
	require.NoError(t, driver.Delete(context.Background(), stored.ObjectKey))
	_, err = driver.Open(context.Background(), stored.ObjectKey)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestLocalVideoStorageDriverRejectsUnsafeKeysAndCleansFailedWrites(t *testing.T) {
	root := t.TempDir()
	driver := &LocalVideoStorageDriver{RootDir: root}

	_, err := driver.Store(
		context.Background(),
		"../escape",
		bytes.NewBufferString("video"),
		VideoObjectMetadata{},
	)
	require.ErrorContains(t, err, "object key")

	_, err = driver.Store(
		context.Background(),
		"task_failed",
		&failingVideoReader{},
		VideoObjectMetadata{},
	)
	require.ErrorContains(t, err, "source interrupted")

	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLocalVideoStorageDriverRejectsEmptyObjects(t *testing.T) {
	driver := &LocalVideoStorageDriver{RootDir: t.TempDir()}
	_, err := driver.Store(
		context.Background(),
		"task_empty",
		bytes.NewReader(nil),
		VideoObjectMetadata{},
	)
	require.ErrorContains(t, err, "empty")
}

func TestValidateVideoStorageReadyRejectsDisabledOrIncompleteConfiguration(t *testing.T) {
	s := silkroad_setting.GetSilkRoadSetting()
	previous := s.Storage
	t.Cleanup(func() { s.Storage = previous })

	s.Storage.Enabled = false
	require.ErrorContains(t, ValidateVideoStorageReady(), "not enabled")

	s.Storage.Enabled = true
	s.Storage.Driver = "local"
	s.Storage.LocalDir = t.TempDir()
	s.Storage.MaxRetry = 3
	s.Storage.IngestNodeName = "node-a"
	s.Storage.PublicDownloadBaseURL = ""
	require.ErrorContains(t, ValidateVideoStorageReady(), "public download")

	s.Storage.PublicDownloadBaseURL = "https://video.example.com"
	require.NoError(t, ValidateVideoStorageReady())
}

func TestValidateVideoGenerationReadyEnforcesFeatureSwitch(t *testing.T) {
	previous := videoGenerationEnabled
	videoGenerationEnabled = func() bool { return false }
	t.Cleanup(func() { videoGenerationEnabled = previous })

	require.ErrorContains(t, ValidateVideoGenerationReady(), "not enabled")

	videoGenerationEnabled = func() bool { return true }
	s := silkroad_setting.GetSilkRoadSetting()
	previousStorage := s.Storage
	t.Cleanup(func() { s.Storage = previousStorage })
	s.Storage.Enabled = true
	s.Storage.Driver = "local"
	s.Storage.LocalDir = t.TempDir()
	s.Storage.MaxRetry = 3
	s.Storage.IngestNodeName = "node-a"
	s.Storage.PublicDownloadBaseURL = "https://video.example.com"

	require.NoError(t, ValidateVideoGenerationReady())
}
