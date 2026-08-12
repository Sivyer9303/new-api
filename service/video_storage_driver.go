package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/video_setting"
)

type VideoObjectMetadata struct {
	ContentType string
}

type StoredVideo struct {
	ObjectKey   string
	Size        int64
	ContentType string
	ReadyAt     int64
	ExpiresAt   int64
}

type VideoReadHandle interface {
	io.ReadSeeker
	io.Closer
	Stat() (os.FileInfo, error)
}

type VideoStorageDriver interface {
	Store(
		ctx context.Context,
		objectKey string,
		source io.Reader,
		metadata VideoObjectMetadata,
	) (StoredVideo, error)
	Open(ctx context.Context, objectKey string) (VideoReadHandle, error)
	Delete(ctx context.Context, objectKey string) error
}

// VideoStoragePresigner is implemented by object-storage drivers that deliver
// playback through a short-lived signed URL instead of an app-proxied stream.
type VideoStoragePresigner interface {
	PresignGet(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
}

// NewVideoStorageDriver builds the driver described by the effective settings.
func NewVideoStorageDriver(storage video_setting.StorageSetting) (VideoStorageDriver, error) {
	if !storage.IsR2() {
		return &LocalVideoStorageDriver{
			RootDir:       storage.LocalDir,
			RetentionDays: storage.RetentionDays(),
		}, nil
	}
	objects, err := newR2HTTPObjectStore(storage.R2)
	if err != nil {
		return nil, err
	}
	return &R2VideoStorageDriver{
		Objects:       objects,
		Prefix:        storage.R2.ResultPrefix,
		RetentionDays: storage.RetentionDays(),
		PresignTTL:    storage.R2.ResultPresignTTL(),
	}, nil
}

// CurrentVideoStorageDriver builds the driver for the live configuration.
func CurrentVideoStorageDriver() (VideoStorageDriver, error) {
	return NewVideoStorageDriver(setting.GetEffectiveVideoSetting().Storage)
}

type LocalVideoStorageDriver struct {
	RootDir       string
	RetentionDays int
	Now           func() time.Time
}

func (d *LocalVideoStorageDriver) Store(
	ctx context.Context,
	objectKey string,
	source io.Reader,
	metadata VideoObjectMetadata,
) (stored StoredVideo, err error) {
	finalPath, err := d.objectPath(objectKey)
	if err != nil {
		return StoredVideo{}, err
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return StoredVideo{}, err
	}

	temp, err := os.CreateTemp(filepath.Dir(finalPath), "."+objectKey+"-*.tmp")
	if err != nil {
		return StoredVideo{}, err
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		if err != nil {
			_ = os.Remove(tempPath)
		}
	}()

	size, err := copyVideoWithContext(ctx, temp, source)
	if err != nil {
		return StoredVideo{}, err
	}
	if size <= 0 {
		return StoredVideo{}, errors.New("stored video is empty")
	}
	if err = temp.Sync(); err != nil {
		return StoredVideo{}, err
	}
	if err = temp.Close(); err != nil {
		return StoredVideo{}, err
	}

	if existing, statErr := os.Stat(finalPath); statErr == nil && existing.Size() > 0 {
		_ = os.Remove(tempPath)
		return d.storedVideo(objectKey, existing.Size(), metadata.ContentType), nil
	}
	if err = os.Rename(tempPath, finalPath); err != nil {
		return StoredVideo{}, err
	}

	handle, err := os.Open(finalPath)
	if err != nil {
		return StoredVideo{}, err
	}
	info, statErr := handle.Stat()
	closeErr := handle.Close()
	if statErr != nil {
		return StoredVideo{}, statErr
	}
	if closeErr != nil {
		return StoredVideo{}, closeErr
	}
	if info.Size() <= 0 {
		_ = os.Remove(finalPath)
		return StoredVideo{}, errors.New("stored video is empty")
	}
	return d.storedVideo(objectKey, info.Size(), metadata.ContentType), nil
}

func (d *LocalVideoStorageDriver) Open(ctx context.Context, objectKey string) (VideoReadHandle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := d.objectPath(objectKey)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (d *LocalVideoStorageDriver) Delete(ctx context.Context, objectKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := d.objectPath(objectKey)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (d *LocalVideoStorageDriver) objectPath(objectKey string) (string, error) {
	key := strings.TrimSpace(objectKey)
	if key == "" || key == "." || key == ".." || filepath.Base(key) != key ||
		strings.ContainsAny(key, `/\`) {
		return "", fmt.Errorf("invalid video object key %q", objectKey)
	}
	root := strings.TrimSpace(d.RootDir)
	if root == "" {
		return "", errors.New("video storage root directory is empty")
	}
	return filepath.Join(root, key), nil
}

func (d *LocalVideoStorageDriver) storedVideo(
	objectKey string,
	size int64,
	contentType string,
) StoredVideo {
	now := time.Now()
	if d.Now != nil {
		now = d.Now()
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	retention := d.RetentionDays
	if retention < video_setting.MinRetentionDays {
		retention = video_setting.DefaultLocalRetentionDays
	}
	return StoredVideo{
		ObjectKey:   objectKey,
		Size:        size,
		ContentType: contentType,
		ReadyAt:     now.Unix(),
		ExpiresAt:   now.Add(time.Duration(retention) * 24 * time.Hour).Unix(),
	}
}

func copyVideoWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buffer := make([]byte, 32*1024)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		read, readErr := src.Read(buffer)
		if read > 0 {
			count, writeErr := dst.Write(buffer[:read])
			written += int64(count)
			if writeErr != nil {
				return written, writeErr
			}
			if count != read {
				return written, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return written, nil
			}
			return written, readErr
		}
	}
}
