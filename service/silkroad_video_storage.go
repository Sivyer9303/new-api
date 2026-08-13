package service

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
)

// IsVideoIngestNode reports whether this process may write video results.
// The local driver requires a designated node to avoid dual-writer races on a
// non-shared filesystem. R2 has no such constraint, so an empty node name means
// every node may ingest; database CAS claiming still prevents duplicate work.
func IsVideoIngestNode() bool {
	storage := videoStorageSetting()
	ingest := strings.TrimSpace(storage.IngestNodeName)
	if ingest == "" {
		return storage.IsR2()
	}
	return common.NodeName == ingest
}

func IsSilkRoadIngestNode() bool {
	return IsVideoIngestNode()
}

// SilkRoadVideoLocalPath returns the local filesystem path for a task video.
func VideoLocalPath(taskID string) string {
	return filepath.Join(setting.GetEffectiveVideoSetting().Storage.LocalDir, taskID)
}

func SilkRoadVideoLocalPath(taskID string) string {
	return VideoLocalPath(taskID)
}

// WriteVideoFile writes video bytes for taskID under LocalDir.
func WriteVideoFile(taskID string, r io.Reader) (absPath string, size int64, err error) {
	driver := localVideoStorageDriver()
	stored, err := driver.Store(context.Background(), taskID, r, VideoObjectMetadata{})
	if err != nil {
		return "", 0, err
	}
	abs, err := filepath.Abs(VideoLocalPath(stored.ObjectKey))
	if err != nil {
		return "", 0, err
	}
	return abs, stored.Size, nil
}

func WriteSilkRoadVideoFile(taskID string, r io.Reader) (absPath string, size int64, err error) {
	return WriteVideoFile(taskID, r)
}

// OpenVideoFile opens the local video file for reading.
func OpenVideoFile(taskID string) (*os.File, error) {
	handle, err := localVideoStorageDriver().Open(context.Background(), taskID)
	if err != nil {
		return nil, err
	}
	file, ok := handle.(*os.File)
	if !ok {
		_ = handle.Close()
		return nil, os.ErrInvalid
	}
	return file, nil
}

func OpenSilkRoadVideoFile(taskID string) (*os.File, error) {
	return OpenVideoFile(taskID)
}

// OpenStoredVideo opens a stored video for streaming. Object-storage drivers
// return ErrVideoStorageStreamUnsupported because their content is delivered by
// redirecting the client to a presigned URL.
func OpenStoredVideo(ctx context.Context, task *model.Task) (VideoReadHandle, error) {
	if task == nil {
		return nil, os.ErrInvalid
	}
	objectKey := storedVideoObjectKey(task)
	storage := videoStorageSetting()
	if storage.IsR2() {
		driver, err := NewVideoStorageDriver(storage)
		if err != nil {
			return nil, err
		}
		return driver.Open(ctx, objectKey)
	}
	if strings.TrimSpace(task.PrivateData.StorageObjectKey) == "" {
		if legacyPath := strings.TrimSpace(task.PrivateData.StoragePath); legacyPath != "" {
			return os.Open(legacyPath)
		}
	}
	return localVideoStorageDriver().Open(ctx, objectKey)
}

// PresignStoredVideoURL returns a short-lived direct download URL when the
// configured driver delivers content by redirect. The second result is false for
// drivers that stream through this application.
func PresignStoredVideoURL(ctx context.Context, task *model.Task) (string, bool, error) {
	if task == nil {
		return "", false, os.ErrInvalid
	}
	storage := videoStorageSetting()
	if !storage.IsR2() {
		return "", false, nil
	}
	driver, err := NewVideoStorageDriver(storage)
	if err != nil {
		return "", true, err
	}
	presigner, ok := driver.(VideoStoragePresigner)
	if !ok {
		return "", false, nil
	}
	signed, err := presigner.PresignGet(
		ctx,
		storedVideoObjectKey(task),
		storage.R2.ResultPresignTTL(),
	)
	if err != nil {
		return "", true, err
	}
	return signed, true, nil
}

// DeleteVideoFile removes the stored video object for taskID.
func DeleteVideoFile(taskID string) error {
	driver, err := CurrentVideoStorageDriver()
	if err != nil {
		return err
	}
	return driver.Delete(context.Background(), taskID)
}

func DeleteSilkRoadVideoFile(taskID string) error {
	return DeleteVideoFile(taskID)
}

// BuildSilkRoadPublicURL builds the public content URL for a stored video.
func BuildVideoPublicURL(taskID string) string {
	base := strings.TrimRight(setting.GetEffectiveVideoSetting().Storage.PublicDownloadBaseURL, "/")
	return base + "/v1/videos/" + taskID + "/content"
}

func BuildSilkRoadPublicURL(taskID string) string {
	return BuildVideoPublicURL(taskID)
}

func localVideoStorageDriver() *LocalVideoStorageDriver {
	storage := videoStorageSetting()
	return &LocalVideoStorageDriver{
		RootDir:       storage.LocalDir,
		RetentionDays: storage.RetentionDays(),
	}
}

func storedVideoObjectKey(task *model.Task) string {
	if task == nil {
		return ""
	}
	if objectKey := strings.TrimSpace(task.PrivateData.StorageObjectKey); objectKey != "" {
		return objectKey
	}
	return strings.TrimSpace(task.TaskID)
}
