package service

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/video_setting"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVideoInputUploadTestDB(t *testing.T) {
	t.Helper()
	originalDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.VideoInputAsset{}))
	model.DB = db
	t.Cleanup(func() { model.DB = originalDB })
}

func withVideoInputUploadStore(t *testing.T, store *fakeR2Store) {
	t.Helper()
	previous := videoInputUploadStoreFactory
	videoInputUploadStoreFactory = func(video_setting.R2StorageSetting) (videoInputUploadStore, error) {
		return store, nil
	}
	t.Cleanup(func() { videoInputUploadStoreFactory = previous })
}

func TestCreateVideoInputAssetPresignRejectsInvalidKindAndSize(t *testing.T) {
	setupVideoInputUploadTestDB(t)
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	resetR2UsageState()
	store := newFakeR2Store()
	withVideoInputUploadStore(t, store)

	_, err := CreateVideoInputAssetPresign(context.Background(), 1, "doc", "image/png", 10)
	require.ErrorIs(t, err, ErrVideoInputUploadInvalid)

	_, err = CreateVideoInputAssetPresign(context.Background(), 1, "image", "image/png", 0)
	require.ErrorIs(t, err, ErrVideoInputUploadInvalid)

	_, err = CreateVideoInputAssetPresign(context.Background(), 1, "image", "application/pdf", 10)
	require.ErrorIs(t, err, ErrVideoInputUploadInvalid)
}

func TestCreateVideoInputAssetPresignAndCompleteSuccess(t *testing.T) {
	setupVideoInputUploadTestDB(t)
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	resetR2UsageState()
	store := newFakeR2Store()
	withVideoInputUploadStore(t, store)

	presign, err := CreateVideoInputAssetPresign(
		context.Background(),
		7,
		"image",
		"image/png",
		128,
	)
	require.NoError(t, err)
	require.NotEmpty(t, presign.AssetID)
	require.Contains(t, presign.UploadURL, "X-Amz-Signature=put")
	assert.Equal(t, "image/png", presign.UploadHeaders["Content-Type"])

	asset, err := model.GetVideoInputAssetByAssetID(7, presign.AssetID)
	require.NoError(t, err)
	assert.Equal(t, model.VideoInputAssetStatusPresigned, asset.Status)

	store.objects[asset.ObjectKey] = r2Object{
		Key:         asset.ObjectKey,
		Size:        100,
		ContentType: "image/png",
	}

	complete, err := CompleteVideoInputAsset(context.Background(), 7, presign.AssetID)
	require.NoError(t, err)
	assert.Equal(t, presign.AssetID, complete.AssetID)
	assert.Contains(t, complete.URL, "X-Amz-Signature=abc")
	assert.Equal(t, int64(100), complete.Size)
	assert.Greater(t, complete.ExpiresAt, time.Now().Unix())

	updated, err := model.GetVideoInputAssetByAssetID(7, presign.AssetID)
	require.NoError(t, err)
	assert.Equal(t, model.VideoInputAssetStatusReady, updated.Status)
}

func TestCompleteVideoInputAssetRejectsMissingObjectAndOversizedBody(t *testing.T) {
	setupVideoInputUploadTestDB(t)
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	resetR2UsageState()
	store := newFakeR2Store()
	withVideoInputUploadStore(t, store)

	presign, err := CreateVideoInputAssetPresign(context.Background(), 3, "image", "image/jpeg", 50)
	require.NoError(t, err)

	_, err = CompleteVideoInputAsset(context.Background(), 3, presign.AssetID)
	require.ErrorIs(t, err, ErrVideoInputUploadInvalid)

	asset, err := model.GetVideoInputAssetByAssetID(3, presign.AssetID)
	require.NoError(t, err)
	assert.Equal(t, model.VideoInputAssetStatusFailed, asset.Status)

	presign2, err := CreateVideoInputAssetPresign(context.Background(), 3, "image", "image/jpeg", 50)
	require.NoError(t, err)
	asset2, err := model.GetVideoInputAssetByAssetID(3, presign2.AssetID)
	require.NoError(t, err)
	store.objects[asset2.ObjectKey] = r2Object{
		Key:         asset2.ObjectKey,
		Size:        51,
		ContentType: "image/jpeg",
	}
	_, err = CompleteVideoInputAsset(context.Background(), 3, presign2.AssetID)
	require.ErrorIs(t, err, ErrVideoInputUploadInvalid)
	_, stillThere := store.objects[asset2.ObjectKey]
	assert.False(t, stillThere)
}

func TestCompleteVideoInputAssetEnforcesOwnership(t *testing.T) {
	setupVideoInputUploadTestDB(t)
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	resetR2UsageState()
	store := newFakeR2Store()
	withVideoInputUploadStore(t, store)

	presign, err := CreateVideoInputAssetPresign(context.Background(), 11, "image", "image/png", 20)
	require.NoError(t, err)

	_, err = CompleteVideoInputAsset(context.Background(), 99, presign.AssetID)
	require.ErrorIs(t, err, ErrVideoInputUploadNotFound)
}

func TestCreateVideoInputAssetPresignEnforcesPendingCap(t *testing.T) {
	setupVideoInputUploadTestDB(t)
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	resetR2UsageState()
	store := newFakeR2Store()
	withVideoInputUploadStore(t, store)

	for i := 0; i < videoInputMaxPendingAssets; i++ {
		_, err := CreateVideoInputAssetPresign(context.Background(), 21, "image", "image/png", 10)
		require.NoError(t, err)
	}
	_, err := CreateVideoInputAssetPresign(context.Background(), 21, "image", "image/png", 10)
	require.ErrorIs(t, err, ErrVideoInputUploadPendingCap)
}

func TestCreateVideoInputAssetPresignEnforcesRateLimit(t *testing.T) {
	setupVideoInputUploadTestDB(t)
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	resetR2UsageState()
	store := newFakeR2Store()
	withVideoInputUploadStore(t, store)

	// Isolate from other tests that share the package-level memory limiter.
	previousLimiter := videoInputPresignMemoryLimiter
	previousRedis := common.RedisEnabled
	videoInputPresignMemoryLimiter = common.InMemoryRateLimiter{}
	common.RedisEnabled = false
	t.Cleanup(func() {
		videoInputPresignMemoryLimiter = previousLimiter
		common.RedisEnabled = previousRedis
	})

	userID := 40404
	for i := 0; i < videoInputPresignMaxPerMinute; i++ {
		_, err := CreateVideoInputAssetPresign(context.Background(), userID, "image", "image/png", 8)
		require.NoError(t, err)
		// Keep pending count under the cap so only the rate limit fires.
		require.NoError(t, model.DB.Where("user_id = ?", userID).Delete(&model.VideoInputAsset{}).Error)
	}
	_, err := CreateVideoInputAssetPresign(context.Background(), userID, "image", "image/png", 8)
	require.ErrorIs(t, err, ErrVideoInputUploadRateLimited)
	assert.Equal(t, http.StatusTooManyRequests, VideoInputUploadHTTPStatus(err))
}

func TestCreateVideoInputAssetPresignRefusedWhenQuotaBlocked(t *testing.T) {
	setupVideoInputUploadTestDB(t)
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	recordR2Usage(video_setting.R2FreeTierBytes, time.Now())
	store := newFakeR2Store()
	withVideoInputUploadStore(t, store)

	_, err := CreateVideoInputAssetPresign(context.Background(), 5, "image", "image/png", 10)
	require.ErrorIs(t, err, ErrVideoStorageQuotaExceeded)
}

func TestDeleteVideoInputAssetUploadRemovesObjectAndRow(t *testing.T) {
	setupVideoInputUploadTestDB(t)
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	resetR2UsageState()
	store := newFakeR2Store()
	withVideoInputUploadStore(t, store)

	presign, err := CreateVideoInputAssetPresign(context.Background(), 8, "audio", "audio/mpeg", 40)
	require.NoError(t, err)
	asset, err := model.GetVideoInputAssetByAssetID(8, presign.AssetID)
	require.NoError(t, err)
	store.objects[asset.ObjectKey] = r2Object{Key: asset.ObjectKey, Size: 40, ContentType: "audio/mpeg"}

	require.NoError(t, DeleteVideoInputAssetUpload(context.Background(), 8, presign.AssetID))
	_, err = model.GetVideoInputAssetByAssetID(8, presign.AssetID)
	require.Error(t, err)
	_, ok := store.objects[asset.ObjectKey]
	assert.False(t, ok)

	err = DeleteVideoInputAssetUpload(context.Background(), 8, presign.AssetID)
	require.ErrorIs(t, err, ErrVideoInputUploadNotFound)
}

func TestCompleteVideoInputAssetMissingHeadUsesNotExist(t *testing.T) {
	setupVideoInputUploadTestDB(t)
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	resetR2UsageState()
	store := newFakeR2Store()
	store.headErr = os.ErrNotExist
	withVideoInputUploadStore(t, store)

	presign, err := CreateVideoInputAssetPresign(context.Background(), 9, "video", "video/mp4", 200)
	require.NoError(t, err)
	_, err = CompleteVideoInputAsset(context.Background(), 9, presign.AssetID)
	require.ErrorIs(t, err, ErrVideoInputUploadInvalid)
}
