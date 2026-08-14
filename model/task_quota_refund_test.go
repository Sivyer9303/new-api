package model

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefundTaskQuotaIsAtomicAndIdempotent(t *testing.T) {
	setupVideoRefundTestDB(t)

	user := User{Username: "task-refund-user", Password: "password", Quota: 100}
	require.NoError(t, DB.Create(&user).Error)
	token := Token{
		UserId:      user.Id,
		Key:         "task-refund-token",
		Name:        "video",
		RemainQuota: 50,
		UsedQuota:   90,
	}
	require.NoError(t, DB.Create(&token).Error)
	task := Task{
		TaskID:        "task_atomic_refund",
		UserId:        user.Id,
		Status:        TaskStatusFailure,
		Quota:         40,
		RefundPending: true,
		PrivateData: TaskPrivateData{
			TokenId:             token.Id,
			BillingRefundReason: "provider failed",
		},
	}
	require.NoError(t, DB.Create(&task).Error)

	first, err := RefundTaskQuota(task.ID)
	require.NoError(t, err)
	assert.False(t, first.AlreadyRefunded)
	assert.Equal(t, 40, first.RefundedQuota)

	second, err := RefundTaskQuota(task.ID)
	require.NoError(t, err)
	assert.True(t, second.AlreadyRefunded)

	var storedUser User
	require.NoError(t, DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 140, storedUser.Quota)
	var storedToken Token
	require.NoError(t, DB.First(&storedToken, token.Id).Error)
	assert.Equal(t, 90, storedToken.RemainQuota)
	assert.Equal(t, 50, storedToken.UsedQuota)
	var storedTask Task
	require.NoError(t, DB.First(&storedTask, task.ID).Error)
	assert.Zero(t, storedTask.Quota)
	assert.False(t, storedTask.RefundPending)
	assert.Empty(t, storedTask.PrivateData.BillingRefundReason)
}

func TestRefundTaskQuotaConcurrentWorkersRefundOnce(t *testing.T) {
	setupVideoRefundTestDB(t)
	sqlDB, err := DB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	user := User{Username: "task-refund-workers", Password: "password", Quota: 100}
	require.NoError(t, DB.Create(&user).Error)
	task := Task{
		TaskID: "task_refund_workers",
		UserId: user.Id,
		Status: TaskStatusFailure,
		Quota:  40,
	}
	require.NoError(t, DB.Create(&task).Error)

	start := make(chan struct{})
	results := make(chan TaskQuotaRefundResult, 2)
	errors := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			result, refundErr := RefundTaskQuota(task.ID)
			results <- result
			errors <- refundErr
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errors)

	for refundErr := range errors {
		require.NoError(t, refundErr)
	}
	applied := 0
	alreadyApplied := 0
	for result := range results {
		if result.AlreadyRefunded {
			alreadyApplied++
		} else {
			applied++
		}
	}
	assert.Equal(t, 1, applied)
	assert.Equal(t, 1, alreadyApplied)

	var storedUser User
	require.NoError(t, DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 140, storedUser.Quota)
}

func TestRefundTaskQuotaRollsBackWhenSubscriptionIsMissing(t *testing.T) {
	setupVideoRefundTestDB(t)

	user := User{Username: "task-refund-rollback", Password: "password", Quota: 100}
	require.NoError(t, DB.Create(&user).Error)
	task := Task{
		TaskID: "task_refund_rollback",
		UserId: user.Id,
		Status: TaskStatusFailure,
		Quota:  40,
		PrivateData: TaskPrivateData{
			BillingSource:  "subscription",
			SubscriptionId: 9999,
		},
	}
	require.NoError(t, DB.Create(&task).Error)

	_, err := RefundTaskQuota(task.ID)
	require.Error(t, err)

	var storedUser User
	require.NoError(t, DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 100, storedUser.Quota)
	var storedTask Task
	require.NoError(t, DB.First(&storedTask, task.ID).Error)
	assert.Equal(t, 40, storedTask.Quota)
}

func TestDecreaseTokenQuotaDirectWritesWhenBatchUpdatesAreEnabled(t *testing.T) {
	setupVideoRefundTestDB(t)
	previousBatch := common.BatchUpdateEnabled
	common.BatchUpdateEnabled = true
	t.Cleanup(func() { common.BatchUpdateEnabled = previousBatch })

	token := Token{
		UserId:      1,
		Key:         "batch-token",
		Name:        "video",
		RemainQuota: 100,
		UsedQuota:   0,
	}
	require.NoError(t, DB.Create(&token).Error)

	require.NoError(t, DecreaseTokenQuota(token.Id, token.Key, 10))
	var stored Token
	require.NoError(t, DB.First(&stored, token.Id).Error)
	assert.Equal(t, 100, stored.RemainQuota)
	assert.Zero(t, stored.UsedQuota)

	require.NoError(t, DecreaseTokenQuotaDirect(token.Id, token.Key, 10))
	require.NoError(t, DB.First(&stored, token.Id).Error)
	assert.Equal(t, 90, stored.RemainQuota)
	assert.Equal(t, 10, stored.UsedQuota)
}

func TestTaskAutoMigrateAddsRefundRecoveryColumns(t *testing.T) {
	setupVideoRefundTestDB(t)
	assert.True(t, DB.Migrator().HasColumn(&Task{}, "refund_pending"))
	assert.True(t, DB.Migrator().HasColumn(&Task{}, "refund_retry_at"))
	assert.True(t, DB.Migrator().HasColumn(&Task{}, "refund_attempts"))
}
