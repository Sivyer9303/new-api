package relay

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type recordingTaskBilling struct {
	reserved int
	err      error
}

type recordingTaskResponseBody struct {
	io.Reader
	closed bool
}

func (body *recordingTaskResponseBody) Close() error {
	body.closed = true
	return nil
}

func (billing *recordingTaskBilling) Settle(int) error         { return nil }
func (billing *recordingTaskBilling) Refund(*gin.Context)      {}
func (billing *recordingTaskBilling) NeedsRefund() bool        { return false }
func (billing *recordingTaskBilling) MarkPersistedRefund()     {}
func (billing *recordingTaskBilling) GetPreConsumedQuota() int { return 0 }
func (billing *recordingTaskBilling) Reserve(target int) error {
	billing.reserved = target
	return billing.err
}

func TestTaskBillingRatioAppliesSecondsOnlyForPerSecondModels(t *testing.T) {
	assert.False(t, taskBillingRatioApplies("seconds", billing_setting.BillingModeRatio))
	assert.False(t, taskBillingRatioApplies("seconds", billing_setting.BillingModeTieredExpr))
	assert.True(t, taskBillingRatioApplies("seconds", billing_setting.BillingModePerSecond))
	assert.True(t, taskBillingRatioApplies("resolution", billing_setting.BillingModeRatio))
}

func TestEnsureTaskQuotaReservedTopsUpExistingRetrySession(t *testing.T) {
	billing := &recordingTaskBilling{}
	info := &relaycommon.RelayInfo{Billing: billing}
	info.PriceData.Quota = 12_345

	require.Nil(t, ensureTaskQuotaReserved(nil, info))

	assert.True(t, info.ForcePreConsume)
	assert.Equal(t, 12_345, billing.reserved)
}

func TestEnsureTaskQuotaReservedReturnsLocalErrorBeforeRetrySubmission(t *testing.T) {
	billing := &recordingTaskBilling{err: errors.New("reserve failed")}
	info := &relaycommon.RelayInfo{Billing: billing}
	info.PriceData.Quota = 12_345

	taskErr := ensureTaskQuotaReserved(nil, info)

	require.NotNil(t, taskErr)
	assert.True(t, taskErr.LocalError)
	assert.Equal(t, "reserve_task_quota_failed", taskErr.Code)
}

func TestValidateTaskSubmitHTTPResponseAcceptsAll2xxAndClosesErrors(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			body := &recordingTaskResponseBody{Reader: strings.NewReader(`{"id":"task"}`)}
			taskErr := validateTaskSubmitHTTPResponse(&http.Response{
				StatusCode: status,
				Body:       body,
			})
			require.Nil(t, taskErr)
			assert.False(t, body.closed)
		})
	}

	body := &recordingTaskResponseBody{
		Reader: strings.NewReader(`{"error":"provider-secret-body"}`),
	}
	taskErr := validateTaskSubmitHTTPResponse(&http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       body,
	})
	require.NotNil(t, taskErr)
	assert.True(t, body.closed)
	assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	assert.NotContains(t, taskErr.Message, "provider-secret-body")
}

func TestAmbiguousTaskSubmitStatusesAreNeverRetriedOrRefunded(t *testing.T) {
	for _, status := range []int{
		http.StatusRequestTimeout,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
	} {
		assert.True(t, isAmbiguousTaskSubmitStatus(status), status)
	}
	for _, status := range []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusTooManyRequests,
	} {
		assert.False(t, isAmbiguousTaskSubmitStatus(status), status)
	}
}

func TestPersistSubmittingTaskCreatesOneDurableRetryRecord(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:relay-task-submit?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	info := &relaycommon.RelayInfo{
		UserId:          7,
		UsingGroup:      "video",
		OriginModelName: "seedance-public",
		RelayMode:       relayconstant.RelayModeVideoSubmit,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeBrioi,
			ChannelId:         11,
			ApiKey:            "selected-key",
			UpstreamModelName: "seedance-upstream",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action:       constant.TaskActionGenerate,
			PublicTaskID: "task_public_durable",
		},
	}
	info.PriceData.ModelPrice = 1.25
	info.PriceData.ModelRatio = 2
	info.PriceData.GroupRatioInfo.GroupRatio = 1.5

	first, err := persistSubmittingTask(
		context,
		info,
		constant.TaskPlatform("62"),
		123,
	)
	require.NoError(t, err)
	assert.NotZero(t, first.ID)
	assert.Equal(t, model.TaskStatusSubmitting, first.Status)
	assert.True(t, first.PrivateData.VideoTask)
	assert.Equal(t, "selected-key", first.PrivateData.Key)
	assert.Equal(t, 123, first.Quota)
	assert.Equal(t, 123, first.PrivateData.BillingReservationTarget)

	info.ChannelId = 12
	info.ChannelMeta.ChannelId = 12
	second, err := persistSubmittingTask(
		context,
		info,
		constant.TaskPlatform("62"),
		456,
	)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, 12, second.ChannelId)
	assert.Equal(t, 456, second.Quota)
	assert.Equal(t, 456, second.PrivateData.BillingReservationTarget)

	var count int64
	require.NoError(t, db.Model(&model.Task{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestPersistSubmittingTaskReturnsDatabaseFailureBeforeUpstreamSubmission(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:relay-task-submit-failure?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	forcedErr := errors.New("forced submitting task insert failure")
	require.NoError(t, db.Callback().Create().Before("gorm:create").
		Register("test:fail_submitting_task_insert", func(tx *gorm.DB) {
			if tx.Statement.Table == "tasks" {
				tx.AddError(forcedErr)
			}
		}))
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove("test:fail_submitting_task_insert")
	})

	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	info := &relaycommon.RelayInfo{
		UserId:          7,
		UsingGroup:      "video",
		OriginModelName: "seedance-public",
		RelayMode:       relayconstant.RelayModeVideoSubmit,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeBrioi,
			ChannelId:         11,
			ApiKey:            "selected-key",
			UpstreamModelName: "seedance-upstream",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action:       constant.TaskActionGenerate,
			PublicTaskID: "task_public_insert_failure",
		},
	}

	_, err = persistSubmittingTask(
		context,
		info,
		constant.TaskPlatform("62"),
		123,
	)
	require.ErrorIs(t, err, forcedErr)
	var count int64
	require.NoError(t, db.Model(&model.Task{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestPersistProviderAcceptanceLeavesSubmittingRecordRecoverableOnDatabaseFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:relay-task-acceptance-failure?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	task := &model.Task{
		TaskID: "task_public_acceptance_failure",
		Status: model.TaskStatusSubmitting,
		Quota:  100,
	}
	require.NoError(t, db.Create(task).Error)

	forcedErr := errors.New("forced provider acceptance update failure")
	require.NoError(t, db.Callback().Update().Before("gorm:update").
		Register("test:fail_provider_acceptance_update", func(tx *gorm.DB) {
			if tx.Statement.Table == "tasks" {
				tx.AddError(forcedErr)
			}
		}))
	t.Cleanup(func() {
		_ = db.Callback().Update().Remove("test:fail_provider_acceptance_update")
	})

	updated, err := persistProviderAcceptance(task, &channel.TaskSubmitResponse{
		UpstreamTaskID: "upstream-accepted",
		TaskData:       []byte(`{"id":"public"}`),
		ResponseData:   []byte(`{"id":"public"}`),
	})
	require.ErrorIs(t, err, forcedErr)
	assert.False(t, updated)

	var stored model.Task
	require.NoError(t, db.First(&stored, task.ID).Error)
	assert.Equal(t, model.TaskStatusSubmitting, stored.Status)
	assert.Empty(t, stored.PrivateData.UpstreamTaskID)
	assert.False(t, stored.PrivateData.NoAutomaticRefund)
	assert.Equal(t, 100, stored.Quota)
}
