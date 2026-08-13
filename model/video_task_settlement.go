package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var ErrVideoTaskNotSettling = errors.New("video task is not awaiting billing settlement")

type VideoTaskSettlementResult struct {
	Task           *Task
	PreviousQuota  int
	ActualQuota    int
	QuotaDelta     int
	AlreadyApplied bool
}

// SettleVideoTaskQuota applies an asynchronous video's billing delta exactly
// once. Funding, token quota, and the task's settlement marker are committed in
// one database transaction so a recovery pass cannot charge the same delta
// twice after a process interruption.
func SettleVideoTaskQuota(taskID int64, actualQuota int) (VideoTaskSettlementResult, error) {
	var result VideoTaskSettlementResult
	var userID int
	var tokenKey string

	if taskID <= 0 {
		return result, errors.New("invalid video task id")
	}
	if actualQuota < 0 || actualQuota > common.MaxQuota {
		return result, errors.New("invalid video settlement quota")
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		var task Task
		if err := lockForUpdate(tx).Where("id = ?", taskID).First(&task).Error; err != nil {
			return err
		}
		if task.Status != TaskStatusSettlementProcessing &&
			task.Status != TaskStatusSettlementRecovering {
			return ErrVideoTaskNotSettling
		}
		if task.PrivateData.BillingSettlementApplied {
			result = VideoTaskSettlementResult{
				Task:           &task,
				PreviousQuota:  task.Quota,
				ActualQuota:    task.Quota,
				AlreadyApplied: true,
			}
			return nil
		}
		if task.Quota < 0 || task.Quota > common.MaxQuota {
			return errors.New("invalid persisted video task quota")
		}

		previousQuota := task.Quota
		quotaDelta := actualQuota - previousQuota
		if quotaDelta != 0 {
			if task.PrivateData.BillingSource == "subscription" &&
				task.PrivateData.SubscriptionId > 0 {
				var subscription UserSubscription
				if err := lockForUpdate(tx).
					Where("id = ?", task.PrivateData.SubscriptionId).
					First(&subscription).Error; err != nil {
					return err
				}
				nextUsed := subscription.AmountUsed + int64(quotaDelta)
				if nextUsed < 0 {
					nextUsed = 0
				}
				if subscription.AmountTotal > 0 && nextUsed > subscription.AmountTotal {
					return fmt.Errorf(
						"subscription used exceeds total, used=%d total=%d",
						nextUsed,
						subscription.AmountTotal,
					)
				}
				subscription.AmountUsed = nextUsed
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
				user.Quota -= quotaDelta
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
					token.RemainQuota -= quotaDelta
					token.UsedQuota += quotaDelta
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
		}

		task.Quota = actualQuota
		task.PrivateData.BillingSettlementApplied = true
		task.PrivateData.StorageStatus = "settled"
		if err := tx.Select(
			"quota",
			"private_data",
			"updated_at",
		).Save(&task).Error; err != nil {
			return err
		}
		result = VideoTaskSettlementResult{
			Task:          &task,
			PreviousQuota: previousQuota,
			ActualQuota:   actualQuota,
			QuotaDelta:    quotaDelta,
		}
		return nil
	})
	if err != nil {
		return VideoTaskSettlementResult{}, err
	}
	if result.AlreadyApplied {
		return result, nil
	}
	if userID > 0 && result.QuotaDelta != 0 {
		if err := cacheIncrUserQuota(userID, int64(-result.QuotaDelta)); err != nil {
			common.SysError("failed to update settled user quota cache: " + err.Error())
		}
	}
	if tokenKey != "" && common.RedisEnabled && result.QuotaDelta != 0 {
		if err := cacheIncrTokenQuota(tokenKey, int64(-result.QuotaDelta)); err != nil {
			common.SysError("failed to update settled token quota cache: " + err.Error())
		}
	}
	return result, nil
}
