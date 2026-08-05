package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	silkRoadVideoIngestInterval = 30 * time.Second
	silkRoadVideoIngestBatch    = 20
)

var silkRoadVideoIngestOnce sync.Once

// silkRoadVideoFetchFunc downloads an upstream video body. Tests inject httptest.
type silkRoadVideoFetchFunc func(url string) (io.ReadCloser, error)

// shouldSilkRoadStore reports whether a completed task should be queued for
// local SilkRoad video ingest instead of exposing the upstream ResultURL.
func shouldSilkRoadStore(task *model.Task) bool {
	if task == nil {
		return false
	}
	if !silkroad_setting.GetSilkRoadSetting().Storage.Enabled {
		return false
	}
	return task.Platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI))
}

// markSilkRoadPendingStore records the upstream URL privately and points
// ResultURL at the public content path. Never puts the upstream URL in ResultURL.
func markSilkRoadPendingStore(task *model.Task, upstreamURL string) {
	task.PrivateData.UpstreamResultURL = upstreamURL
	task.PrivateData.StorageStatus = "pending"
	task.PrivateData.StorageRetryCount = 0
	task.PrivateData.ResultURL = BuildSilkRoadPublicURL(task.TaskID)
}

// StartSilkRoadVideoIngestTask starts the periodic ingest loop on the configured
// ingest node only. Non-ingest nodes return immediately.
func StartSilkRoadVideoIngestTask() {
	silkRoadVideoIngestOnce.Do(func() {
		if !IsSilkRoadIngestNode() {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf(
				"silkroad video ingest task started: tick=%s node=%s",
				silkRoadVideoIngestInterval, common.NodeName,
			))
			ticker := time.NewTicker(silkRoadVideoIngestInterval)
			defer ticker.Stop()

			_ = RunSilkRoadVideoIngestOnce(context.Background())
			for range ticker.C {
				_ = RunSilkRoadVideoIngestOnce(context.Background())
			}
		})
	})
}

// RunSilkRoadVideoIngestOnce claims pending/failed SilkRoad video tasks and
// downloads them to local storage. No-op unless this process is the ingest node.
// Download failures never refund quota.
func RunSilkRoadVideoIngestOnce(ctx context.Context) error {
	if !IsSilkRoadIngestNode() {
		return nil
	}
	storage := silkroad_setting.GetSilkRoadSetting().Storage
	if !storage.Enabled {
		return nil
	}

	tasks, err := claimSilkRoadIngestTasks(silkRoadVideoIngestBatch, storage.MaxRetry)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := ingestOne(task, defaultSilkRoadVideoFetch); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf(
				"silkroad ingest failed task=%s retry=%d: %s",
				task.TaskID, task.PrivateData.StorageRetryCount, err.Error(),
			))
		}
		if err := task.Update(); err != nil {
			logger.LogError(ctx, fmt.Sprintf(
				"silkroad ingest persist failed task=%s: %s",
				task.TaskID, err.Error(),
			))
		}
	}
	return nil
}

func claimSilkRoadIngestTasks(limit int, maxRetry int) ([]*model.Task, error) {
	if limit <= 0 {
		return nil, nil
	}
	platform := strconv.Itoa(constant.ChannelTypeNewAPI)
	var candidates []*model.Task
	// LIKE keeps SQLite/MySQL/PostgreSQL compatible without dialect JSON ops.
	err := model.DB.
		Where("status = ? AND platform = ?", model.TaskStatusSuccess, platform).
		Where(
			"private_data LIKE ? OR private_data LIKE ?",
			`%"storage_status":"pending"%`,
			`%"storage_status":"failed"%`,
		).
		Order("id").
		Limit(limit * 3).
		Find(&candidates).Error
	if err != nil {
		return nil, err
	}

	out := make([]*model.Task, 0, limit)
	for _, task := range candidates {
		status := task.PrivateData.StorageStatus
		if status != "pending" && status != "failed" {
			continue
		}
		if task.PrivateData.StorageRetryCount >= maxRetry {
			continue
		}
		if strings.TrimSpace(task.PrivateData.UpstreamResultURL) == "" {
			continue
		}
		out = append(out, task)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func defaultSilkRoadVideoFetch(url string) (io.ReadCloser, error) {
	resp, err := GetSSRFProtectedHTTPClient().Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("upstream video download status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// ingestOne downloads the upstream video into LocalDir and updates PrivateData.
// On failure it increments StorageRetryCount and sets pending/failed; it never refunds.
func ingestOne(task *model.Task, fetch silkRoadVideoFetchFunc) error {
	if task == nil {
		return fmt.Errorf("nil task")
	}
	upstreamURL := strings.TrimSpace(task.PrivateData.UpstreamResultURL)
	if upstreamURL == "" {
		return markSilkRoadIngestFailure(task, fmt.Errorf("missing upstream result url"))
	}
	if fetch == nil {
		fetch = defaultSilkRoadVideoFetch
	}

	body, err := fetch(upstreamURL)
	if err != nil {
		return markSilkRoadIngestFailure(task, err)
	}
	defer body.Close()

	path, _, err := WriteSilkRoadVideoFile(task.TaskID, body)
	if err != nil {
		return markSilkRoadIngestFailure(task, err)
	}

	retentionDays := silkroad_setting.GetSilkRoadSetting().Storage.RetentionDays
	if retentionDays < 1 {
		retentionDays = 1
	}
	task.PrivateData.StorageStatus = "ready"
	task.PrivateData.StoragePath = path
	task.PrivateData.StorageExpiresAt = time.Now().Unix() + int64(retentionDays)*86400
	if task.PrivateData.ResultURL == "" {
		task.PrivateData.ResultURL = BuildSilkRoadPublicURL(task.TaskID)
	}
	return nil
}

func markSilkRoadIngestFailure(task *model.Task, cause error) error {
	task.PrivateData.StorageRetryCount++
	maxRetry := silkroad_setting.GetSilkRoadSetting().Storage.MaxRetry
	if maxRetry < 1 {
		maxRetry = 1
	}
	if task.PrivateData.StorageRetryCount >= maxRetry {
		task.PrivateData.StorageStatus = "failed"
	} else {
		task.PrivateData.StorageStatus = "pending"
	}
	return cause
}
