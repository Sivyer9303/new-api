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

// IsSilkRoadIngestNode reports whether this process is the configured SilkRoad
// video ingest node. Empty IngestNodeName returns false to avoid dual-writer races.
func IsVideoIngestNode() bool {
	ingest := setting.GetEffectiveVideoSetting().Storage.IngestNodeName
	if ingest == "" {
		return false
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

// WriteSilkRoadVideoFile writes video bytes for taskID under LocalDir.
func WriteSilkRoadVideoFile(taskID string, r io.Reader) (absPath string, size int64, err error) {
	driver := &LocalVideoStorageDriver{RootDir: setting.GetEffectiveVideoSetting().Storage.LocalDir}
	stored, err := driver.Store(context.Background(), taskID, r, VideoObjectMetadata{})
	if err != nil {
		return "", 0, err
	}
	abs, err := filepath.Abs(SilkRoadVideoLocalPath(stored.ObjectKey))
	if err != nil {
		return "", 0, err
	}
	return abs, stored.Size, nil
}

// OpenSilkRoadVideoFile opens the local video file for reading.
func OpenSilkRoadVideoFile(taskID string) (*os.File, error) {
	driver := &LocalVideoStorageDriver{RootDir: setting.GetEffectiveVideoSetting().Storage.LocalDir}
	handle, err := driver.Open(context.Background(), taskID)
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

func OpenStoredVideo(ctx context.Context, task *model.Task) (VideoReadHandle, error) {
	if task == nil {
		return nil, os.ErrInvalid
	}
	objectKey := strings.TrimSpace(task.PrivateData.StorageObjectKey)
	if objectKey == "" {
		if legacyPath := strings.TrimSpace(task.PrivateData.StoragePath); legacyPath != "" {
			return os.Open(legacyPath)
		}
	}
	if objectKey == "" {
		objectKey = strings.TrimSpace(task.TaskID)
	}
	driver := &LocalVideoStorageDriver{RootDir: setting.GetEffectiveVideoSetting().Storage.LocalDir}
	return driver.Open(ctx, objectKey)
}

// DeleteSilkRoadVideoFile removes the local video file for taskID.
func DeleteSilkRoadVideoFile(taskID string) error {
	driver := &LocalVideoStorageDriver{RootDir: setting.GetEffectiveVideoSetting().Storage.LocalDir}
	return driver.Delete(context.Background(), taskID)
}

// BuildSilkRoadPublicURL builds the public content URL for a stored video.
func BuildVideoPublicURL(taskID string) string {
	base := strings.TrimRight(setting.GetEffectiveVideoSetting().Storage.PublicDownloadBaseURL, "/")
	return base + "/v1/videos/" + taskID + "/content"
}

func BuildSilkRoadPublicURL(taskID string) string {
	return BuildVideoPublicURL(taskID)
}
