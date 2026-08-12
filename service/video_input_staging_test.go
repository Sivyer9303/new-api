package service

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/video_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withStagedInputStore(t *testing.T, store *fakeR2Store) {
	t.Helper()
	previous := videoInputStagingStore
	videoInputStagingStore = func(video_setting.R2StorageSetting) (videoInputStore, error) {
		return store, nil
	}
	t.Cleanup(func() { videoInputStagingStore = previous })
}

func imageDataURL(payload string) string {
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte(payload))
}

func TestStageVideoInputMediaUploadsDataURLAndReturnsPresignedURL(t *testing.T) {
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	resetR2UsageState()
	store := newFakeR2Store()
	withStagedInputStore(t, store)

	signed, err := StageVideoInputMedia(context.Background(), 42, imageDataURL("jpeg-bytes"))
	require.NoError(t, err)
	assert.Contains(t, signed, "video-inputs/42/")
	assert.Contains(t, signed, ".jpg")
	assert.Contains(t, signed, "X-Amz-Expires=21600")

	require.Len(t, store.bodies, 1)
	for key, payload := range store.bodies {
		assert.True(t, strings.HasPrefix(key, "video-inputs/42/"))
		assert.Equal(t, "jpeg-bytes", string(payload))
	}
}

func TestStageVideoInputMediaPassesThroughRemoteURLs(t *testing.T) {
	withVideoStorageSetting(t, r2TestStorageSetting())
	store := newFakeR2Store()
	withStagedInputStore(t, store)

	remote := "https://cdn.example.com/reference.png"
	staged, err := StageVideoInputMedia(context.Background(), 1, remote)
	require.NoError(t, err)
	assert.Equal(t, remote, staged)
	assert.Equal(t, 0, store.putCalls)

	empty, err := StageVideoInputMedia(context.Background(), 1, "  ")
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestStageVideoInputMediaRequiresConfiguredR2Driver(t *testing.T) {
	withVideoStorageSetting(t, video_setting.StorageSetting{
		Driver:   video_setting.DriverLocal,
		LocalDir: t.TempDir(),
	})
	store := newFakeR2Store()
	withStagedInputStore(t, store)

	_, err := StageVideoInputMedia(context.Background(), 1, imageDataURL("bytes"))
	require.ErrorIs(t, err, errVideoInputStagingUnavailable)
	assert.Equal(t, 0, store.putCalls)
}

func TestStageVideoInputMediaRefusedWhenQuotaBlocked(t *testing.T) {
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	recordR2Usage(video_setting.R2FreeTierBytes, time.Now())
	store := newFakeR2Store()
	withStagedInputStore(t, store)

	_, err := StageVideoInputMedia(context.Background(), 1, imageDataURL("bytes"))
	require.ErrorIs(t, err, ErrVideoStorageQuotaExceeded)
	assert.Equal(t, 0, store.putCalls)
}

func TestStageVideoInputMediaListPreservesOrder(t *testing.T) {
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	resetR2UsageState()
	withStagedInputStore(t, newFakeR2Store())

	staged, err := StageVideoInputMediaList(context.Background(), 7, []string{
		imageDataURL("first"),
		"https://cdn.example.com/second.png",
	})
	require.NoError(t, err)
	require.Len(t, staged, 2)
	assert.Contains(t, staged[0], "video-inputs/7/")
	assert.Equal(t, "https://cdn.example.com/second.png", staged[1])
}

func TestStagedVideoInputExtensionMapsKnownMediaTypes(t *testing.T) {
	assert.Equal(t, ".jpg", stagedVideoInputExtension("image/jpeg"))
	assert.Equal(t, ".png", stagedVideoInputExtension("image/png"))
	assert.Equal(t, ".mp3", stagedVideoInputExtension("audio/mpeg"))
	assert.Equal(t, ".bin", stagedVideoInputExtension("application/x-thing"))
	assert.Equal(t, ".bin", stagedVideoInputExtension(""))
}
