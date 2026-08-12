package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/video_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeR2Store struct {
	objects  map[string]r2Object
	bodies   map[string][]byte
	putCalls int
	putErr   error
	headErr  error
}

func newFakeR2Store() *fakeR2Store {
	return &fakeR2Store{
		objects: make(map[string]r2Object),
		bodies:  make(map[string][]byte),
	}
}

func (s *fakeR2Store) PutObject(
	_ context.Context,
	key string,
	body io.Reader,
	size int64,
	contentType string,
) error {
	s.putCalls++
	if s.putErr != nil {
		return s.putErr
	}
	payload, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if int64(len(payload)) != size {
		return fmt.Errorf("content length mismatch: header %d, body %d", size, len(payload))
	}
	s.bodies[key] = payload
	s.objects[key] = r2Object{
		Key:          key,
		Size:         size,
		ContentType:  contentType,
		LastModified: time.Unix(1_800_000_000, 0),
	}
	return nil
}

func (s *fakeR2Store) HeadObject(_ context.Context, key string) (r2Object, error) {
	if s.headErr != nil {
		return r2Object{}, s.headErr
	}
	object, ok := s.objects[key]
	if !ok {
		return r2Object{}, os.ErrNotExist
	}
	return object, nil
}

func (s *fakeR2Store) DeleteObject(_ context.Context, key string) error {
	delete(s.objects, key)
	delete(s.bodies, key)
	return nil
}

func (s *fakeR2Store) ListObjects(_ context.Context, prefix string, _ string) (r2ObjectPage, error) {
	page := r2ObjectPage{}
	for key, object := range s.objects {
		if strings.HasPrefix(key, prefix) {
			page.Objects = append(page.Objects, object)
		}
	}
	return page, nil
}

func (s *fakeR2Store) PresignGetObject(_ context.Context, key string, ttl time.Duration) (string, error) {
	return fmt.Sprintf(
		"https://bucket.r2.example/%s?X-Amz-Expires=%d&X-Amz-Signature=abc",
		key, int64(ttl/time.Second),
	), nil
}

// withVideoStorageSetting overrides the effective storage configuration for one test.
func withVideoStorageSetting(t *testing.T, storage video_setting.StorageSetting) {
	t.Helper()
	video_setting.NormalizeStorageSetting(&storage)
	previous := videoStorageSetting
	videoStorageSetting = func() video_setting.StorageSetting { return storage }
	t.Cleanup(func() { videoStorageSetting = previous })
}

func r2TestStorageSetting() video_setting.StorageSetting {
	return video_setting.StorageSetting{
		Driver:                video_setting.DriverR2,
		MaxRetry:              5,
		PublicDownloadBaseURL: "https://video.example.com",
		R2: video_setting.R2StorageSetting{
			AccountID:       "acct",
			AccessKeyID:     "ak",
			SecretAccessKey: "sk",
			APIToken:        "token",
			Bucket:          "videos",
			ResultPrefix:    "videos/",
			InputPrefix:     "video-inputs/",
			RetentionDays:   3,
		},
	}
}

func TestR2VideoStorageDriverStoresVerifiesAndDeletes(t *testing.T) {
	store := newFakeR2Store()
	now := time.Unix(1_800_000_000, 0)
	driver := &R2VideoStorageDriver{
		Objects:       store,
		Prefix:        "videos/",
		RetentionDays: 3,
		PresignTTL:    900 * time.Second,
		Now:           func() time.Time { return now },
	}

	stored, err := driver.Store(
		context.Background(),
		"task_public_1",
		bytes.NewBufferString("video-bytes"),
		VideoObjectMetadata{ContentType: "video/mp4"},
	)
	require.NoError(t, err)
	assert.Equal(t, "videos/task_public_1", stored.ObjectKey)
	assert.Equal(t, int64(len("video-bytes")), stored.Size)
	assert.Equal(t, "video/mp4", stored.ContentType)
	assert.Equal(t, now.Unix(), stored.ReadyAt)
	assert.Equal(t, now.Add(3*24*time.Hour).Unix(), stored.ExpiresAt)

	_, err = driver.Open(context.Background(), stored.ObjectKey)
	require.ErrorIs(t, err, ErrVideoStorageStreamUnsupported)

	require.NoError(t, driver.Delete(context.Background(), stored.ObjectKey))
	require.ErrorIs(t, driver.Exists(context.Background(), stored.ObjectKey), os.ErrNotExist)
}

func TestR2VideoStorageDriverRejectsEmptyAndUnverifiableObjects(t *testing.T) {
	store := newFakeR2Store()
	driver := &R2VideoStorageDriver{Objects: store, Prefix: "videos/"}

	_, err := driver.Store(
		context.Background(),
		"task_empty",
		bytes.NewBuffer(nil),
		VideoObjectMetadata{ContentType: "video/mp4"},
	)
	require.ErrorContains(t, err, "empty")
	assert.Equal(t, 0, store.putCalls)

	store.headErr = os.ErrNotExist
	_, err = driver.Store(
		context.Background(),
		"task_unverifiable",
		bytes.NewBufferString("bytes"),
		VideoObjectMetadata{ContentType: "video/mp4"},
	)
	require.ErrorContains(t, err, "verify stored video")
}

func TestR2VideoStorageDriverRefusesUploadWhenQuotaBlocked(t *testing.T) {
	withVideoStorageSetting(t, r2TestStorageSetting())
	t.Cleanup(resetR2UsageState)
	recordR2Usage(video_setting.R2SoftLimitBytes(), time.Now())

	store := newFakeR2Store()
	driver := &R2VideoStorageDriver{Objects: store, Prefix: "videos/"}

	_, err := driver.Store(
		context.Background(),
		"task_blocked",
		bytes.NewBufferString("video-bytes"),
		VideoObjectMetadata{ContentType: "video/mp4"},
	)
	require.ErrorIs(t, err, ErrVideoStorageQuotaExceeded)
	assert.Equal(t, 0, store.putCalls)
}

func TestR2VideoStorageDriverPresignsWithConfiguredTTL(t *testing.T) {
	driver := &R2VideoStorageDriver{
		Objects:    newFakeR2Store(),
		Prefix:     "videos/",
		PresignTTL: 900 * time.Second,
	}

	signed, err := driver.PresignGet(context.Background(), "task_public_1", 0)
	require.NoError(t, err)
	assert.Contains(t, signed, "videos/task_public_1")
	assert.Contains(t, signed, "X-Amz-Expires=900")
}

func TestR2ResolveKeyIsIdempotentAcrossPrefixedKeys(t *testing.T) {
	driver := &R2VideoStorageDriver{Prefix: "videos/"}

	assert.Equal(t, "videos/task_1", driver.ResolveKey("task_1"))
	assert.Equal(t, "videos/task_1", driver.ResolveKey("videos/task_1"))
	assert.Equal(t, "task_1", (&R2VideoStorageDriver{}).ResolveKey("task_1"))
}

func TestValidateR2ObjectKeyRejectsTraversalAndEmptySegments(t *testing.T) {
	require.NoError(t, validateR2ObjectKey("videos/task_1.mp4"))

	for _, key := range []string{"", "/videos/task", "videos//task", "videos/../task", "videos/task/"} {
		require.Error(t, validateR2ObjectKey(key), "expected %q to be rejected", key)
	}
}

func TestParseR2ListResultReadsKeysAndContinuationToken(t *testing.T) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <IsTruncated>true</IsTruncated>
  <NextContinuationToken>token-2</NextContinuationToken>
  <Contents>
    <Key>video-inputs/1/a.jpg</Key>
    <Size>1234</Size>
    <LastModified>2026-08-12T10:00:00.000Z</LastModified>
  </Contents>
</ListBucketResult>`)

	page, err := parseR2ListResult(body)
	require.NoError(t, err)
	require.Len(t, page.Objects, 1)
	assert.Equal(t, "video-inputs/1/a.jpg", page.Objects[0].Key)
	assert.Equal(t, int64(1234), page.Objects[0].Size)
	assert.Equal(t, 2026, page.Objects[0].LastModified.Year())
	assert.Equal(t, "token-2", page.NextToken)
}

func TestR2PresignedURLCarriesSignatureAndExpiry(t *testing.T) {
	config := r2TestStorageSetting().R2
	config.SecretAccessKey = "super-secret-signing-value"
	store, err := newR2HTTPObjectStore(config)
	require.NoError(t, err)
	store.now = func() time.Time { return time.Unix(1_800_000_000, 0) }

	signed, err := store.PresignGetObject(context.Background(), "videos/task_1", 900*time.Second)
	require.NoError(t, err)
	assert.Contains(t, signed, "https://acct.r2.cloudflarestorage.com/videos/videos/task_1")
	assert.Contains(t, signed, "X-Amz-Expires=900")
	assert.Contains(t, signed, "X-Amz-Signature=")
	assert.NotContains(t, signed, config.SecretAccessKey)
}
