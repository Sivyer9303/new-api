package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/setting"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	silkRoadVideoIngestInterval = 30 * time.Second
	silkRoadVideoIngestBatch    = 20
	videoStorageClaimTimeout    = 10 * time.Minute
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
	return videoStorageWiringComplete()
}

// videoStorageWiringComplete reports whether storage is enabled and has every
// field required to deliver a stored result. R2 needs no designated ingest node.
func videoStorageWiringComplete() bool {
	storage := setting.GetEffectiveVideoSetting()
	if !storage.StorageEnabled {
		return false
	}
	if strings.TrimSpace(storage.Storage.PublicDownloadBaseURL) == "" {
		return false
	}
	if storage.Storage.IsR2() {
		return ValidateVideoR2StorageConfigured() == nil
	}
	return strings.TrimSpace(storage.Storage.IngestNodeName) != ""
}

// silkRoadNewAPIAvoidUpstreamResultURL is true when NewAPI storage is enabled
// but incomplete (missing ingest node and/or public base). Polling Success must
// use BuildProxyURL instead of the upstream ResultURL.
func silkRoadNewAPIAvoidUpstreamResultURL(task *model.Task) bool {
	if !isSilkRoadVideoTask(task) {
		return false
	}
	return !videoStorageWiringComplete()
}

func isSilkRoadVideoTask(task *model.Task) bool {
	if task == nil {
		return false
	}
	return task.Platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI)) ||
		task.Platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSilkRoad))
}

// markSilkRoadPendingStore records the upstream URL privately and points
// ResultURL at the public content path. Never puts the upstream URL in ResultURL.
func markSilkRoadPendingStore(task *model.Task, upstreamURL string) {
	task.PrivateData.UpstreamResultURL = upstreamURL
	task.PrivateData.StorageStatus = "pending"
	task.PrivateData.StorageRetryCount = 0
	task.PrivateData.StorageLastError = ""
	task.PrivateData.NoAutomaticRefund = false
	task.PrivateData.ResultURL = BuildSilkRoadPublicURL(task.TaskID)
	task.Status = model.TaskStatusStoring
	task.Progress = "99%"
}

// redactSilkRoadUpstreamURLs removes upstream CDN URL fields from task.Data JSON
// so TaskDto.Data cannot leak video_url / url / result_url (including nested maps).
func redactSilkRoadUpstreamURLs(data []byte, publicTaskID string) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	var v any
	if err := common.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	sanitizeSilkRoadPublicTaskData(v, publicTaskID)
	return common.Marshal(v)
}

// applySilkRoadDataRedaction redacts upstream URL fields from task.Data.
// On error it fail-closes to an empty JSON object so unredacted video_url cannot leak.
func applySilkRoadDataRedaction(data []byte, publicTaskID string) ([]byte, error) {
	redacted, err := redactSilkRoadUpstreamURLs(data, publicTaskID)
	if err != nil {
		return []byte("{}"), err
	}
	return redacted, nil
}

func sanitizeSilkRoadPublicTaskData(v any, publicTaskID string) {
	switch x := v.(type) {
	case map[string]any:
		delete(x, "video_url")
		delete(x, "url")
		delete(x, "result_url")
		for key, child := range x {
			if key == "id" || key == "task_id" {
				if publicTaskID == "" {
					delete(x, key)
				} else {
					x[key] = publicTaskID
				}
				continue
			}
			sanitizeSilkRoadPublicTaskData(child, publicTaskID)
		}
	case []any:
		for _, child := range x {
			sanitizeSilkRoadPublicTaskData(child, publicTaskID)
		}
	}
}

// StartSilkRoadVideoIngestTask starts the periodic ingest ticker once.
// Each tick (and the initial run) no-ops unless this process is the ingest
// node and storage is enabled, so late SyncOptions config takes effect without restart.
func StartVideoStorageTask() {
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
	if !setting.GetEffectiveVideoSetting().StorageEnabled {
		return
	}
	_ = RunVideoStorageOnce(context.Background())
	_ = RunVideoCleanupOnce(context.Background())
}

// RunSilkRoadVideoIngestOnce claims pending/failed SilkRoad video tasks and
// downloads them to local storage. No-op unless this process is the ingest node.
// Download failures never refund quota.
func RunVideoStorageOnce(ctx context.Context) error {
	if !IsSilkRoadIngestNode() {
		return nil
	}
	storage := setting.GetEffectiveVideoSetting()
	if !storage.StorageEnabled {
		return nil
	}

	tasks, err := claimSilkRoadIngestTasks(silkRoadVideoIngestBatch, storage.Storage.MaxRetry)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := ingestOne(task, nil); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf(
				"video ingest failed task=%s retry=%d",
				task.TaskID, task.PrivateData.StorageRetryCount,
			))
		}
		if _, err := task.UpdateWithStatus(model.TaskStatusStorageProcessing); err != nil {
			logger.LogError(ctx, fmt.Sprintf(
				"silkroad ingest persist failed task=%s: %s",
				task.TaskID, err.Error(),
			))
		}
	}
	return nil
}

func StartSilkRoadVideoIngestTask() {
	StartVideoStorageTask()
}

func RunSilkRoadVideoIngestOnce(ctx context.Context) error {
	return RunVideoStorageOnce(ctx)
}

func claimSilkRoadIngestTasks(limit int, maxRetry int) ([]*model.Task, error) {
	if limit <= 0 {
		return nil, nil
	}
	var candidates []*model.Task
	// LIKE keeps SQLite/MySQL/PostgreSQL compatible without dialect JSON ops.
	likeExpr := silkRoadPrivateDataTextExpr()
	err := model.DB.
		Where(
			"status IN ?",
			[]model.TaskStatus{model.TaskStatusStoring, model.TaskStatusSuccess},
		).
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
	if len(candidates) < limit*3 {
		var stale []*model.Task
		err = model.DB.
			Where("status = ? AND updated_at < ?", model.TaskStatusStorageProcessing, time.Now().Add(-videoStorageClaimTimeout).Unix()).
			Where(likeExpr+` LIKE ?`, `%"storage_status":"processing"%`).
			Order("id").
			Limit(limit * 3).
			Find(&stale).Error
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, stale...)
	}

	out := make([]*model.Task, 0, limit)
	for _, task := range candidates {
		status := task.PrivateData.StorageStatus
		if status != "pending" && status != "failed" && status != "processing" {
			continue
		}
		if task.PrivateData.StorageRetryCount >= maxRetry {
			continue
		}
		fromStatus := task.Status
		if fromStatus == model.TaskStatusStorageProcessing {
			task.Status = model.TaskStatusStoring
			task.PrivateData.StorageStatus = "pending"
			won, err := task.UpdateWithStatus(fromStatus)
			if err != nil {
				return nil, err
			}
			if !won {
				continue
			}
			fromStatus = model.TaskStatusStoring
		}
		task.Status = model.TaskStatusStorageProcessing
		task.PrivateData.StorageStatus = "processing"
		won, err := task.UpdateWithStatus(fromStatus)
		if err != nil {
			return nil, err
		}
		if !won {
			continue
		}
		out = append(out, task)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func openVideoDataURL(dataURL string) (io.ReadCloser, string, error) {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "data:") ||
		!strings.HasSuffix(parts[0], ";base64") {
		return nil, "", errors.New("invalid video data URL")
	}
	contentType := strings.TrimSuffix(strings.TrimPrefix(parts[0], "data:"), ";base64")
	payload, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.RawStdEncoding.DecodeString(parts[1])
	}
	if err != nil {
		return nil, "", errors.New("invalid video data URL payload")
	}
	return io.NopCloser(bytes.NewReader(payload)), contentType, nil
}

// ingestOne downloads the upstream video into LocalDir and updates PrivateData.
// On failure it increments StorageRetryCount and sets pending/failed; it never refunds.
func ingestOne(task *model.Task, fetch silkRoadVideoFetchFunc) error {
	if task == nil {
		return fmt.Errorf("nil task")
	}
	if blocked, reason := VideoStorageUploadBlocked(); blocked {
		return markVideoQuotaDeliveryFailure(task, reason)
	}
	upstreamURL := strings.TrimSpace(task.PrivateData.UpstreamResultURL)
	if upstreamURL == "" && fetch != nil {
		return markSilkRoadIngestFailure(task, fmt.Errorf("missing upstream result url"))
	}
	var body io.ReadCloser
	var contentType string
	var err error
	if strings.HasPrefix(upstreamURL, "data:") {
		body, contentType, err = openVideoDataURL(upstreamURL)
	} else if fetch == nil {
		source, sourceErr := openTaskVideoResultSource(context.Background(), task)
		if sourceErr != nil {
			err = sourceErr
		} else {
			body = source.Body
			contentType = source.ContentType
		}
	} else {
		body, err = fetch(upstreamURL)
	}
	if err != nil {
		return markSilkRoadIngestFailure(task, err)
	}
	defer body.Close()

	contentType, err = NormalizeStoredVideoContentType(contentType)
	if err != nil {
		return markSilkRoadIngestFailure(task, err)
	}
	storage := setting.GetEffectiveVideoSetting().Storage
	driver, err := NewVideoStorageDriver(storage)
	if err != nil {
		return markSilkRoadIngestFailure(task, err)
	}
	stored, err := driver.Store(
		context.Background(),
		task.TaskID,
		body,
		VideoObjectMetadata{ContentType: contentType},
	)
	if err != nil {
		return markSilkRoadIngestFailure(task, err)
	}
	// Object-storage drivers have no local path; only the local driver records one.
	path := ""
	if !storage.IsR2() {
		path, err = filepath.Abs(SilkRoadVideoLocalPath(stored.ObjectKey))
		if err != nil {
			return markSilkRoadIngestFailure(task, err)
		}
	}

	task.Status = model.TaskStatusSuccess
	task.Progress = taskcommon.ProgressComplete
	task.FinishTime = stored.ReadyAt
	task.PrivateData.StorageStatus = "ready"
	task.PrivateData.StoragePath = path
	task.PrivateData.StorageObjectKey = stored.ObjectKey
	task.PrivateData.StorageContentType = stored.ContentType
	task.PrivateData.StorageSize = stored.Size
	task.PrivateData.StorageReadyAt = stored.ReadyAt
	task.PrivateData.StorageExpiresAt = stored.ExpiresAt
	task.PrivateData.StorageLastError = ""
	task.PrivateData.UpstreamResultURL = ""
	if task.PrivateData.ResultURL == "" {
		task.PrivateData.ResultURL = BuildSilkRoadPublicURL(task.TaskID)
	}
	return nil
}

// markVideoQuotaDeliveryFailure ends the task immediately instead of burning
// retries: nothing can be uploaded until an administrator frees bucket space.
// The original charge stays, matching the existing delivery-failure policy.
func markVideoQuotaDeliveryFailure(task *model.Task, cause error) error {
	maxRetry := setting.GetEffectiveVideoSetting().Storage.MaxRetry
	if maxRetry < 1 {
		maxRetry = 1
	}
	task.PrivateData.StorageRetryCount = maxRetry
	task.PrivateData.StorageStatus = "failed"
	task.PrivateData.StorageLastError = cause.Error()
	task.PrivateData.NoAutomaticRefund = true
	task.Status = model.TaskStatusFailure
	task.Progress = taskcommon.ProgressComplete
	task.FinishTime = time.Now().Unix()
	task.FailReason = VideoDeliveryFailureMessage(task.TaskID)
	logger.LogError(context.Background(), fmt.Sprintf(
		"video storage upload refused task=%s: %s",
		task.TaskID, cause.Error(),
	))
	return cause
}

func markSilkRoadIngestFailure(task *model.Task, cause error) error {
	task.PrivateData.StorageRetryCount++
	maxRetry := setting.GetEffectiveVideoSetting().Storage.MaxRetry
	if maxRetry < 1 {
		maxRetry = 1
	}
	if task.PrivateData.StorageRetryCount >= maxRetry {
		task.PrivateData.StorageStatus = "failed"
		task.Status = model.TaskStatusFailure
		task.Progress = taskcommon.ProgressComplete
		task.FinishTime = time.Now().Unix()
		task.FailReason = VideoDeliveryFailureMessage(task.TaskID)
		task.PrivateData.NoAutomaticRefund = true
	} else {
		task.PrivateData.StorageStatus = "pending"
		task.Status = model.TaskStatusStoring
		task.Progress = "99%"
	}
	task.PrivateData.StorageLastError = cause.Error()
	return cause
}
