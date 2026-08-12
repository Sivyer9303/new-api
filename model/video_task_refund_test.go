package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVideoRefundTestDB(t *testing.T) {
	t.Helper()
	originalDB := DB
	originalLogDB := LOG_DB
	originalRedisEnabled := common.RedisEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&User{},
		&Token{},
		&UserSubscription{},
		&Task{},
		&Log{},
	))
	DB = db
	LOG_DB = db
	common.RedisEnabled = false
	t.Cleanup(func() {
		DB = originalDB
		LOG_DB = originalLogDB
		common.RedisEnabled = originalRedisEnabled
	})
}

func TestRefundVideoDeliveryFailureIsAtomicAndIdempotent(t *testing.T) {
	setupVideoRefundTestDB(t)

	user := User{Username: "video-refund-user", Password: "password", Quota: 100}
	require.NoError(t, DB.Create(&user).Error)
	token := Token{
		UserId:      user.Id,
		Key:         "video-refund-token",
		Name:        "video",
		RemainQuota: 50,
		UsedQuota:   90,
	}
	require.NoError(t, DB.Create(&token).Error)
	task := Task{
		TaskID: "task_video_refund",
		UserId: user.Id,
		Status: TaskStatusFailure,
		Quota:  40,
		PrivateData: TaskPrivateData{
			TokenId:           token.Id,
			StorageStatus:     "failed",
			StorageLastError:  "disk full",
			NoAutomaticRefund: true,
		},
	}
	require.NoError(t, DB.Create(&task).Error)

	first, err := RefundVideoDeliveryFailure(task.TaskID, 99, "confirmed delivery failure")
	require.NoError(t, err)
	assert.False(t, first.AlreadyRefunded)
	assert.Equal(t, 40, first.RefundedQuota)

	var storedUser User
	require.NoError(t, DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 140, storedUser.Quota)
	var storedToken Token
	require.NoError(t, DB.First(&storedToken, token.Id).Error)
	assert.Equal(t, 90, storedToken.RemainQuota)
	assert.Equal(t, 50, storedToken.UsedQuota)
	var storedTask Task
	require.NoError(t, DB.Where("task_id = ?", task.TaskID).First(&storedTask).Error)
	assert.Equal(t, TaskStatusRefunded, storedTask.Status)
	assert.Zero(t, storedTask.Quota)
	assert.Equal(t, "refunded", storedTask.PrivateData.StorageStatus)
	assert.Equal(t, 99, storedTask.PrivateData.ManualRefundAdmin)
	assert.Equal(t, 40, storedTask.PrivateData.ManualRefundQuota)
	var refundLogs int64
	require.NoError(t, LOG_DB.Model(&Log{}).
		Where("type = ?", LogTypeRefund).
		Count(&refundLogs).Error)
	assert.Equal(t, int64(1), refundLogs)

	second, err := RefundVideoDeliveryFailure(task.TaskID, 100, "duplicate request")
	require.NoError(t, err)
	assert.True(t, second.AlreadyRefunded)
	assert.Zero(t, second.RefundedQuota)
	require.NoError(t, DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 140, storedUser.Quota)
	require.NoError(t, DB.First(&storedToken, token.Id).Error)
	assert.Equal(t, 90, storedToken.RemainQuota)
	require.NoError(t, LOG_DB.Model(&Log{}).
		Where("type = ?", LogTypeRefund).
		Count(&refundLogs).Error)
	assert.Equal(t, int64(1), refundLogs)
}

func TestRefundVideoDeliveryFailureRestoresSubscriptionUsage(t *testing.T) {
	setupVideoRefundTestDB(t)

	user := User{Username: "video-subscription-user", Password: "password", Quota: 25}
	require.NoError(t, DB.Create(&user).Error)
	subscription := UserSubscription{
		UserId:      user.Id,
		AmountTotal: 1_000,
		AmountUsed:  120,
		Status:      "active",
	}
	require.NoError(t, DB.Create(&subscription).Error)
	task := Task{
		TaskID: "task_video_subscription_refund",
		UserId: user.Id,
		Status: TaskStatusFailure,
		Quota:  70,
		PrivateData: TaskPrivateData{
			BillingSource:     "subscription",
			SubscriptionId:    subscription.Id,
			StorageStatus:     "failed",
			NoAutomaticRefund: true,
		},
	}
	require.NoError(t, DB.Create(&task).Error)

	result, err := RefundVideoDeliveryFailure(task.TaskID, 99, "storage failure")
	require.NoError(t, err)
	assert.Equal(t, 70, result.RefundedQuota)

	var storedSubscription UserSubscription
	require.NoError(t, DB.First(&storedSubscription, subscription.Id).Error)
	assert.Equal(t, int64(50), storedSubscription.AmountUsed)
	var storedUser User
	require.NoError(t, DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 25, storedUser.Quota)
}
