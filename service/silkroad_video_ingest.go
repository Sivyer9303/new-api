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

// silkRoadPrivateDataTextExpr is the SQL expression for LIKE on private_data
// without dialect-specific JSON operators (see claimSilkRoadIngestTasks).
func silkRoadPrivateDataTextExpr() string {
	switch common.MainDatabaseType() {
	case common.DatabaseTypeSQLite:
		return "CAST(private_data AS TEXT)"
	case common.DatabaseTypePostgreSQL:
		return "private_data::text"
	default:
		return "private_data"
	}
}

var silkRoadVideoIngestOnce sync.Once

// silkRoadVideoFetchFunc downloads an upstream video body. Tests inject httptest.
type silkRoadVideoFetchFunc func(url string) (io.ReadCloser, error)

// shouldSilkRoadStore reports whether a completed task should be queued for
// local SilkRoad video ingest instead of exposing the upstream ResultURL.
// Requires Storage.Enabled plus non-empty ingest node name and public download
// base URL. When Enabled but ingest/public are incomplete, callers must fail
// closed (proxy URL) rather than writing the upstream CDN URL into ResultURL.
func shouldSilkRoadStore(task *model.Task) bool {
	if task == nil {
		return false
	}
	storage := silkroad_setting.GetSilkRoadSetting().Storage
	if !storage.Enabled {
		return false
	}
	if strings.TrimSpace(storage.IngestNodeName) == "" {
		return false
	}
	if strings.TrimSpace(storage.PublicDownloadBaseURL) == "" {
		return false
	}
	return task.Platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI))
}

// silkRoadNewAPIAvoidUpstreamResultURL is true when NewAPI storage is enabled
// but incomplete (missing ingest node and/or public base). Polling Success must
// use BuildProxyURL instead of the upstream ResultURL.
func silkRoadNewAPIAvoidUpstreamResultURL(task *model.Task) bool {
	if task == nil {
		return false
	}
	if task.Platform != constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI)) {
		return false
	}
	storage := silkroad_setting.GetSilkRoadSetting().Storage
	if !storage.Enabled {
		return false
	}
	return strings.TrimSpace(storage.IngestNodeName) == "" ||
		strings.TrimSpace(storage.PublicDownloadBaseURL) == ""
}

// markSilkRoadPendingStore records the upstream URL privately and points
// ResultURL at the public content path. Never puts the upstream URL in ResultURL.
func markSilkRoadPendingStore(task *model.Task, upstreamURL string) {
	task.PrivateData.UpstreamResultURL = upstreamURL
	task.PrivateData.StorageStatus = "pending"
	task.PrivateData.StorageRetryCount = 0
	task.PrivateData.ResultURL = BuildSilkRoadPublicURL(task.TaskID)
}

// redactSilkRoadUpstreamURLs removes upstream CDN URL fields from task.Data JSON
// so TaskDto.Data cannot leak video_url / url / result_url (including nested maps).
func redactSilkRoadUpstreamURLs(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	var v any
	if err := common.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	stripSilkRoadUpstreamURLFields(v)
	return common.Marshal(v)
}

// applySilkRoadDataRedaction redacts upstream URL fields from task.Data.
// On error it fail-closes to an empty JSON object so unredacted video_url cannot leak.
func applySilkRoadDataRedaction(data []byte) ([]byte, error) {
	redacted, err := redactSilkRoadUpstreamURLs(data)
	if err != nil {
		return []byte("{}"), err
	}
	return redacted, nil
}

func stripSilkRoadUpstreamURLFields(v any) {
	switch x := v.(type) {
	case map[string]any:
		delete(x, "video_url")
		delete(x, "url")
		delete(x, "result_url")
		for _, child := range x {
			stripSilkRoadUpstreamURLFields(child)
		}
	case []any:
		for _, child := range x {
			stripSilkRoadUpstreamURLFields(child)
		}
	}
}

// StartSilkRoadVideoIngestTask starts the periodic ingest ticker once.
// Each tick (and the initial run) no-ops unless this process is the ingest
// node and storage is enabled, so late SyncOptions config takes effect without restart.
func StartSilkRoadVideoIngestTask() {
	silkRoadVideoIngestOnce.Do(func() {
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf(
				"silkroad video ingest ticker started: tick=%s",
				silkRoadVideoIngestInterval,
			))
			ticker := time.NewTicker(silkRoadVideoIngestInterval)
			defer ticker.Stop()

			runSilkRoadVideoIngestTick()
			for range ticker.C {
				runSilkRoadVideoIngestTick()
			}
		})
	})
}

func runSilkRoadVideoIngestTick() {
	if !IsSilkRoadIngestNode() {
		return
	}
	if !silkroad_setting.GetSilkRoadSetting().Storage.Enabled {
		return
	}
	_ = RunSilkRoadVideoIngestOnce(context.Background())
	_ = RunSilkRoadVideoCleanupOnce(context.Background())
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
	likeExpr := silkRoadPrivateDataTextExpr()
	err := model.DB.
		Where("status = ? AND platform = ?", model.TaskStatusSuccess, platform).
		Where(
			likeExpr+` LIKE ? OR `+likeExpr+` LIKE ?`,
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
