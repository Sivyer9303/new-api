package service

import (
	"errors"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/setting/video_setting"
)

// VideoStorageQuotaExceededMessage is the single user- and log-facing reason used
// whenever an upload is refused because the R2 free tier is nearly exhausted.
const VideoStorageQuotaExceededMessage = "R2 bucket full, please contact the administrator"

var ErrVideoStorageQuotaExceeded = errors.New(VideoStorageQuotaExceededMessage)

// R2UsageSnapshot is the last known bucket usage plus the derived upload gate.
type R2UsageSnapshot struct {
	UsageBytes     int64  `json:"usage_bytes"`
	SoftLimitBytes int64  `json:"soft_limit_bytes"`
	QuotaBytes     int64  `json:"quota_bytes"`
	Blocked        bool   `json:"blocked"`
	CheckedAt      int64  `json:"checked_at"`
	LastError      string `json:"last_error"`
}

var (
	r2UsageMutex    sync.RWMutex
	r2UsageBytes    int64
	r2UsageChecked  int64
	r2UploadBlocked bool
	r2UsageLastErr  string
)

// GetR2UsageSnapshot returns the cached usage reading for admin display.
func GetR2UsageSnapshot() R2UsageSnapshot {
	r2UsageMutex.RLock()
	defer r2UsageMutex.RUnlock()
	return R2UsageSnapshot{
		UsageBytes:     r2UsageBytes,
		SoftLimitBytes: video_setting.R2SoftLimitBytes(),
		QuotaBytes:     video_setting.R2FreeTierBytes,
		Blocked:        r2UploadBlocked,
		CheckedAt:      r2UsageChecked,
		LastError:      r2UsageLastErr,
	}
}

// IsR2UploadBlocked reports whether the last successful usage reading crossed the
// free-tier soft limit. A failed reading never flips the gate on its own.
func IsR2UploadBlocked() bool {
	r2UsageMutex.RLock()
	defer r2UsageMutex.RUnlock()
	return r2UploadBlocked
}

// VideoStorageUploadBlocked reports whether new object uploads must be refused.
// Only the R2 driver is gated; local disk has no free-tier allowance to protect.
func VideoStorageUploadBlocked() (bool, error) {
	if !videoStorageSetting().IsR2() {
		return false, nil
	}
	if !IsR2UploadBlocked() {
		return false, nil
	}
	return true, ErrVideoStorageQuotaExceeded
}

func recordR2Usage(usageBytes int64, at time.Time) bool {
	blocked := usageBytes >= video_setting.R2SoftLimitBytes()
	r2UsageMutex.Lock()
	defer r2UsageMutex.Unlock()
	changed := blocked != r2UploadBlocked
	r2UsageBytes = usageBytes
	r2UsageChecked = at.Unix()
	r2UploadBlocked = blocked
	r2UsageLastErr = ""
	return changed
}

// recordR2UsageFailure keeps the previous gate so a Cloudflare API outage cannot
// silently stop (or resume) uploads.
func recordR2UsageFailure(cause error, at time.Time) {
	r2UsageMutex.Lock()
	defer r2UsageMutex.Unlock()
	r2UsageChecked = at.Unix()
	if cause != nil {
		r2UsageLastErr = cause.Error()
	}
}

func resetR2UsageState() {
	r2UsageMutex.Lock()
	defer r2UsageMutex.Unlock()
	r2UsageBytes = 0
	r2UsageChecked = 0
	r2UploadBlocked = false
	r2UsageLastErr = ""
}
