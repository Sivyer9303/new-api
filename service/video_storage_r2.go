package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/setting/video_setting"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const (
	r2SigningService  = "s3"
	r2UnsignedPayload = "UNSIGNED-PAYLOAD"
	// r2MaxObjectBytes caps a single stored video so a hostile upstream cannot
	// fill the free tier (and local temp disk) with one response.
	r2MaxObjectBytes = int64(2) << 30
	r2ListPageSize   = 1000
)

// ErrVideoStorageStreamUnsupported is returned by object-storage drivers that
// cannot expose a seekable local handle. Callers must redirect to a presigned
// URL instead of streaming the body themselves.
var ErrVideoStorageStreamUnsupported = errors.New("video storage driver does not support direct streaming")

// r2ObjectKeyPattern keeps keys inside a charset that needs no URI escaping, so
// the SigV4 canonical path always matches the wire path.
var r2ObjectKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,511}$`)

func validateR2ObjectKey(key string) error {
	if !r2ObjectKeyPattern.MatchString(key) ||
		strings.Contains(key, "//") ||
		strings.Contains(key, "..") ||
		strings.HasSuffix(key, "/") {
		return fmt.Errorf("invalid r2 object key %q", key)
	}
	return nil
}

type r2Object struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
}

type r2ObjectPage struct {
	Objects   []r2Object
	NextToken string
}

// r2ObjectStore is the minimal S3-compatible surface the video storage needs.
// Tests substitute an in-memory implementation.
type r2ObjectStore interface {
	PutObject(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	HeadObject(ctx context.Context, key string) (r2Object, error)
	DeleteObject(ctx context.Context, key string) error
	ListObjects(ctx context.Context, prefix string, token string) (r2ObjectPage, error)
	PresignGetObject(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type r2HTTPObjectStore struct {
	endpoint    string
	bucket      string
	region      string
	credentials aws.Credentials
	client      *http.Client
	signer      *v4.Signer
	now         func() time.Time
}

func newR2HTTPObjectStore(cfg video_setting.R2StorageSetting) (*r2HTTPObjectStore, error) {
	endpoint := cfg.ResolveEndpoint()
	if endpoint == "" {
		return nil, errors.New("r2 endpoint is not configured")
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if bucket == "" {
		return nil, errors.New("r2 bucket is not configured")
	}
	accessKey := strings.TrimSpace(cfg.AccessKeyID)
	secretKey := strings.TrimSpace(cfg.SecretAccessKey)
	if accessKey == "" || secretKey == "" {
		return nil, errors.New("r2 access credentials are not configured")
	}
	return &r2HTTPObjectStore{
		endpoint: endpoint,
		bucket:   bucket,
		region:   cfg.ResolveRegion(),
		credentials: aws.Credentials{
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
		},
		client: &http.Client{Timeout: 30 * time.Minute},
		signer: v4.NewSigner(func(options *v4.SignerOptions) {
			// S3 canonical requests must not double-escape the object path.
			options.DisableURIPathEscaping = true
		}),
	}, nil
}

func (s *r2HTTPObjectStore) currentTime() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *r2HTTPObjectStore) objectURL(key string) (*url.URL, error) {
	if err := validateR2ObjectKey(key); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(s.endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid r2 endpoint %q", s.endpoint)
	}
	parsed.Path = "/" + s.bucket + "/" + key
	parsed.RawPath = ""
	parsed.RawQuery = ""
	return parsed, nil
}

func (s *r2HTTPObjectStore) sign(ctx context.Context, request *http.Request) error {
	request.Header.Set("X-Amz-Content-Sha256", r2UnsignedPayload)
	return s.signer.SignHTTP(
		ctx,
		s.credentials,
		request,
		r2UnsignedPayload,
		r2SigningService,
		s.region,
		s.currentTime().UTC(),
	)
}

func (s *r2HTTPObjectStore) PutObject(
	ctx context.Context,
	key string,
	body io.Reader,
	size int64,
	contentType string,
) error {
	target, err := s.objectURL(key)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, target.String(), body)
	if err != nil {
		return err
	}
	request.ContentLength = size
	if strings.TrimSpace(contentType) != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if err := s.sign(ctx, request); err != nil {
		return err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer drainAndCloseR2Response(response)
	return r2ResponseError(response, "put object")
}

func (s *r2HTTPObjectStore) HeadObject(ctx context.Context, key string) (r2Object, error) {
	target, err := s.objectURL(key)
	if err != nil {
		return r2Object{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, target.String(), nil)
	if err != nil {
		return r2Object{}, err
	}
	if err := s.sign(ctx, request); err != nil {
		return r2Object{}, err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return r2Object{}, err
	}
	defer drainAndCloseR2Response(response)
	if response.StatusCode == http.StatusNotFound {
		return r2Object{}, os.ErrNotExist
	}
	if err := r2ResponseError(response, "head object"); err != nil {
		return r2Object{}, err
	}
	size, _ := strconv.ParseInt(response.Header.Get("Content-Length"), 10, 64)
	lastModified, _ := http.ParseTime(response.Header.Get("Last-Modified"))
	return r2Object{
		Key:          key,
		Size:         size,
		ContentType:  response.Header.Get("Content-Type"),
		LastModified: lastModified,
	}, nil
}

func (s *r2HTTPObjectStore) DeleteObject(ctx context.Context, key string) error {
	target, err := s.objectURL(key)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, target.String(), nil)
	if err != nil {
		return err
	}
	if err := s.sign(ctx, request); err != nil {
		return err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer drainAndCloseR2Response(response)
	if response.StatusCode == http.StatusNotFound {
		return nil
	}
	return r2ResponseError(response, "delete object")
}

func (s *r2HTTPObjectStore) ListObjects(
	ctx context.Context,
	prefix string,
	token string,
) (r2ObjectPage, error) {
	parsed, err := url.Parse(s.endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return r2ObjectPage{}, fmt.Errorf("invalid r2 endpoint %q", s.endpoint)
	}
	parsed.Path = "/" + s.bucket
	query := url.Values{}
	query.Set("list-type", "2")
	query.Set("max-keys", strconv.Itoa(r2ListPageSize))
	if strings.TrimSpace(prefix) != "" {
		query.Set("prefix", prefix)
	}
	if strings.TrimSpace(token) != "" {
		query.Set("continuation-token", token)
	}
	parsed.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return r2ObjectPage{}, err
	}
	if err := s.sign(ctx, request); err != nil {
		return r2ObjectPage{}, err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return r2ObjectPage{}, err
	}
	defer drainAndCloseR2Response(response)
	if err := r2ResponseError(response, "list objects"); err != nil {
		return r2ObjectPage{}, err
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return r2ObjectPage{}, err
	}
	return parseR2ListResult(body)
}

func (s *r2HTTPObjectStore) PresignGetObject(
	ctx context.Context,
	key string,
	ttl time.Duration,
) (string, error) {
	target, err := s.objectURL(key)
	if err != nil {
		return "", err
	}
	if ttl <= 0 {
		return "", errors.New("presign ttl must be positive")
	}
	query := url.Values{}
	query.Set("X-Amz-Expires", strconv.FormatInt(int64(ttl/time.Second), 10))
	target.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return "", err
	}
	signedURL, _, err := s.signer.PresignHTTP(
		ctx,
		s.credentials,
		request,
		r2UnsignedPayload,
		r2SigningService,
		s.region,
		s.currentTime().UTC(),
	)
	if err != nil {
		return "", err
	}
	return signedURL, nil
}

type r2ListBucketResult struct {
	XMLName               xml.Name `xml:"ListBucketResult"`
	IsTruncated           bool     `xml:"IsTruncated"`
	NextContinuationToken string   `xml:"NextContinuationToken"`
	Contents              []struct {
		Key          string `xml:"Key"`
		Size         int64  `xml:"Size"`
		LastModified string `xml:"LastModified"`
	} `xml:"Contents"`
}

func parseR2ListResult(body []byte) (r2ObjectPage, error) {
	var parsed r2ListBucketResult
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return r2ObjectPage{}, fmt.Errorf("parse r2 list response: %w", err)
	}
	page := r2ObjectPage{Objects: make([]r2Object, 0, len(parsed.Contents))}
	for _, item := range parsed.Contents {
		lastModified, err := time.Parse(time.RFC3339, item.LastModified)
		if err != nil {
			lastModified = time.Time{}
		}
		page.Objects = append(page.Objects, r2Object{
			Key:          item.Key,
			Size:         item.Size,
			LastModified: lastModified,
		})
	}
	if parsed.IsTruncated {
		page.NextToken = parsed.NextContinuationToken
	}
	return page, nil
}

func drainAndCloseR2Response(response *http.Response) {
	if response == nil || response.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	_ = response.Body.Close()
}

// r2ResponseError converts a non-2xx response into an error without echoing the
// signed request URL, which would leak credentials into logs.
func r2ResponseError(response *http.Response, operation string) error {
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	detail := ""
	if response.Body != nil {
		snippet, _ := io.ReadAll(io.LimitReader(response.Body, 2<<10))
		detail = strings.TrimSpace(string(snippet))
	}
	if detail == "" {
		return fmt.Errorf("r2 %s failed with status %d", operation, response.StatusCode)
	}
	return fmt.Errorf("r2 %s failed with status %d: %s", operation, response.StatusCode, detail)
}

// R2VideoStorageDriver stores video results as Cloudflare R2 objects. Playback
// happens through short-lived presigned URLs, never by streaming through this app.
type R2VideoStorageDriver struct {
	Objects       r2ObjectStore
	Prefix        string
	RetentionDays int
	PresignTTL    time.Duration
	Now           func() time.Time
}

func (d *R2VideoStorageDriver) Store(
	ctx context.Context,
	objectKey string,
	source io.Reader,
	metadata VideoObjectMetadata,
) (StoredVideo, error) {
	key := d.ResolveKey(objectKey)
	if err := validateR2ObjectKey(key); err != nil {
		return StoredVideo{}, err
	}
	if blocked, reason := VideoStorageUploadBlocked(); blocked {
		return StoredVideo{}, reason
	}

	// R2 rejects aws-chunked streaming uploads, so the body must be buffered to
	// learn its exact length before signing the PUT.
	temp, err := os.CreateTemp("", "video-r2-*.tmp")
	if err != nil {
		return StoredVideo{}, err
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()

	size, err := copyVideoWithContext(ctx, temp, io.LimitReader(source, r2MaxObjectBytes+1))
	if err != nil {
		return StoredVideo{}, err
	}
	if size <= 0 {
		return StoredVideo{}, errors.New("stored video is empty")
	}
	if size > r2MaxObjectBytes {
		return StoredVideo{}, fmt.Errorf("stored video exceeds %d bytes", r2MaxObjectBytes)
	}
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		return StoredVideo{}, err
	}

	contentType := strings.TrimSpace(metadata.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := d.Objects.PutObject(ctx, key, temp, size, contentType); err != nil {
		return StoredVideo{}, err
	}
	stored, err := d.Objects.HeadObject(ctx, key)
	if err != nil {
		return StoredVideo{}, fmt.Errorf("verify stored video: %w", err)
	}
	if stored.Size <= 0 {
		return StoredVideo{}, errors.New("stored video is empty")
	}

	now := time.Now()
	if d.Now != nil {
		now = d.Now()
	}
	retention := d.RetentionDays
	if retention < video_setting.MinRetentionDays {
		retention = video_setting.DefaultR2RetentionDays
	}
	return StoredVideo{
		ObjectKey:   key,
		Size:        stored.Size,
		ContentType: contentType,
		ReadyAt:     now.Unix(),
		ExpiresAt:   now.Add(time.Duration(retention) * 24 * time.Hour).Unix(),
	}, nil
}

// Open always fails: R2-backed videos are delivered by redirecting the client to
// a presigned URL, so the app never proxies the object bytes.
func (d *R2VideoStorageDriver) Open(context.Context, string) (VideoReadHandle, error) {
	return nil, ErrVideoStorageStreamUnsupported
}

func (d *R2VideoStorageDriver) Delete(ctx context.Context, objectKey string) error {
	key := d.ResolveKey(objectKey)
	if err := validateR2ObjectKey(key); err != nil {
		return err
	}
	return d.Objects.DeleteObject(ctx, key)
}

func (d *R2VideoStorageDriver) PresignGet(
	ctx context.Context,
	objectKey string,
	ttl time.Duration,
) (string, error) {
	key := d.ResolveKey(objectKey)
	if err := validateR2ObjectKey(key); err != nil {
		return "", err
	}
	if ttl <= 0 {
		ttl = d.PresignTTL
	}
	if ttl <= 0 {
		ttl = time.Duration(video_setting.DefaultR2ResultPresignTTLSeconds) * time.Second
	}
	return d.Objects.PresignGetObject(ctx, key, ttl)
}

// Exists reports whether the object is still readable in the bucket.
func (d *R2VideoStorageDriver) Exists(ctx context.Context, objectKey string) error {
	key := d.ResolveKey(objectKey)
	if err := validateR2ObjectKey(key); err != nil {
		return err
	}
	_, err := d.Objects.HeadObject(ctx, key)
	return err
}

// ResolveKey accepts either a bare task identifier or an already-prefixed key so
// tasks stored before a prefix change stay addressable.
func (d *R2VideoStorageDriver) ResolveKey(objectKey string) string {
	key := strings.Trim(strings.TrimSpace(objectKey), "/")
	prefix := strings.Trim(strings.TrimSpace(d.Prefix), "/")
	if prefix == "" || strings.HasPrefix(key, prefix+"/") {
		return key
	}
	return prefix + "/" + key
}
