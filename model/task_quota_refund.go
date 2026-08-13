package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type TaskQuotaRefundResult struct {
	Task            *Task
	RefundedQuota   int
	AlreadyRefunded bool
}

// RefundTaskQuota atomically restores a persisted task's funding and token
// quota, then clears the task quota marker. The row lock and zero marker make
// retries and concurrent refund workers idempotent.
func RefundTaskQuota(taskID int64) (TaskQuotaRefundResult, error) {
	var result TaskQuotaRefundResult
	var userID int
	var tokenKey string

	if taskID <= 0 {
		return result, errors.New("invalid task id")
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		var task Task
		if err := lockForUpdate(tx).Where("id = ?", taskID).First(&task).Error; err != nil {
			return err
		}
		if task.Quota == 0 {
			result = TaskQuotaRefundResult{
				Task:            &task,
				AlreadyRefunded: true,
			}
			return nil
		}
		if task.Quota < 0 || task.Quota > common.MaxQuota {
			return errors.New("invalid persisted task refund quota")
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
			if err := tx.Select("amount_used").Save(&subscription).Error; err != nil {
				return err
			}
		} else {
			var user User
			if err := lockForUpdate(tx).
				Where("id = ?", task.UserId).
				First(&user).Error; err != nil {
				return err
			}
			nextQuota := int64(user.Quota) + int64(quota)
			if nextQuota > int64(common.MaxQuota) {
				return fmt.Errorf("refunded user quota exceeds maximum: %d", nextQuota)
			}
			user.Quota = int(nextQuota)
			if err := tx.Select("quota").Save(&user).Error; err != nil {
				return err
			}
			userID = user.Id
		}

		if task.PrivateData.TokenId > 0 {
			var token Token
			err := lockForUpdate(tx).
				Where("id = ?", task.PrivateData.TokenId).
				First(&token).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err == nil {
				nextRemainQuota := int64(token.RemainQuota) + int64(quota)
				if nextRemainQuota > int64(common.MaxQuota) {
					return fmt.Errorf(
						"refunded token quota exceeds maximum: %d",
						nextRemainQuota,
					)
				}
				token.RemainQuota = int(nextRemainQuota)
				token.UsedQuota -= quota
				if token.UsedQuota < 0 {
					token.UsedQuota = 0
				}
				if err := tx.Select(
					"remain_quota",
					"used_quota",
				).Save(&token).Error; err != nil {
					return err
				}
				tokenKey = token.Key
			}
		}

		task.Quota = 0
		task.RefundPending = false
		task.RefundRetryAt = 0
		task.RefundAttempts = 0
		task.PrivateData.BillingRefundReason = ""
		if err := tx.Select(
			"quota",
			"refund_pending",
			"refund_retry_at",
			"refund_attempts",
			"private_data",
			"updated_at",
		).Save(&task).Error; err != nil {
			return err
		}
		result = TaskQuotaRefundResult{
			Task:          &task,
			RefundedQuota: quota,
		}
		return nil
	})
	if err != nil {
		return TaskQuotaRefundResult{}, err
	}
	if result.AlreadyRefunded {
		return result, nil
	}
	if userID > 0 && result.RefundedQuota > 0 {
		if err := cacheIncrUserQuota(userID, int64(result.RefundedQuota)); err != nil {
			common.SysError("failed to update refunded user quota cache: " + err.Error())
		}
	}
	if tokenKey != "" && common.RedisEnabled && result.RefundedQuota > 0 {
		if err := cacheIncrTokenQuota(tokenKey, int64(result.RefundedQuota)); err != nil {
			common.SysError("failed to update refunded token quota cache: " + err.Error())
		}
	}
	return result, nil
}
