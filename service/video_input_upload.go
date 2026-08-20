package service

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/video_setting"
)

const (
	videoInputPresignRateLimitMark = "VIA"
	videoInputPresignMaxPerMinute  = 20
	videoInputPresignWindowSeconds = int64(60)
	videoInputMaxPendingAssets     = 5
	videoInputMaxReadyAssets       = 30
	videoInputUploadPresignTTL     = 15 * time.Minute
)

var (
	ErrVideoInputUploadUnavailable = errors.New("video input upload requires the R2 video storage driver")
	ErrVideoInputUploadRateLimited = errors.New("video input upload rate limit exceeded")
	ErrVideoInputUploadPendingCap  = errors.New("too many incomplete video input uploads")
	ErrVideoInputUploadReadyCap    = errors.New("too many unused uploaded video inputs")
	ErrVideoInputUploadNotFound    = errors.New("video input asset not found")
	ErrVideoInputUploadInvalid     = errors.New("invalid video input upload request")
)

type VideoInputPresignResult struct {
	AssetID       string            `json:"asset_id"`
	UploadURL     string            `json:"upload_url"`
	UploadHeaders map[string]string `json:"upload_headers"`
	ExpiresAt     int64             `json:"expires_at"`
}

type VideoInputCompleteResult struct {
	AssetID     string `json:"asset_id"`
	URL         string `json:"url"`
	ExpiresAt   int64  `json:"expires_at"`
	Kind        string `json:"kind"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type videoInputUploadStore interface {
	HeadObject(ctx context.Context, key string) (r2Object, error)
	DeleteObject(ctx context.Context, key string) error
	PresignGetObject(ctx context.Context, key string, ttl time.Duration) (string, error)
	PresignPutObject(ctx context.Context, key string, contentType string, ttl time.Duration) (string, map[string]string, error)
}

var videoInputUploadStoreFactory = func(cfg video_setting.R2StorageSetting) (videoInputUploadStore, error) {
	return newR2HTTPObjectStore(cfg)
}

var videoInputPresignMemoryLimiter common.InMemoryRateLimiter

func CreateVideoInputAssetPresign(
	ctx context.Context,
	userID int,
	kind string,
	contentType string,
	size int64,
) (*VideoInputPresignResult, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("%w: user is required", ErrVideoInputUploadInvalid)
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	contentType, err := normalizeVideoInputUploadContentType(kind, contentType)
	if err != nil {
		return nil, err
	}
	if size <= 0 {
		return nil, fmt.Errorf("%w: size must be positive", ErrVideoInputUploadInvalid)
	}
	limits := settingUploadLimits()
	maxBytes := limits.MaxBytesForContentType(contentType)
	if size > maxBytes {
		return nil, fmt.Errorf("%w: size exceeds %d bytes", ErrVideoInputUploadInvalid, maxBytes)
	}

	storage := videoStorageSetting()
	if !storage.IsR2() {
		return nil, ErrVideoInputUploadUnavailable
	}
	if err := ValidateVideoR2StorageConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrVideoInputUploadUnavailable, err.Error())
	}
	if blocked, reason := VideoStorageUploadBlocked(); blocked {
		return nil, fmt.Errorf("%w: %w", ErrVideoInputUploadUnavailable, reason)
	}
	if !allowVideoInputPresign(ctx, userID) {
		return nil, ErrVideoInputUploadRateLimited
	}

	pending, err := model.CountVideoInputAssetsByStatus(userID, model.VideoInputAssetStatusPresigned)
	if err != nil {
		return nil, err
	}
	if pending >= videoInputMaxPendingAssets {
		return nil, ErrVideoInputUploadPendingCap
	}
	ready, err := model.CountVideoInputAssetsByStatus(userID, model.VideoInputAssetStatusReady)
	if err != nil {
		return nil, err
	}
	if ready >= videoInputMaxReadyAssets {
		return nil, ErrVideoInputUploadReadyCap
	}

	store, err := videoInputUploadStoreFactory(storage.R2)
	if err != nil {
		return nil, fmt.Errorf("%w: create object store: %v", ErrVideoInputUploadUnavailable, err)
	}
	assetID := model.NewVideoInputAssetID()
	objectKey := stagedVideoInputKey(storage.R2.InputPrefix, userID, contentType)
	uploadURL, headers, err := store.PresignPutObject(ctx, objectKey, contentType, videoInputUploadPresignTTL)
	if err != nil {
		return nil, fmt.Errorf("presign put object: %w", err)
	}
	expiresAt := time.Now().Add(videoInputUploadPresignTTL).Unix()
	asset := &model.VideoInputAsset{
		AssetId:     assetID,
		UserId:      userID,
		ObjectKey:   objectKey,
		Kind:        kind,
		ContentType: contentType,
		Size:        size,
		Status:      model.VideoInputAssetStatusPresigned,
		ExpiresAt:   expiresAt,
	}
	if err := model.CreateVideoInputAsset(asset); err != nil {
		return nil, err
	}
	return &VideoInputPresignResult{
		AssetID:       assetID,
		UploadURL:     uploadURL,
		UploadHeaders: headers,
		ExpiresAt:     expiresAt,
	}, nil
}

func CompleteVideoInputAsset(
	ctx context.Context,
	userID int,
	assetID string,
) (*VideoInputCompleteResult, error) {
	asset, err := model.GetVideoInputAssetByAssetID(userID, assetID)
	if err != nil {
		return nil, ErrVideoInputUploadNotFound
	}
	if asset.Status != model.VideoInputAssetStatusPresigned &&
		asset.Status != model.VideoInputAssetStatusReady {
		return nil, fmt.Errorf("%w: asset status is %s", ErrVideoInputUploadInvalid, asset.Status)
	}

	storage := videoStorageSetting()
	if !storage.IsR2() {
		return nil, ErrVideoInputUploadUnavailable
	}
	store, err := videoInputUploadStoreFactory(storage.R2)
	if err != nil {
		return nil, fmt.Errorf("%w: create object store: %v", ErrVideoInputUploadUnavailable, err)
	}

	object, headErr := store.HeadObject(ctx, asset.ObjectKey)
	if headErr != nil {
		asset.Status = model.VideoInputAssetStatusFailed
		_ = model.UpdateVideoInputAsset(asset)
		return nil, fmt.Errorf("%w: uploaded object is missing", ErrVideoInputUploadInvalid)
	}
	if object.Size <= 0 || object.Size > asset.Size {
		_ = store.DeleteObject(ctx, asset.ObjectKey)
		asset.Status = model.VideoInputAssetStatusFailed
		_ = model.UpdateVideoInputAsset(asset)
		return nil, fmt.Errorf("%w: uploaded object size is invalid", ErrVideoInputUploadInvalid)
	}
	detectedType := strings.TrimSpace(object.ContentType)
	if detectedType != "" {
		normalized, normErr := normalizeVideoInputUploadContentType(asset.Kind, detectedType)
		if normErr != nil || normalized != asset.ContentType {
			_ = store.DeleteObject(ctx, asset.ObjectKey)
			asset.Status = model.VideoInputAssetStatusFailed
			_ = model.UpdateVideoInputAsset(asset)
			return nil, fmt.Errorf("%w: uploaded object content type is invalid", ErrVideoInputUploadInvalid)
		}
	}

	getTTL := storage.R2.InputPresignTTL()
	url, err := store.PresignGetObject(ctx, asset.ObjectKey, getTTL)
	if err != nil {
		return nil, fmt.Errorf("presign get object: %w", err)
	}
	expiresAt := time.Now().Add(getTTL).Unix()
	asset.Size = object.Size
	asset.Status = model.VideoInputAssetStatusReady
	asset.ExpiresAt = expiresAt
	if err := model.UpdateVideoInputAsset(asset); err != nil {
		return nil, err
	}
	return &VideoInputCompleteResult{
		AssetID:     asset.AssetId,
		URL:         url,
		ExpiresAt:   expiresAt,
		Kind:        asset.Kind,
		ContentType: asset.ContentType,
		Size:        asset.Size,
	}, nil
}

func DeleteVideoInputAssetUpload(ctx context.Context, userID int, assetID string) error {
	asset, err := model.GetVideoInputAssetByAssetID(userID, assetID)
	if err != nil {
		return ErrVideoInputUploadNotFound
	}
	storage := videoStorageSetting()
	if storage.IsR2() {
		if store, storeErr := videoInputUploadStoreFactory(storage.R2); storeErr == nil {
			_ = store.DeleteObject(ctx, asset.ObjectKey)
		}
	}
	return model.DeleteVideoInputAsset(userID, assetID)
}

func normalizeVideoInputUploadContentType(kind, contentType string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil || mediaType == "" {
		return "", fmt.Errorf("%w: content_type is invalid", ErrVideoInputUploadInvalid)
	}
	mediaType = strings.ToLower(mediaType)
	if mediaType == "image/jpg" {
		mediaType = "image/jpeg"
	}
	switch kind {
	case model.VideoInputAssetKindImage:
		switch mediaType {
		case "image/jpeg", "image/png", "image/webp", "image/gif":
			return mediaType, nil
		}
	case model.VideoInputAssetKindAudio:
		switch mediaType {
		case "audio/mpeg", "audio/mp3", "audio/wav", "audio/x-wav", "audio/wave":
			return mediaType, nil
		}
	case model.VideoInputAssetKindVideo:
		switch mediaType {
		case "video/mp4", "video/quicktime":
			return mediaType, nil
		}
	default:
		return "", fmt.Errorf("%w: kind must be image, audio, or video", ErrVideoInputUploadInvalid)
	}
	return "", fmt.Errorf("%w: content_type %q is not allowed for kind %q", ErrVideoInputUploadInvalid, mediaType, kind)
}

func allowVideoInputPresign(ctx context.Context, userID int) bool {
	key := fmt.Sprintf("rateLimit:v2:user:%s:%d", videoInputPresignRateLimitMark, userID)
	if common.RedisEnabled && common.RDB != nil {
		allowed, err := redisAllowVideoInputPresign(ctx, key)
		if err == nil {
			return allowed
		}
		common.SysError(fmt.Sprintf("video input presign redis rate limit failed: %v", err))
	}
	videoInputPresignMemoryLimiter.Init(common.RateLimitKeyExpirationDuration)
	return videoInputPresignMemoryLimiter.Request(
		key,
		videoInputPresignMaxPerMinute,
		videoInputPresignWindowSeconds,
	)
}

func redisAllowVideoInputPresign(ctx context.Context, key string) (bool, error) {
	const script = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
end
if count > tonumber(ARGV[1]) then
  return 0
end
return 1
`
	result, err := common.RDB.Eval(
		ctx,
		script,
		[]string{key},
		videoInputPresignMaxPerMinute,
		videoInputPresignWindowSeconds,
	).Result()
	if err != nil {
		return false, err
	}
	switch typed := result.(type) {
	case int64:
		return typed == 1, nil
	default:
		return false, fmt.Errorf("unexpected redis reply type %T", result)
	}
}

func VideoInputUploadHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrVideoInputUploadRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrVideoInputUploadNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrVideoInputUploadUnavailable),
		errors.Is(err, ErrVideoStorageQuotaExceeded):
		return http.StatusServiceUnavailable
	case errors.Is(err, ErrVideoInputUploadInvalid),
		errors.Is(err, ErrVideoInputUploadPendingCap),
		errors.Is(err, ErrVideoInputUploadReadyCap):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
