package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/video_setting"
)

// maxStagedInputBytes caps a single staged reference asset. Reference images and
// MP3 clips are small; a larger payload is either abuse or a client bug.
const maxStagedInputBytes = 24 << 20

var errVideoInputStagingUnavailable = errors.New(
	"video input media staging requires the R2 video storage driver",
)

type videoInputStore interface {
	PutObject(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	PresignGetObject(ctx context.Context, key string, ttl time.Duration) (string, error)
}

// videoInputStagingStore is overridden in tests.
var videoInputStagingStore func(video_setting.R2StorageSetting) (videoInputStore, error) = func(
	cfg video_setting.R2StorageSetting,
) (videoInputStore, error) {
	return newR2HTTPObjectStore(cfg)
}

// StageVideoInputMedia uploads one reference asset to the R2 input prefix and
// returns a presigned URL the upstream provider can fetch. Values that already
// are http(s) URLs pass through unchanged.
func StageVideoInputMedia(ctx context.Context, channelID int, media string) (string, error) {
	media = strings.TrimSpace(media)
	if media == "" {
		return "", nil
	}
	if strings.HasPrefix(media, "http://") || strings.HasPrefix(media, "https://") {
		return media, nil
	}
	if !strings.HasPrefix(media, "data:") {
		return "", errors.New("reference media must be a data URL or an http(s) URL")
	}

	storage := videoStorageSetting()
	if !storage.IsR2() {
		return "", errVideoInputStagingUnavailable
	}
	if err := ValidateVideoR2StorageConfigured(); err != nil {
		return "", fmt.Errorf("%w: %s", errVideoInputStagingUnavailable, err.Error())
	}
	if blocked, reason := VideoStorageUploadBlocked(); blocked {
		return "", reason
	}

	body, contentType, err := openVideoDataURL(media)
	if err != nil {
		return "", err
	}
	defer body.Close()

	buffer := &bytes.Buffer{}
	size, err := copyVideoWithContext(ctx, buffer, io.LimitReader(body, maxStagedInputBytes+1))
	if err != nil {
		return "", err
	}
	if size <= 0 {
		return "", errors.New("reference media is empty")
	}
	if size > maxStagedInputBytes {
		return "", fmt.Errorf("reference media exceeds %d bytes", maxStagedInputBytes)
	}

	store, err := videoInputStagingStore(storage.R2)
	if err != nil {
		return "", err
	}
	key := stagedVideoInputKey(storage.R2.InputPrefix, channelID, contentType)
	if err := store.PutObject(ctx, key, bytes.NewReader(buffer.Bytes()), size, contentType); err != nil {
		return "", err
	}
	return store.PresignGetObject(ctx, key, storage.R2.InputPresignTTL())
}

// StageVideoInputMediaList stages every asset, preserving order.
func StageVideoInputMediaList(ctx context.Context, channelID int, media []string) ([]string, error) {
	if len(media) == 0 {
		return media, nil
	}
	staged := make([]string, 0, len(media))
	for index, item := range media {
		url, err := StageVideoInputMedia(ctx, channelID, item)
		if err != nil {
			return nil, fmt.Errorf("stage reference media %d: %w", index, err)
		}
		staged = append(staged, url)
	}
	return staged, nil
}

func stagedVideoInputKey(prefix string, channelID int, contentType string) string {
	name := common.GetUUID() + stagedVideoInputExtension(contentType)
	return video_setting.ObjectKey(
		video_setting.ObjectKey(prefix, strconv.Itoa(channelID)),
		name,
	)
}

// stagedVideoInputExtension keeps a recognizable suffix so upstream providers
// that sniff the URL path still see a plausible media file.
func stagedVideoInputExtension(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		return ".bin"
	}
	switch mediaType {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	default:
		return ".bin"
	}
}
