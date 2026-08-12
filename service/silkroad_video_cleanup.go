package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
)

const silkRoadVideoCleanupBatch = 20

// RunSilkRoadVideoCleanupOnce deletes local files past retention and marks tasks
// expired. No-op unless this process is the ingest node.
func RunVideoCleanupOnce(ctx context.Context) error {
	if !IsSilkRoadIngestNode() {
		return nil
	}
	storage := setting.GetEffectiveVideoSetting()
	if !storage.StorageEnabled {
		return nil
	}

	now := time.Now().Unix()
	tasks, err := claimSilkRoadExpiredVideoTasks(silkRoadVideoCleanupBatch, now)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := expireOneSilkRoadVideo(task); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf(
				"video cleanup delete failed task=%s: %s",
				task.TaskID, err.Error(),
			))
			continue
		}
		if _, err := task.UpdateWithStatus(model.TaskStatusStorageDeleting); err != nil {
			logger.LogError(ctx, fmt.Sprintf(
				"silkroad cleanup persist failed task=%s: %s",
				task.TaskID, err.Error(),
			))
		}
	}
	return nil
}

func RunSilkRoadVideoCleanupOnce(ctx context.Context) error {
	return RunVideoCleanupOnce(ctx)
}

func claimSilkRoadExpiredVideoTasks(limit int, now int64) ([]*model.Task, error) {
	if limit <= 0 {
		return nil, nil
	}
	out := make([]*model.Task, 0, limit)
	pageSize := limit * 3
	var afterID int64

	for len(out) < limit {
		var candidates []*model.Task
		// LIKE keeps SQLite/MySQL/PostgreSQL compatible without dialect JSON ops.
		likeExpr := silkRoadPrivateDataTextExpr()
		q := model.DB.
			Where(
				"(status = ? OR (status = ? AND updated_at < ?))",
				model.TaskStatusSuccess,
				model.TaskStatusStorageDeleting,
				time.Now().Add(-videoStorageClaimTimeout).Unix(),
			).
			Where(likeExpr+` LIKE ?`, `%"storage_status":"ready"%`).
			Order("id").
			Limit(pageSize)
		if afterID > 0 {
			q = q.Where("id > ?", afterID)
		}
		err := q.Find(&candidates).Error
		if err != nil {
			return nil, err
		}
		if len(candidates) == 0 {
			break
		}
		for _, task := range candidates {
			afterID = task.ID
			if task.PrivateData.StorageStatus != "ready" {
				continue
			}
			expiresAt := task.PrivateData.StorageExpiresAt
			if expiresAt <= 0 || expiresAt >= now {
				continue
			}
			if task.Status == model.TaskStatusSuccess {
				task.Status = model.TaskStatusStorageDeleting
				won, err := task.UpdateWithStatus(model.TaskStatusSuccess)
				if err != nil {
					return nil, err
				}
				if !won {
					continue
				}
			}
			out = append(out, task)
			if len(out) >= limit {
				break
			}
		}
		if len(candidates) < pageSize {
			break
		}
	}
	return out, nil
}

func expireOneSilkRoadVideo(task *model.Task) error {
	if task == nil {
		return nil
	}
	objectKey := strings.TrimSpace(task.PrivateData.StorageObjectKey)
	legacyPath := strings.TrimSpace(task.PrivateData.StoragePath)
	if objectKey == "" && legacyPath == "" {
		objectKey = strings.TrimSpace(task.TaskID)
	}
	if legacyPath != "" && task.PrivateData.StorageObjectKey == "" {
		if err := os.Remove(legacyPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete legacy stored video: %w", err)
		}
	} else if objectKey != "" {
		if err := DeleteSilkRoadVideoFile(objectKey); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete stored video: %w", err)
		}
	}
	task.Status = model.TaskStatusExpired
	task.Progress = "100%"
	task.FailReason = fmt.Sprintf(
		"Video expired after the configured %d-day retention period.",
		setting.GetEffectiveVideoSetting().Storage.RetentionDays(),
	)
	task.PrivateData.StorageStatus = "expired"
	task.PrivateData.ResultURL = ""
	task.PrivateData.UpstreamResultURL = ""
	task.PrivateData.StoragePath = ""
	task.PrivateData.StorageObjectKey = ""
	task.PrivateData.StorageContentType = ""
	task.PrivateData.StorageSize = 0
	return nil
}
