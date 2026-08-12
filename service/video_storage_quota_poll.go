package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/video_setting"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	r2UsagePollInterval = time.Hour
	r2UsageAPIBaseURL   = "https://api.cloudflare.com/client/v4"
	r2UsageRequestLimit = 30 * time.Second
)

var (
	r2UsagePollOnce   sync.Once
	r2UsageHTTPClient = &http.Client{Timeout: r2UsageRequestLimit}
)

// StartVideoStorageQuotaTask polls R2 bucket usage hourly so uploads can be
// stopped before the free tier is exhausted.
func StartVideoStorageQuotaTask() {
	r2UsagePollOnce.Do(func() {
		gopool.Go(func() {
			ticker := time.NewTicker(r2UsagePollInterval)
			defer ticker.Stop()

			runR2UsagePollTick()
			for range ticker.C {
				runR2UsagePollTick()
			}
		})
	})
}

func runR2UsagePollTick() {
	ctx := context.Background()
	if !videoStorageSetting().IsR2() {
		return
	}
	if err := RefreshR2Usage(ctx); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("r2 usage check failed: %s", err.Error()))
	}
	if err := RunVideoInputCleanupOnce(ctx); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("video input cleanup failed: %s", err.Error()))
	}
}

// RefreshR2Usage reads current bucket usage from the Cloudflare API and updates
// the upload gate. Failures leave the previous gate untouched.
func RefreshR2Usage(ctx context.Context) error {
	storage := videoStorageSetting()
	if !storage.IsR2() {
		return nil
	}
	usage, err := fetchR2BucketUsage(ctx, storage.R2)
	if err != nil {
		recordR2UsageFailure(err, time.Now())
		return err
	}
	if recordR2Usage(usage, time.Now()) {
		if usage >= video_setting.R2SoftLimitBytes() {
			logger.LogWarn(ctx, fmt.Sprintf(
				"video storage uploads disabled: r2 usage %d bytes reached the %d byte soft limit; %s",
				usage, video_setting.R2SoftLimitBytes(), VideoStorageQuotaExceededMessage,
			))
			return nil
		}
		logger.LogInfo(ctx, fmt.Sprintf(
			"video storage uploads re-enabled: r2 usage %d bytes is below the %d byte soft limit",
			usage, video_setting.R2SoftLimitBytes(),
		))
	}
	return nil
}

func fetchR2BucketUsage(ctx context.Context, cfg video_setting.R2StorageSetting) (int64, error) {
	account := strings.TrimSpace(cfg.AccountID)
	bucket := strings.TrimSpace(cfg.Bucket)
	token := strings.TrimSpace(cfg.APIToken)
	if account == "" || bucket == "" || token == "" {
		return 0, errors.New("r2 account, bucket, and api token are required to read usage")
	}

	endpoint := fmt.Sprintf(
		"%s/accounts/%s/r2/buckets/%s/usage",
		r2UsageAPIBaseURL,
		account,
		bucket,
	)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")

	response, err := r2UsageHTTPClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return 0, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("cloudflare usage api returned status %d", response.StatusCode)
	}
	return parseR2UsageResponse(body)
}

type cloudflareR2UsageResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Result struct {
		PayloadSize  any `json:"payloadSize"`
		MetadataSize any `json:"metadataSize"`
	} `json:"result"`
}

func parseR2UsageResponse(body []byte) (int64, error) {
	var parsed cloudflareR2UsageResponse
	if err := common.Unmarshal(body, &parsed); err != nil {
		return 0, fmt.Errorf("parse cloudflare usage response: %w", err)
	}
	if !parsed.Success {
		if len(parsed.Errors) > 0 {
			return 0, fmt.Errorf(
				"cloudflare usage api error %d: %s",
				parsed.Errors[0].Code, parsed.Errors[0].Message,
			)
		}
		return 0, errors.New("cloudflare usage api reported failure")
	}
	payload, payloadOK := r2UsageNumber(parsed.Result.PayloadSize)
	metadata, _ := r2UsageNumber(parsed.Result.MetadataSize)
	if !payloadOK {
		return 0, errors.New("cloudflare usage response is missing payloadSize")
	}
	return payload + metadata, nil
}

// r2UsageNumber accepts both JSON numbers and numeric strings, which the
// Cloudflare API has used interchangeably for byte counts.
func r2UsageNumber(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		if typed < 0 {
			return 0, false
		}
		return int64(typed), true
	case int64:
		if typed < 0 {
			return 0, false
		}
		return typed, true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil || parsed < 0 {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}
