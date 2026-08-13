package model

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettleVideoTaskQuotaIsAtomicAndIdempotent(t *testing.T) {
	setupVideoRefundTestDB(t)

	user := User{Username: "video-settlement-user", Password: "password", Quota: 100}
	require.NoError(t, DB.Create(&user).Error)
	token := Token{
		UserId:      user.Id,
		Key:         "video-settlement-token",
		Name:        "video",
		RemainQuota: 50,
		UsedQuota:   90,
	}
	require.NoError(t, DB.Create(&token).Error)
	task := Task{
		TaskID: "task_video_settlement",
		UserId: user.Id,
		Status: TaskStatusSettlementProcessing,
		Quota:  40,
		PrivateData: TaskPrivateData{
			TokenId:       token.Id,
			StorageStatus: "settling",
		},
	}
	require.NoError(t, DB.Create(&task).Error)

	first, err := SettleVideoTaskQuota(task.ID, 70)
	require.NoError(t, err)
	assert.False(t, first.AlreadyApplied)
	assert.Equal(t, 30, first.QuotaDelta)

	var storedUser User
	require.NoError(t, DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 70, storedUser.Quota)
	var storedToken Token
	require.NoError(t, DB.First(&storedToken, token.Id).Error)
	assert.Equal(t, 20, storedToken.RemainQuota)
	assert.Equal(t, 120, storedToken.UsedQuota)
	var storedTask Task
	require.NoError(t, DB.First(&storedTask, task.ID).Error)
	assert.Equal(t, 70, storedTask.Quota)
	assert.True(t, storedTask.PrivateData.BillingSettlementApplied)
	assert.Equal(t, "settled", storedTask.PrivateData.StorageStatus)

	second, err := SettleVideoTaskQuota(task.ID, 70)
	require.NoError(t, err)
	assert.True(t, second.AlreadyApplied)
	require.NoError(t, DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 70, storedUser.Quota)
	require.NoError(t, DB.First(&storedToken, token.Id).Error)
	assert.Equal(t, 20, storedToken.RemainQuota)
}

func TestSettleVideoTaskQuotaRollsBackAllChanges(t *testing.T) {
	setupVideoRefundTestDB(t)

	user := User{Username: "video-settlement-sub", Password: "password", Quota: 25}
	require.NoError(t, DB.Create(&user).Error)
	subscription := UserSubscription{
		UserId:      user.Id,
		AmountTotal: 100,
		AmountUsed:  90,
		Status:      "active",
	}
	require.NoError(t, DB.Create(&subscription).Error)
	token := Token{
		UserId:      user.Id,
		Key:         "video-settlement-sub-token",
		Name:        "video",
		RemainQuota: 80,
		UsedQuota:   20,
	}
	require.NoError(t, DB.Create(&token).Error)
	task := Task{
		TaskID: "task_video_settlement_rollback",
		UserId: user.Id,
		Status: TaskStatusSettlementProcessing,
		Quota:  40,
		PrivateData: TaskPrivateData{
			BillingSource:  "subscription",
			SubscriptionId: subscription.Id,
			TokenId:        token.Id,
			StorageStatus:  "settling",
		},
	}
	require.NoError(t, DB.Create(&task).Error)

	_, err := SettleVideoTaskQuota(task.ID, 60)
	require.Error(t, err)

	var storedSubscription UserSubscription
	require.NoError(t, DB.First(&storedSubscription, subscription.Id).Error)
	assert.Equal(t, int64(90), storedSubscription.AmountUsed)
	var storedToken Token
	require.NoError(t, DB.First(&storedToken, token.Id).Error)
	assert.Equal(t, 80, storedToken.RemainQuota)
	assert.Equal(t, 20, storedToken.UsedQuota)
	var storedTask Task
	require.NoError(t, DB.First(&storedTask, task.ID).Error)
	assert.Equal(t, 40, storedTask.Quota)
	assert.False(t, storedTask.PrivateData.BillingSettlementApplied)
	assert.Equal(t, "settling", storedTask.PrivateData.StorageStatus)
}

func TestSettleVideoTaskQuotaConcurrentWorkersChargeOnce(t *testing.T) {
	setupVideoRefundTestDB(t)
	sqlDB, err := DB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	user := User{Username: "video-settlement-workers", Password: "password", Quota: 100}
	require.NoError(t, DB.Create(&user).Error)
	task := Task{
		TaskID: "task_video_settlement_workers",
		UserId: user.Id,
		Status: TaskStatusSettlementProcessing,
		Quota:  40,
		PrivateData: TaskPrivateData{
			StorageStatus: "settling",
		},
	}
	require.NoError(t, DB.Create(&task).Error)

	start := make(chan struct{})
	results := make(chan VideoTaskSettlementResult, 2)
	errors := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			result, settleErr := SettleVideoTaskQuota(task.ID, 70)
			results <- result
			errors <- settleErr
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errors)

	for settleErr := range errors {
		require.NoError(t, settleErr)
	}
	applied := 0
	alreadyApplied := 0
	for result := range results {
		if result.AlreadyApplied {
			alreadyApplied++
		} else {
			applied++
		}
	}
	assert.Equal(t, 1, applied)
	assert.Equal(t, 1, alreadyApplied)

	var storedUser User
	require.NoError(t, DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 70, storedUser.Quota)
}
