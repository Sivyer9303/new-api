package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var (
	ErrVideoTaskNotDeliveryFailure = errors.New("video task is not a refundable delivery failure")
)

type VideoTaskRefundResult struct {
	Task            *Task
	RefundedQuota   int
	AlreadyRefunded bool
}

func RefundVideoDeliveryFailure(
	taskID string,
	adminID int,
	reason string,
) (VideoTaskRefundResult, error) {
	var result VideoTaskRefundResult
	var userID int
	var tokenKey string

	err := DB.Transaction(func(tx *gorm.DB) error {
		var matches []Task
		if err := lockForUpdate(tx).
			Where("task_id = ?", taskID).
			Limit(2).
			Find(&matches).Error; err != nil {
			return err
		}
		if len(matches) == 0 {
			return gorm.ErrRecordNotFound
		}
		if len(matches) > 1 {
			return ErrAmbiguousTaskID
		}
		task := matches[0]
		if task.Status == TaskStatusRefunded || task.PrivateData.ManualRefundedAt > 0 {
			result = VideoTaskRefundResult{Task: &task, AlreadyRefunded: true}
			return nil
		}
		if task.Status != TaskStatusFailure ||
			task.PrivateData.StorageStatus != "failed" ||
			!task.PrivateData.NoAutomaticRefund ||
			task.Quota <= 0 {
			return ErrVideoTaskNotDeliveryFailure
		}

		quota := task.Quota
		if task.PrivateData.BillingSource == "subscription" &&
			task.PrivateData.SubscriptionId > 0 {
			var subscription UserSubscription
			if err := lockForUpdate(tx).
				Where("id = ?", task.PrivateData.SubscriptionId).
				First(&subscription).Error; err != nil {
				return err
			}
			subscription.AmountUsed -= int64(quota)
			if subscription.AmountUsed < 0 {
				subscription.AmountUsed = 0
			}
			if err := tx.Save(&subscription).Error; err != nil {
				return err
			}
		} else {
			var user User
			if err := lockForUpdate(tx).Where("id = ?", task.UserId).First(&user).Error; err != nil {
				return err
			}
			user.Quota += quota
			if err := tx.Select("quota").Save(&user).Error; err != nil {
				return err
			}
			userID = user.Id
		}

		if task.PrivateData.TokenId > 0 {
			var token Token
			if err := lockForUpdate(tx).
				Where("id = ?", task.PrivateData.TokenId).
				First(&token).Error; err != nil {
				return err
			}
			token.RemainQuota += quota
			token.UsedQuota -= quota
			if token.UsedQuota < 0 {
				token.UsedQuota = 0
			}
			if err := tx.Select("remain_quota", "used_quota").Save(&token).Error; err != nil {
				return err
			}
			tokenKey = token.Key
		}

		now := time.Now().Unix()
		task.Status = TaskStatusRefunded
		task.Progress = "100%"
		task.PrivateData.StorageStatus = "refunded"
		task.PrivateData.ManualRefundedAt = now
		task.PrivateData.ManualRefundAdmin = adminID
		task.PrivateData.ManualRefundReason = strings.TrimSpace(reason)
		task.PrivateData.ManualRefundQuota = quota
		task.PrivateData.ResultURL = ""
		task.PrivateData.UpstreamResultURL = ""
		task.Quota = 0
		if err := tx.Select(
			"status",
			"progress",
			"quota",
			"private_data",
			"updated_at",
		).Save(&task).Error; err != nil {
			return err
		}
		result = VideoTaskRefundResult{Task: &task, RefundedQuota: quota}
		return nil
	})
	if err != nil {
		return VideoTaskRefundResult{}, err
	}
	if result.AlreadyRefunded {
		return result, nil
	}
	if userID > 0 {
		if err := cacheIncrUserQuota(userID, int64(result.RefundedQuota)); err != nil {
			common.SysError("failed to update refunded user quota cache: " + err.Error())
		}
	}
	if tokenKey != "" && common.RedisEnabled {
		if err := cacheIncrTokenQuota(tokenKey, int64(result.RefundedQuota)); err != nil {
			common.SysError("failed to update refunded token quota cache: " + err.Error())
		}
	}
	if result.Task != nil {
		modelName := result.Task.Properties.OriginModelName
		if billing := result.Task.PrivateData.BillingContext; billing != nil &&
			billing.OriginModelName != "" {
			modelName = billing.OriginModelName
		}
		RecordTaskBillingLog(RecordTaskBillingLogParams{
			UserId:    result.Task.UserId,
			LogType:   LogTypeRefund,
			Content:   result.Task.PrivateData.ManualRefundReason,
			ChannelId: result.Task.ChannelId,
			ModelName: modelName,
			Quota:     result.RefundedQuota,
			TokenId:   result.Task.PrivateData.TokenId,
			Group:     result.Task.Group,
			NodeName:  result.Task.PrivateData.NodeName,
			Other: map[string]interface{}{
				"is_task":             true,
				"task_id":             result.Task.TaskID,
				"reason":              result.Task.PrivateData.ManualRefundReason,
				"manual_video_refund": true,
				"admin_id":            result.Task.PrivateData.ManualRefundAdmin,
			},
		})
	}
	return result, nil
}
