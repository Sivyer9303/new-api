package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/setting"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	videoIngestInterval      = 30 * time.Second
	videoIngestBatch         = 20
	videoStorageClaimTimeout = 10 * time.Minute
)

// videoPrivateDataTextExpr is the SQL expression for LIKE on private_data
// without dialect-specific JSON operators.
func videoPrivateDataTextExpr() string {
	switch common.MainDatabaseType() {
	case common.DatabaseTypeSQLite:
		return "CAST(private_data AS TEXT)"
	case common.DatabaseTypePostgreSQL:
		return "private_data::text"
	default:
		return "private_data"
	}
}

var videoIngestOnce sync.Once

// videoFetchFunc downloads an upstream video body. Tests inject httptest.
type videoFetchFunc func(url string) (io.ReadCloser, error)

// shouldStoreVideoResult reports whether a completed task can enter the
// mandatory result-storage lifecycle.
// Requires Storage.Enabled plus non-empty ingest node name and public download
// base URL. When Enabled but ingest/public are incomplete, callers must fail
// closed (proxy URL) rather than writing the upstream CDN URL into ResultURL.
func shouldStoreVideoResult(task *model.Task) bool {
	if task == nil {
		return false
	}
	return videoStorageWiringComplete()
}

func shouldSilkRoadStore(task *model.Task) bool {
	return shouldStoreVideoResult(task)
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

// videoTaskAvoidUpstreamResultURL is true when mandatory video storage is
// incomplete. Polling success must never expose the upstream ResultURL.
func videoTaskAvoidUpstreamResultURL(task *model.Task) bool {
	if !IsVideoTask(task) {
		return false
	}
	return !videoStorageWiringComplete()
}

func silkRoadNewAPIAvoidUpstreamResultURL(task *model.Task) bool {
	return videoTaskAvoidUpstreamResultURL(task)
}

func isSilkRoadVideoTask(task *model.Task) bool {
	return IsVideoTask(task)
}

// markVideoPendingStore records the upstream URL privately and points
// ResultURL at the public content path. Never puts the upstream URL in ResultURL.
func markVideoPendingStore(task *model.Task, upstreamURL string) {
	task.PrivateData.UpstreamResultURL = upstreamURL
	task.PrivateData.StorageStatus = "pending"
	task.PrivateData.StorageRetryCount = 0
	task.PrivateData.StorageLastError = ""
	task.PrivateData.NoAutomaticRefund = false
	task.PrivateData.ResultURL = BuildVideoPublicURL(task.TaskID)
	task.Status = model.TaskStatusStoring
	task.Progress = "99%"
}

func markSilkRoadPendingStore(task *model.Task, upstreamURL string) {
	markVideoPendingStore(task, upstreamURL)
}

// redactVideoUpstreamURLs removes upstream CDN URL fields from task.Data JSON
// so TaskDto.Data cannot leak video_url / url / result_url (including nested maps).
func redactVideoUpstreamURLs(data []byte, publicTaskID string) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	var v any
	if err := common.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	sanitizePublicVideoTaskData(v, publicTaskID)
	return common.Marshal(v)
}

func redactSilkRoadUpstreamURLs(data []byte, publicTaskID string) ([]byte, error) {
	return redactVideoUpstreamURLs(data, publicTaskID)
}

// applyVideoDataRedaction redacts upstream URL fields from task.Data.
// On error it fail-closes to an empty JSON object so unredacted video_url cannot leak.
func applyVideoDataRedaction(data []byte, publicTaskID string) ([]byte, error) {
	redacted, err := redactVideoUpstreamURLs(data, publicTaskID)
	if err != nil {
		return []byte("{}"), err
	}
	return redacted, nil
}

func applySilkRoadDataRedaction(data []byte, publicTaskID string) ([]byte, error) {
	return applyVideoDataRedaction(data, publicTaskID)
}

func sanitizePublicVideoTaskData(v any, publicTaskID string) {
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
			if text, ok := child.(string); ok && containsPrivateVideoMedia(text) {
				x[key] = "[redacted]"
				continue
			}
			sanitizePublicVideoTaskData(child, publicTaskID)
		}
	case []any:
		for index, child := range x {
			if text, ok := child.(string); ok && containsPrivateVideoMedia(text) {
				x[index] = "[redacted]"
				continue
			}
			sanitizePublicVideoTaskData(child, publicTaskID)
		}
	}
}

func containsPrivateVideoMedia(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "http://") ||
		strings.Contains(lower, "https://") ||
		strings.Contains(lower, "data:image/") ||
		strings.Contains(lower, "data:audio/") ||
		strings.Contains(lower, "data:video/")
}

// StartVideoStorageTask starts the periodic ingest ticker once.
// Each tick (and the initial run) no-ops unless this process is the ingest
// node and storage is enabled, so late SyncOptions config takes effect without restart.
func StartVideoStorageTask() {
	videoIngestOnce.Do(func() {
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf(
				"video ingest ticker started: tick=%s",
				videoIngestInterval,
			))
			ticker := time.NewTicker(videoIngestInterval)
			defer ticker.Stop()

			runVideoIngestTick()
			for range ticker.C {
				runVideoIngestTick()
			}
		})
	})
}

func runVideoIngestTick() {
	if !IsVideoIngestNode() {
		return
	}
	if !setting.GetEffectiveVideoSetting().StorageEnabled {
		return
	}
	_ = RunVideoStorageOnce(context.Background())
	_ = RunVideoCleanupOnce(context.Background())
}

// RunVideoStorageOnce claims pending/failed video tasks and stores their result.
// It is a no-op unless this process is an ingest node.
// Download failures never refund quota.
func RunVideoStorageOnce(ctx context.Context) error {
	if !IsVideoIngestNode() {
		return nil
	}
	storage := setting.GetEffectiveVideoSetting()
	if !storage.StorageEnabled {
		return nil
	}

	if err := recoverStaleVideoSettlements(ctx, videoIngestBatch); err != nil {
		return err
	}
	tasks, err := claimVideoIngestTasks(videoIngestBatch, storage.Storage.MaxRetry)
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
		updated, err := task.UpdateWithStatus(model.TaskStatusStorageProcessing)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf(
				"video ingest persist failed task=%s: %s",
				task.TaskID, err.Error(),
			))
		} else if updated && task.Status == model.TaskStatusSuccess {
			logger.LogInfo(ctx, fmt.Sprintf(
				"task_lifecycle stage=storage_ready task_id=%s channel_id=%d",
				task.TaskID,
				task.ChannelId,
			))
		}
	}
	return nil
}

func recoverStaleVideoSettlements(ctx context.Context, limit int) error {
	if limit <= 0 {
		return nil
	}
	likeExpr := videoPrivateDataTextExpr()
	var candidates []*model.Task
	err := model.DB.
		Where("status IN ? AND updated_at < ?", []model.TaskStatus{
			model.TaskStatusSettlementProcessing,
			model.TaskStatusSettlementRecovering,
		}, time.Now().Add(-videoStorageClaimTimeout).Unix()).
		Where(
			likeExpr+` LIKE ? OR `+likeExpr+` LIKE ?`,
			`%"storage_status":"settling"%`,
			`%"storage_status":"settled"%`,
		).
		Order("id").
		Limit(limit).
		Find(&candidates).Error
	if err != nil {
		return err
	}

	for _, task := range candidates {
		fromStatus := task.Status
		won, err := task.ClaimWithStatusAndUpdatedAt(
			fromStatus,
			model.TaskStatusSettlementRecovering,
		)
		if err != nil {
			return err
		}
		if !won {
			continue
		}

		if !task.PrivateData.BillingSettlementApplied {
			if err := settleTaskBillingOnComplete(ctx, nil, task, nil); err != nil {
				expectedUpdatedAt := task.UpdatedAt
				task.Status = model.TaskStatusFailure
				task.Progress = taskcommon.ProgressComplete
				task.FinishTime = time.Now().Unix()
				task.FailReason = "Video billing settlement recovery requires administrator review"
				task.PrivateData.NoAutomaticRefund = true
				task.PrivateData.StorageStatus = model.TaskStorageStatusProviderReview
				task.PrivateData.StorageLastError = err.Error()
				if _, updateErr := task.UpdateWithStatusAndUpdatedAt(
					model.TaskStatusSettlementRecovering,
					expectedUpdatedAt,
				); updateErr != nil {
					return updateErr
				}
				continue
			}
		}

		expectedUpdatedAt := task.UpdatedAt
		task.Status = model.TaskStatusStoring
		task.PrivateData.StorageStatus = "pending"
		won, err = task.UpdateWithStatusAndUpdatedAt(
			model.TaskStatusSettlementRecovering,
			expectedUpdatedAt,
		)
		if err != nil {
			return err
		}
		if !won {
			continue
		}
		logger.LogInfo(ctx, fmt.Sprintf(
			"recovered video settlement for task %s",
			task.TaskID,
		))
	}
	return nil
}

func StartSilkRoadVideoIngestTask() {
	StartVideoStorageTask()
}

func RunSilkRoadVideoIngestOnce(ctx context.Context) error {
	return RunVideoStorageOnce(ctx)
}

func claimVideoIngestTasks(limit int, maxRetry int) ([]*model.Task, error) {
	if limit <= 0 {
		return nil, nil
	}
	var candidates []*model.Task
	// LIKE keeps SQLite/MySQL/PostgreSQL compatible without dialect JSON ops.
	likeExpr := videoPrivateDataTextExpr()
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

func claimSilkRoadIngestTasks(limit int, maxRetry int) ([]*model.Task, error) {
	return claimVideoIngestTasks(limit, maxRetry)
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
func ingestOne(task *model.Task, fetch videoFetchFunc) error {
	if task == nil {
		return fmt.Errorf("nil task")
	}
	if blocked, reason := VideoStorageUploadBlocked(); blocked {
		return markVideoQuotaDeliveryFailure(task, reason)
	}
	upstreamURL := strings.TrimSpace(task.PrivateData.UpstreamResultURL)
	if upstreamURL == "" && fetch != nil {
		return markVideoIngestFailure(task, fmt.Errorf("missing upstream result url"))
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
		return markVideoIngestFailure(task, err)
	}
	defer body.Close()

	contentType, err = NormalizeStoredVideoContentType(contentType)
	if err != nil {
		return markVideoIngestFailure(task, err)
	}
	storage := setting.GetEffectiveVideoSetting().Storage
	driver, err := NewVideoStorageDriver(storage)
	if err != nil {
		return markVideoIngestFailure(task, err)
	}
	stored, err := driver.Store(
		context.Background(),
		task.TaskID,
		body,
		VideoObjectMetadata{ContentType: contentType},
	)
	if err != nil {
		return markVideoIngestFailure(task, err)
	}
	// Object-storage drivers have no local path; only the local driver records one.
	path := ""
	if !storage.IsR2() {
		path, err = filepath.Abs(VideoLocalPath(stored.ObjectKey))
		if err != nil {
			return markVideoIngestFailure(task, err)
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
		task.PrivateData.ResultURL = BuildVideoPublicURL(task.TaskID)
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

func markVideoIngestFailure(task *model.Task, cause error) error {
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

func markSilkRoadIngestFailure(task *model.Task, cause error) error {
	return markVideoIngestFailure(task, cause)
}
