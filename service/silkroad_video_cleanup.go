package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
)

const silkRoadVideoCleanupBatch = 20

// RunSilkRoadVideoCleanupOnce deletes local files past retention and marks tasks
// expired. No-op unless this process is the ingest node.
func RunSilkRoadVideoCleanupOnce(ctx context.Context) error {
	if !IsSilkRoadIngestNode() {
		return nil
	}
	storage := silkroad_setting.GetSilkRoadSetting().Storage
	if !storage.Enabled {
		return nil
	}

	now := time.Now().Unix()
	tasks, err := claimSilkRoadExpiredVideoTasks(silkRoadVideoCleanupBatch, now)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		expireOneSilkRoadVideo(task)
		if err := task.Update(); err != nil {
			logger.LogError(ctx, fmt.Sprintf(
				"silkroad cleanup persist failed task=%s: %s",
				task.TaskID, err.Error(),
			))
		}
	}
	return nil
}

func claimSilkRoadExpiredVideoTasks(limit int, now int64) ([]*model.Task, error) {
	if limit <= 0 {
		return nil, nil
	}
	platform := strconv.Itoa(constant.ChannelTypeNewAPI)
	out := make([]*model.Task, 0, limit)
	pageSize := limit * 3
	var afterID int64

	for len(out) < limit {
		var candidates []*model.Task
		// LIKE keeps SQLite/MySQL/PostgreSQL compatible without dialect JSON ops.
		likeExpr := silkRoadPrivateDataTextExpr()
		q := model.DB.
			Where("status = ? AND platform = ?", model.TaskStatusSuccess, platform).
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

func expireOneSilkRoadVideo(task *model.Task) {
	if task == nil {
		return
	}
	taskID := strings.TrimSpace(task.TaskID)
	if taskID != "" {
		if err := DeleteSilkRoadVideoFile(taskID); err != nil && !os.IsNotExist(err) {
			logger.LogWarn(context.Background(), fmt.Sprintf(
				"silkroad cleanup delete file failed task=%s: %s",
				taskID, err.Error(),
			))
		}
	}
	task.PrivateData.StorageStatus = "expired"
}
