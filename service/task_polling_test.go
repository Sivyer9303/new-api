package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type taskPollingFetchAdaptor struct {
	mu           sync.Mutex
	taskIDs      []string
	fetched      chan string
	blockTaskID  string
	blockStarted chan struct{}
	releaseBlock chan struct{}
	blockOnce    sync.Once
}

type sunoFailurePollingAdaptor struct {
	failReason string
}

type trackingPollingBody struct {
	*bytes.Reader
	closed bool
}

func (b *trackingPollingBody) Close() error {
	b.closed = true
	return nil
}

func TestReadTaskPollingResponseIsBoundedAndAlwaysClosed(t *testing.T) {
	body := &trackingPollingBody{Reader: bytes.NewReader([]byte(`{"status":"queued"}`))}
	data, err := readTaskPollingResponse(&http.Response{Body: body})
	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"queued"}`, string(data))
	assert.True(t, body.closed)

	oversized := &trackingPollingBody{
		Reader: bytes.NewReader(bytes.Repeat(
			[]byte{'x'},
			maxTaskPollingResponseSize+1,
		)),
	}
	_, err = readTaskPollingResponse(&http.Response{Body: oversized})
	require.Error(t, err)
	assert.True(t, oversized.closed)

	_, err = readTaskPollingResponse(nil)
	require.Error(t, err)
}

func TestRecoverPendingTaskRefundRetriesAfterDatabaseFailure(t *testing.T) {
	truncate(t)
	const userID = 806
	seedUser(t, userID, 1_000)
	task := &model.Task{
		TaskID:        "pending_refund_retry",
		UserId:        userID,
		Status:        model.TaskStatusFailure,
		Quota:         500,
		Progress:      taskcommon.ProgressComplete,
		RefundPending: true,
		PrivateData: model.TaskPrivateData{
			BillingRefundReason: "provider failed",
		},
	}
	require.NoError(t, model.DB.Create(task).Error)

	forcedErr := errors.New("forced refund update failure")
	require.NoError(t, model.DB.Callback().Update().Before("gorm:update").
		Register("test:fail_pending_refund_once", func(tx *gorm.DB) {
			if tx.Statement.Table == "users" {
				tx.AddError(forcedErr)
			}
		}))
	t.Cleanup(func() {
		_ = model.DB.Callback().Update().Remove("test:fail_pending_refund_once")
	})
	require.NoError(t, recoverPendingTaskRefunds(context.Background(), 10))

	var stored model.Task
	require.NoError(t, model.DB.First(&stored, task.ID).Error)
	assert.Equal(t, 500, stored.Quota)
	assert.True(t, stored.RefundPending)
	assert.Equal(t, 1, stored.RefundAttempts)
	assert.Greater(t, stored.RefundRetryAt, time.Now().Unix())
	assert.Equal(t, 1_000, getUserQuota(t, userID))
	assert.False(t, HasPendingTaskRefunds())

	require.NoError(t, model.DB.Callback().Update().Remove("test:fail_pending_refund_once"))
	require.NoError(t, model.DB.Model(&model.Task{}).
		Where("id = ?", task.ID).
		Update("refund_retry_at", 0).Error)
	require.NoError(t, recoverPendingTaskRefunds(context.Background(), 10))

	stored = model.Task{}
	require.NoError(t, model.DB.First(&stored, task.ID).Error)
	assert.Zero(t, stored.Quota)
	assert.False(t, stored.RefundPending)
	assert.Equal(t, 1_500, getUserQuota(t, userID))
	assert.False(t, HasPendingTaskRefunds())
}

func TestRecoverPendingTaskRefundRotatesFailedRows(t *testing.T) {
	truncate(t)
	const userID = 807
	seedUser(t, userID, 1_000)
	taskIDs := make([]int64, 0, 3)
	for index := range 3 {
		task := &model.Task{
			TaskID:        fmt.Sprintf("poison_refund_%d", index),
			UserId:        userID,
			Status:        model.TaskStatusFailure,
			Quota:         100,
			RefundPending: true,
			PrivateData: model.TaskPrivateData{
				BillingSource:       BillingSourceSubscription,
				SubscriptionId:      90_000 + index,
				BillingRefundReason: "subscription disappeared",
			},
		}
		require.NoError(t, model.DB.Create(task).Error)
		taskIDs = append(taskIDs, task.ID)
	}

	require.NoError(t, recoverPendingTaskRefunds(context.Background(), 2))
	require.NoError(t, recoverPendingTaskRefunds(context.Background(), 2))

	var tasks []model.Task
	require.NoError(t, model.DB.Where("id IN ?", taskIDs).Order("id").Find(&tasks).Error)
	require.Len(t, tasks, 3)
	for _, task := range tasks {
		assert.Equal(t, 1, task.RefundAttempts)
		assert.Greater(t, task.RefundRetryAt, time.Now().Unix())
		assert.True(t, task.RefundPending)
	}
}

func (a *sunoFailurePollingAdaptor) Init(_ *relaycommon.RelayInfo) {}

func (a *sunoFailurePollingAdaptor) FetchTask(_ string, _ string, body map[string]any, _ string) (*http.Response, error) {
	taskIDs, _ := body["ids"].([]string)
	items := make([]taskdto.SunoDataResponse, 0, len(taskIDs))
	for _, taskID := range taskIDs {
		items = append(items, taskdto.SunoDataResponse{
			TaskID:     taskID,
			Status:     string(model.TaskStatusFailure),
			FailReason: a.failReason,
			FinishTime: time.Now().Unix(),
		})
	}

	responseBody, err := common.Marshal(taskdto.TaskResponse[[]taskdto.SunoDataResponse]{
		Code: taskdto.TaskSuccessCode,
		Data: items,
	})
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(responseBody)),
	}, nil
}

func (a *sunoFailurePollingAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	return nil, nil
}

func (a *sunoFailurePollingAdaptor) AdjustBillingOnComplete(_ *model.Task, _ *relaycommon.TaskInfo) int {
	return 0
}

func (a *taskPollingFetchAdaptor) Init(_ *relaycommon.RelayInfo) {}

func (a *taskPollingFetchAdaptor) FetchTask(_ string, _ string, body map[string]any, _ string) (*http.Response, error) {
	taskID, _ := body["task_id"].(string)
	if taskID == a.blockTaskID && a.releaseBlock != nil {
		a.blockOnce.Do(func() {
			if a.blockStarted != nil {
				close(a.blockStarted)
			}
		})
		<-a.releaseBlock
	}

	a.mu.Lock()
	a.taskIDs = append(a.taskIDs, taskID)
	a.mu.Unlock()
	if a.fetched != nil {
		select {
		case a.fetched <- taskID:
		default:
		}
	}

	response := taskdto.TaskResponse[model.Task]{
		Code: taskdto.TaskSuccessCode,
		Data: model.Task{
			TaskID:   taskID,
			Status:   model.TaskStatusInProgress,
			Progress: "30%",
		},
	}
	responseBody, err := common.Marshal(response)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(responseBody)),
	}, nil
}

func (a *taskPollingFetchAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	return &relaycommon.TaskInfo{Status: model.TaskStatusInProgress}, nil
}

func (a *taskPollingFetchAdaptor) AdjustBillingOnComplete(_ *model.Task, _ *relaycommon.TaskInfo) int {
	return 0
}

func (a *taskPollingFetchAdaptor) fetchCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.taskIDs)
}

func (a *taskPollingFetchAdaptor) fetchedTaskIDs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.taskIDs...)
}

func withTaskPollingQueryLimit(t *testing.T, limit int) {
	t.Helper()
	previous := constant.TaskQueryLimit
	constant.TaskQueryLimit = limit
	t.Cleanup(func() { constant.TaskQueryLimit = previous })
}

func seedTaskPollingChannel(t *testing.T, id int, disableSleep bool) {
	t.Helper()
	ch := &model.Channel{
		Id:     id,
		Type:   constant.ChannelTypeKling,
		Name:   "polling_channel",
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	if disableSleep {
		ch.SetOtherSettings(dto.ChannelOtherSettings{DisableTaskPollingSleep: true})
	}
	require.NoError(t, model.DB.Create(ch).Error)
}

func seedPollingTask(t *testing.T, channelID int, publicID string, upstreamID string) *model.Task {
	t.Helper()
	task := &model.Task{
		TaskID:    publicID,
		Platform:  constant.TaskPlatform("kling"),
		UserId:    1,
		ChannelId: channelID,
		Action:    constant.TaskActionGenerate,
		Status:    model.TaskStatusInProgress,
		Progress:  "30%",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: upstreamID,
		},
	}
	require.NoError(t, model.DB.Create(task).Error)
	return task
}

func TestUpdateVideoTasksDefaultSleepWaitsBetweenTasks(t *testing.T) {
	truncate(t)

	const channelID = 101
	seedTaskPollingChannel(t, channelID, false)
	first := seedPollingTask(t, channelID, "task_public_1", "upstream_1")
	second := seedPollingTask(t, channelID, "task_public_2", "upstream_2")

	adaptor := &taskPollingFetchAdaptor{}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := UpdateVideoTasks(ctx, constant.TaskPlatform("kling"), map[int][]string{
		channelID: {
			first.GetUpstreamTaskID(),
			second.GetUpstreamTaskID(),
		},
	}, map[string]*model.Task{
		first.GetUpstreamTaskID():  first,
		second.GetUpstreamTaskID(): second,
	})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 1, adaptor.fetchCount())
}

func TestRunTaskPollingSkipsSubmittingTasksWithoutUpstreamID(t *testing.T) {
	truncate(t)
	withTaskPollingQueryLimit(t, 1000)

	const channelID = 100
	seedTaskPollingChannel(t, channelID, true)
	task := seedPollingTask(t, channelID, "task_submitting", "")
	task.Status = model.TaskStatusSubmitting
	task.SubmitTime = time.Now().Unix()
	require.NoError(t, task.Update())

	adaptor := &taskPollingFetchAdaptor{}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	RunTaskPollingOnce(context.Background(), nil)

	assert.Zero(t, adaptor.fetchCount())
	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.Equal(t, model.TaskStatusSubmitting, reloaded.Status)
}

func TestRunTaskPollingPollsSubmittingTasksWithUpstreamID(t *testing.T) {
	truncate(t)
	withTaskPollingQueryLimit(t, 1000)

	const channelID = 103
	seedTaskPollingChannel(t, channelID, true)
	task := seedPollingTask(t, channelID, "task_submitting_accepted", "upstream_submitting_accepted")
	task.Status = model.TaskStatusSubmitting
	task.SubmitTime = time.Now().Unix()
	task.PrivateData.NoAutomaticRefund = true
	require.NoError(t, task.Update())

	adaptor := &taskPollingFetchAdaptor{}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	RunTaskPollingOnce(context.Background(), nil)

	assert.Equal(t, 1, adaptor.fetchCount())
	assert.Equal(t, []string{"upstream_submitting_accepted"}, adaptor.fetchedTaskIDs())
	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, model.TaskStatusInProgress, reloaded.Status)
	assert.True(t, reloaded.PrivateData.NoAutomaticRefund)
}

func TestUpdateVideoTasksCanSkipPollingSleepPerChannel(t *testing.T) {
	truncate(t)

	const channelID = 102
	seedTaskPollingChannel(t, channelID, true)
	first := seedPollingTask(t, channelID, "task_public_3", "upstream_3")
	second := seedPollingTask(t, channelID, "task_public_4", "upstream_4")

	adaptor := &taskPollingFetchAdaptor{}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := UpdateVideoTasks(ctx, constant.TaskPlatform("kling"), map[int][]string{
		channelID: {
			first.GetUpstreamTaskID(),
			second.GetUpstreamTaskID(),
		},
	}, map[string]*model.Task{
		first.GetUpstreamTaskID():  first,
		second.GetUpstreamTaskID(): second,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, adaptor.fetchCount())
}

func TestUpdateVideoTasksDefaultSleepDoesNotBlockOtherChannels(t *testing.T) {
	truncate(t)

	const firstChannelID = 201
	const secondChannelID = 202
	seedTaskPollingChannel(t, firstChannelID, false)
	seedTaskPollingChannel(t, secondChannelID, false)
	firstChannelFirst := seedPollingTask(t, firstChannelID, "task_public_5", "upstream_a_1")
	firstChannelSecond := seedPollingTask(t, firstChannelID, "task_public_6", "upstream_a_2")
	secondChannelFirst := seedPollingTask(t, secondChannelID, "task_public_7", "upstream_b_1")
	secondChannelSecond := seedPollingTask(t, secondChannelID, "task_public_8", "upstream_b_2")

	adaptor := &taskPollingFetchAdaptor{}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := UpdateVideoTasks(ctx, constant.TaskPlatform("kling"), map[int][]string{
		firstChannelID: {
			firstChannelFirst.GetUpstreamTaskID(),
			firstChannelSecond.GetUpstreamTaskID(),
		},
		secondChannelID: {
			secondChannelFirst.GetUpstreamTaskID(),
			secondChannelSecond.GetUpstreamTaskID(),
		},
	}, map[string]*model.Task{
		firstChannelFirst.GetUpstreamTaskID():   firstChannelFirst,
		firstChannelSecond.GetUpstreamTaskID():  firstChannelSecond,
		secondChannelFirst.GetUpstreamTaskID():  secondChannelFirst,
		secondChannelSecond.GetUpstreamTaskID(): secondChannelSecond,
	})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.ElementsMatch(t, []string{"upstream_a_1", "upstream_b_1"}, adaptor.fetchedTaskIDs())
}

func TestUpdateVideoTasksSlowChannelDoesNotBlockOtherChannels(t *testing.T) {
	truncate(t)

	const slowChannelID = 251
	const fastChannelID = 252
	seedTaskPollingChannel(t, slowChannelID, false)
	seedTaskPollingChannel(t, fastChannelID, true)
	slowTask := seedPollingTask(t, slowChannelID, "task_public_slow", "upstream_slow_1")
	fastFirst := seedPollingTask(t, fastChannelID, "task_public_fast_1", "upstream_fast_parallel_1")
	fastSecond := seedPollingTask(t, fastChannelID, "task_public_fast_2", "upstream_fast_parallel_2")

	adaptor := &taskPollingFetchAdaptor{
		fetched:      make(chan string, 4),
		blockTaskID:  slowTask.GetUpstreamTaskID(),
		blockStarted: make(chan struct{}),
		releaseBlock: make(chan struct{}),
	}
	var releaseOnce sync.Once
	releaseBlockedTask := func() {
		releaseOnce.Do(func() {
			close(adaptor.releaseBlock)
		})
	}
	t.Cleanup(releaseBlockedTask)
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	errCh := make(chan error, 1)
	gopool.Go(func() {
		errCh <- UpdateVideoTasks(context.Background(), constant.TaskPlatform("kling"), map[int][]string{
			slowChannelID: {
				slowTask.GetUpstreamTaskID(),
			},
			fastChannelID: {
				fastFirst.GetUpstreamTaskID(),
				fastSecond.GetUpstreamTaskID(),
			},
		}, map[string]*model.Task{
			slowTask.GetUpstreamTaskID():   slowTask,
			fastFirst.GetUpstreamTaskID():  fastFirst,
			fastSecond.GetUpstreamTaskID(): fastSecond,
		})
	})

	select {
	case <-adaptor.blockStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("slow channel did not start blocking")
	}

	require.Eventually(t, func() bool {
		fetchedTaskIDs := adaptor.fetchedTaskIDs()
		return len(fetchedTaskIDs) == 2 &&
			fetchedTaskIDs[0] == fastFirst.GetUpstreamTaskID() &&
			fetchedTaskIDs[1] == fastSecond.GetUpstreamTaskID()
	}, 500*time.Millisecond, 10*time.Millisecond)

	releaseBlockedTask()
	require.NoError(t, <-errCh)
	assert.ElementsMatch(t, []string{
		slowTask.GetUpstreamTaskID(),
		fastFirst.GetUpstreamTaskID(),
		fastSecond.GetUpstreamTaskID(),
	}, adaptor.fetchedTaskIDs())
}

func TestUpdateVideoTasksMixedChannelSleepSettings(t *testing.T) {
	truncate(t)

	const sleepyChannelID = 301
	const fastChannelID = 302
	seedTaskPollingChannel(t, sleepyChannelID, false)
	seedTaskPollingChannel(t, fastChannelID, true)
	sleepyFirst := seedPollingTask(t, sleepyChannelID, "task_public_9", "upstream_sleepy_1")
	sleepySecond := seedPollingTask(t, sleepyChannelID, "task_public_10", "upstream_sleepy_2")
	fastFirst := seedPollingTask(t, fastChannelID, "task_public_11", "upstream_fast_1")
	fastSecond := seedPollingTask(t, fastChannelID, "task_public_12", "upstream_fast_2")

	adaptor := &taskPollingFetchAdaptor{}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := UpdateVideoTasks(ctx, constant.TaskPlatform("kling"), map[int][]string{
		sleepyChannelID: {
			sleepyFirst.GetUpstreamTaskID(),
			sleepySecond.GetUpstreamTaskID(),
		},
		fastChannelID: {
			fastFirst.GetUpstreamTaskID(),
			fastSecond.GetUpstreamTaskID(),
		},
	}, map[string]*model.Task{
		sleepyFirst.GetUpstreamTaskID():  sleepyFirst,
		sleepySecond.GetUpstreamTaskID(): sleepySecond,
		fastFirst.GetUpstreamTaskID():    fastFirst,
		fastSecond.GetUpstreamTaskID():   fastSecond,
	})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.ElementsMatch(t, []string{"upstream_sleepy_1", "upstream_fast_1", "upstream_fast_2"}, adaptor.fetchedTaskIDs())
}

// videoFailurePollingAdaptor simulates an upstream whose poll result fails the
// task; noRefund toggles the manual-review path (unknown upstream status).
type videoFailurePollingAdaptor struct {
	noRefund    bool
	reason      string
	directParse bool
}

type videoSuccessPollingAdaptor struct {
	resultURL string
}

type videoHTTPStatusPollingAdaptor struct {
	statusCode int
	body       *trackingPollingBody
	parsed     bool
}

func (a *videoHTTPStatusPollingAdaptor) Init(_ *relaycommon.RelayInfo) {}

func (a *videoHTTPStatusPollingAdaptor) FetchTask(
	_ string,
	_ string,
	_ map[string]any,
	_ string,
) (*http.Response, error) {
	return &http.Response{
		StatusCode: a.statusCode,
		Body:       a.body,
	}, nil
}

func (a *videoHTTPStatusPollingAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	a.parsed = true
	return &relaycommon.TaskInfo{Status: model.TaskStatusFailure}, nil
}

func (a *videoHTTPStatusPollingAdaptor) AdjustBillingOnComplete(
	_ *model.Task,
	_ *relaycommon.TaskInfo,
) int {
	return 0
}

func (a *videoHTTPStatusPollingAdaptor) PreferDirectTaskResultParsing() bool {
	return true
}

func TestUpdateVideoSingleTaskClassifiesHTTPStatusBeforeParsing(t *testing.T) {
	t.Run("retryable response remains pollable", func(t *testing.T) {
		truncate(t)
		task := &model.Task{
			TaskID:    "video_public_retryable_http",
			ChannelId: 701,
			Status:    model.TaskStatusInProgress,
			Quota:     500,
			PrivateData: model.TaskPrivateData{
				UpstreamTaskID: "video_upstream_retryable_http",
				VideoTask:      true,
			},
		}
		require.NoError(t, model.DB.Create(task).Error)
		body := &trackingPollingBody{Reader: bytes.NewReader([]byte(
			`{"status":"failed","message":"https://r2.example/input?X-Amz-Signature=secret"}`,
		))}
		adaptor := &videoHTTPStatusPollingAdaptor{
			statusCode: http.StatusServiceUnavailable,
			body:       body,
		}

		err := updateVideoSingleTask(
			context.Background(),
			adaptor,
			&model.Channel{Id: 701, Key: "provider-key"},
			task.GetUpstreamTaskID(),
			map[string]*model.Task{task.GetUpstreamTaskID(): task},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "retryable status 503")
		assert.True(t, body.closed)
		assert.False(t, adaptor.parsed)

		var stored model.Task
		require.NoError(t, model.DB.First(&stored, task.ID).Error)
		assert.EqualValues(t, model.TaskStatusInProgress, stored.Status)
		assert.Equal(t, 500, stored.Quota)
	})

	t.Run("permanent response requires provider review", func(t *testing.T) {
		truncate(t)
		task := &model.Task{
			TaskID:    "video_public_permanent_http",
			ChannelId: 702,
			Status:    model.TaskStatusInProgress,
			Quota:     500,
			PrivateData: model.TaskPrivateData{
				UpstreamTaskID: "video_upstream_permanent_http",
				VideoTask:      true,
			},
		}
		require.NoError(t, model.DB.Create(task).Error)
		body := &trackingPollingBody{Reader: bytes.NewReader([]byte(
			`{"status":"failed","message":"https://r2.example/input?X-Amz-Signature=secret"}`,
		))}
		adaptor := &videoHTTPStatusPollingAdaptor{
			statusCode: http.StatusUnauthorized,
			body:       body,
		}

		require.NoError(t, updateVideoSingleTask(
			context.Background(),
			adaptor,
			&model.Channel{Id: 702, Key: "provider-key"},
			task.GetUpstreamTaskID(),
			map[string]*model.Task{task.GetUpstreamTaskID(): task},
		))
		assert.True(t, body.closed)
		assert.False(t, adaptor.parsed)

		var stored model.Task
		require.NoError(t, model.DB.First(&stored, task.ID).Error)
		assert.EqualValues(t, model.TaskStatusFailure, stored.Status)
		assert.True(t, stored.PrivateData.NoAutomaticRefund)
		assert.Equal(t, model.TaskStorageStatusProviderReview, stored.PrivateData.StorageStatus)
		assert.Equal(t, 500, stored.Quota)
		assert.NotContains(t, stored.FailReason, "X-Amz-Signature")
	})
}

func TestSanitizeTaskFailureReasonRedactsProviderURLsAndBoundsLength(t *testing.T) {
	reason := "provider could not fetch https://r2.example/input.png?X-Amz-Signature=secret"
	sanitized := sanitizeTaskFailureReason(reason)
	assert.Equal(t, "provider could not fetch [provider URL redacted]", sanitized)
	assert.NotContains(t, sanitized, "X-Amz-Signature")

	assert.Equal(t, "Provider task failed", sanitizeTaskFailureReason("Brioi task failed"))
	assert.Equal(t, "Provider task failed", sanitizeTaskFailureReason("SilkRoad task failed"))
	assert.Equal(
		t,
		"Provider returned an unknown task status; administrator review is required",
		sanitizeTaskFailureReason("Brioi returned an unknown task status; administrator review is required"),
	)
	assert.NotContains(t, sanitizeTaskFailureReason("Silk Road poll failed"), "Silk")
	assert.NotContains(t, sanitizeTaskFailureReason("silk_road task failed"), "silk")

	longReason := strings.Repeat("故障", 600)
	assert.LessOrEqual(t, len([]rune(sanitizeTaskFailureReason(longReason))), 515)
}

func (a *videoSuccessPollingAdaptor) Init(_ *relaycommon.RelayInfo) {}

func (a *videoSuccessPollingAdaptor) FetchTask(
	_ string,
	_ string,
	_ map[string]any,
	_ string,
) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(
			`{"id":"upstream-secret","status":"completed","video_url":"` +
				a.resultURL +
				`"}`,
		)),
	}, nil
}

func (a *videoSuccessPollingAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	return &relaycommon.TaskInfo{
		Status: model.TaskStatusSuccess,
		Url:    a.resultURL,
	}, nil
}

func (a *videoSuccessPollingAdaptor) AdjustBillingOnComplete(
	_ *model.Task,
	_ *relaycommon.TaskInfo,
) int {
	return 0
}

func (a *videoSuccessPollingAdaptor) PreferDirectTaskResultParsing() bool {
	return true
}

func TestUpdateVideoTasksMovesProviderSuccessIntoStoragePhase(t *testing.T) {
	truncate(t)
	withSilkRoadStorage(t, t.TempDir(), "node-a", "https://video.example.com")

	const channelID = 504
	channel := &model.Channel{
		Id:     channelID,
		Type:   constant.ChannelTypeBrioi,
		Name:   "brioi_polling_channel",
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{DisableTaskPollingSleep: true})
	require.NoError(t, model.DB.Create(channel).Error)
	task := &model.Task{
		TaskID:     "video_public_storing",
		Platform:   constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeBrioi)),
		ChannelId:  channelID,
		Status:     model.TaskStatusInProgress,
		Progress:   "50%",
		SubmitTime: time.Now().Unix(),
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "video_upstream_storing",
			VideoTask:      true,
		},
	}
	require.NoError(t, model.DB.Create(task).Error)

	resultURL := "https://cdn.example/private-result.mp4"
	adaptor := &videoSuccessPollingAdaptor{resultURL: resultURL}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	require.NoError(t, UpdateVideoTasks(
		context.Background(),
		task.Platform,
		map[int][]string{channelID: {task.GetUpstreamTaskID()}},
		map[string]*model.Task{task.GetUpstreamTaskID(): task},
	))

	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.Equal(t, model.TaskStatusStoring, reloaded.Status)
	assert.Equal(t, "99%", reloaded.Progress)
	assert.Equal(t, "pending", reloaded.PrivateData.StorageStatus)
	assert.Equal(t, resultURL, reloaded.PrivateData.UpstreamResultURL)
	assert.Equal(t, "https://video.example.com/v1/videos/"+task.TaskID+"/content", reloaded.PrivateData.ResultURL)
	assert.Zero(t, reloaded.FinishTime)
	assert.NotContains(t, string(reloaded.Data), resultURL)
	assert.NotContains(t, string(reloaded.Data), "upstream-secret")
	assert.Contains(t, string(reloaded.Data), task.TaskID)
}

func TestRunTaskPollingRecoversAcceptedSubmittingTaskIntoStorage(t *testing.T) {
	truncate(t)
	withTaskPollingQueryLimit(t, 1000)
	withSilkRoadStorage(t, t.TempDir(), "node-a", "https://video.example.com")

	const channelID = 505
	channel := &model.Channel{
		Id:     channelID,
		Type:   constant.ChannelTypeBrioi,
		Name:   "brioi_accepted_submitting",
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{DisableTaskPollingSleep: true})
	require.NoError(t, model.DB.Create(channel).Error)
	task := &model.Task{
		TaskID:     "video_public_accepted_submitting",
		Platform:   constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeBrioi)),
		ChannelId:  channelID,
		Status:     model.TaskStatusSubmitting,
		Progress:   "0%",
		SubmitTime: time.Now().Unix(),
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID:    "video_upstream_accepted_submitting",
			VideoTask:         true,
			NoAutomaticRefund: true,
		},
	}
	require.NoError(t, model.DB.Create(task).Error)

	resultURL := "https://cdn.example/private-result.mp4"
	adaptor := &videoSuccessPollingAdaptor{resultURL: resultURL}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	RunTaskPollingOnce(context.Background(), nil)

	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, model.TaskStatusStoring, reloaded.Status)
	assert.Equal(t, "99%", reloaded.Progress)
	assert.Equal(t, "pending", reloaded.PrivateData.StorageStatus)
	assert.Equal(t, resultURL, reloaded.PrivateData.UpstreamResultURL)
	assert.Equal(t, "https://video.example.com/v1/videos/"+task.TaskID+"/content", reloaded.PrivateData.ResultURL)
	assert.Zero(t, reloaded.FinishTime)
}

func TestUpdateVideoTasksRequiresResultBeforeSettlement(t *testing.T) {
	truncate(t)

	const channelID = 506
	channel := &model.Channel{
		Id:     channelID,
		Type:   constant.ChannelTypeSilkRoad,
		Name:   "silkroad_missing_result",
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{DisableTaskPollingSleep: true})
	require.NoError(t, model.DB.Create(channel).Error)
	task := &model.Task{
		TaskID:     "video_missing_result",
		Platform:   constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSilkRoad)),
		ChannelId:  channelID,
		Status:     model.TaskStatusInProgress,
		Progress:   "50%",
		Quota:      321,
		SubmitTime: time.Now().Unix(),
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "video_upstream_missing_result",
			VideoTask:      true,
		},
	}
	require.NoError(t, model.DB.Create(task).Error)

	adaptor := &videoSuccessPollingAdaptor{}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })
	require.NoError(t, UpdateVideoTasks(
		context.Background(),
		task.Platform,
		map[int][]string{channelID: {task.GetUpstreamTaskID()}},
		map[string]*model.Task{task.GetUpstreamTaskID(): task},
	))

	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), reloaded.Status)
	assert.Equal(t, model.TaskStorageStatusProviderReview, reloaded.PrivateData.StorageStatus)
	assert.True(t, reloaded.PrivateData.NoAutomaticRefund)
	assert.Equal(t, 321, reloaded.Quota)
	assert.Empty(t, reloaded.PrivateData.UpstreamResultURL)
	assert.Contains(t, reloaded.FailReason, "without a usable result")
}

func TestUpdateVideoTasksMarksRemovedChannelForReview(t *testing.T) {
	truncate(t)
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = previousMemoryCacheEnabled })

	task := seedPollingTask(t, 999_999, "video_missing_channel", "upstream_missing_channel")
	task.Quota = 654
	task.PrivateData.VideoTask = true
	require.NoError(t, task.Update())

	require.NoError(t, UpdateVideoTasks(
		context.Background(),
		task.Platform,
		map[int][]string{task.ChannelId: {task.GetUpstreamTaskID()}},
		map[string]*model.Task{task.GetUpstreamTaskID(): task},
	))

	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), reloaded.Status)
	assert.Equal(t, model.TaskStorageStatusProviderReview, reloaded.PrivateData.StorageStatus)
	assert.True(t, reloaded.PrivateData.NoAutomaticRefund)
	assert.Equal(t, 654, reloaded.Quota)
	assert.Contains(t, reloaded.FailReason, "no longer exists")
}

func TestUpdateVideoTasksLeavesTaskPendingOnChannelDatabaseFailure(t *testing.T) {
	truncate(t)
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { common.MemoryCacheEnabled = previousMemoryCacheEnabled })

	task := seedPollingTask(t, 808, "video_transient_channel_error", "upstream_transient_channel_error")
	healthyDB := model.DB
	brokenDB, err := gorm.Open(
		sqlite.Open("file:broken-video-channel-lookup?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	require.NoError(t, err)
	sqlDB, err := brokenDB.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	model.DB = brokenDB
	t.Cleanup(func() { model.DB = healthyDB })

	err = updateVideoTasks(
		context.Background(),
		task.Platform,
		task.ChannelId,
		[]string{task.GetUpstreamTaskID()},
		map[string]*model.Task{task.GetUpstreamTaskID(): task},
	)
	require.Error(t, err)
	assert.Equal(t, model.TaskStatus(model.TaskStatusInProgress), task.Status)

	model.DB = healthyDB
	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusInProgress), reloaded.Status)
	assert.Empty(t, reloaded.PrivateData.StorageStatus)
}

func TestUpdateVideoTasksFallsBackToDatabaseAfterCacheMiss(t *testing.T) {
	truncate(t)
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() { common.MemoryCacheEnabled = previousMemoryCacheEnabled })

	const channelID = 909_090
	seedTaskPollingChannel(t, channelID, true)
	task := seedPollingTask(t, channelID, "video_cache_miss", "upstream_cache_miss")
	adaptor := &taskPollingFetchAdaptor{}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	require.NoError(t, updateVideoTasks(
		context.Background(),
		task.Platform,
		channelID,
		[]string{task.GetUpstreamTaskID()},
		map[string]*model.Task{task.GetUpstreamTaskID(): task},
	))
	assert.Equal(t, 1, adaptor.fetchCount())
}

type blockingSettlementAdaptor struct {
	videoSuccessPollingAdaptor
	entered chan struct{}
	release chan struct{}
}

func (a *blockingSettlementAdaptor) AdjustBillingOnComplete(
	_ *model.Task,
	_ *relaycommon.TaskInfo,
) int {
	close(a.entered)
	<-a.release
	return 0
}

func TestUpdateVideoTasksDoesNotExposeStorageBeforeBillingSettlement(t *testing.T) {
	truncate(t)
	withSilkRoadStorage(t, t.TempDir(), "node-a", "https://video.example.com")

	const channelID = 505
	seedTaskPollingChannel(t, channelID, true)
	task := &model.Task{
		TaskID:     "video_settlement_guard",
		Platform:   constant.TaskPlatform("kling"),
		ChannelId:  channelID,
		Status:     model.TaskStatusInProgress,
		Progress:   "50%",
		SubmitTime: time.Now().Unix(),
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "video_upstream_settlement_guard",
		},
	}
	require.NoError(t, model.DB.Create(task).Error)

	adaptor := &blockingSettlementAdaptor{
		videoSuccessPollingAdaptor: videoSuccessPollingAdaptor{
			resultURL: "https://cdn.example/private-result.mp4",
		},
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	errCh := make(chan error, 1)
	go func() {
		errCh <- UpdateVideoTasks(
			context.Background(),
			task.Platform,
			map[int][]string{channelID: {task.GetUpstreamTaskID()}},
			map[string]*model.Task{task.GetUpstreamTaskID(): task},
		)
	}()
	<-adaptor.entered

	var settling model.Task
	require.NoError(t, model.DB.First(&settling, task.ID).Error)
	assert.EqualValues(t, model.TaskStatusInProgress, settling.Status)
	assert.Empty(t, settling.PrivateData.StorageStatus)
	claimed, err := claimSilkRoadIngestTasks(1, 5)
	require.NoError(t, err)
	assert.Empty(t, claimed)

	close(adaptor.release)
	require.NoError(t, <-errCh)
	require.NoError(t, model.DB.First(&settling, task.ID).Error)
	assert.Equal(t, model.TaskStatusStoring, settling.Status)
	assert.Equal(t, "pending", settling.PrivateData.StorageStatus)
}

func TestRecoverStaleVideoSettlementResumesStorageExactlyOnce(t *testing.T) {
	truncate(t)

	const userID, initialQuota = 406, 10_000
	seedUser(t, userID, initialQuota)
	task := &model.Task{
		TaskID:     "video_stale_settlement",
		Platform:   constant.TaskPlatform("kling"),
		UserId:     userID,
		Status:     model.TaskStatusSettlementProcessing,
		Progress:   "99%",
		Quota:      500,
		SubmitTime: time.Now().Add(-time.Hour).Unix(),
		PrivateData: model.TaskPrivateData{
			VideoTask:             true,
			UpstreamTaskID:        "video_upstream_stale_settlement",
			UpstreamResultURL:     "https://cdn.example/private-result.mp4",
			StorageStatus:         "settling",
			SettlementTargetQuota: 345,
		},
	}
	require.NoError(t, model.DB.Create(task).Error)
	staleAt := time.Now().Add(-videoStorageClaimTimeout - time.Minute).Unix()
	require.NoError(t, model.DB.Model(task).UpdateColumn("updated_at", staleAt).Error)

	require.NoError(t, recoverStaleVideoSettlements(context.Background(), 10))
	require.NoError(t, recoverStaleVideoSettlements(context.Background(), 10))

	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.Equal(t, model.TaskStatusStoring, reloaded.Status)
	assert.Equal(t, "pending", reloaded.PrivateData.StorageStatus)
	assert.True(t, reloaded.PrivateData.BillingSettlementApplied)
	assert.Equal(t, 345, reloaded.Quota)
	assert.Equal(t, initialQuota+155, getUserQuota(t, userID))
}

func (a *videoFailurePollingAdaptor) Init(_ *relaycommon.RelayInfo) {}

func (a *videoFailurePollingAdaptor) FetchTask(_ string, _ string, _ map[string]any, _ string) (*http.Response, error) {
	body := []byte(`{"status":"expired"}`)
	if a.directParse {
		body = []byte(`{"code":"success","data":{"status":"mystery"}}`)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}, nil
}

func (a *videoFailurePollingAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	return &relaycommon.TaskInfo{
		Status:   model.TaskStatusFailure,
		Reason:   a.reason,
		NoRefund: a.noRefund,
	}, nil
}

func (a *videoFailurePollingAdaptor) AdjustBillingOnComplete(_ *model.Task, _ *relaycommon.TaskInfo) int {
	return 0
}

func (a *videoFailurePollingAdaptor) PreferDirectTaskResultParsing() bool {
	return a.directParse
}

// 上游返回未知终态（NoRefund=true）时：任务标记失败，但预扣额度必须保留、
// 不产生退款日志，等待人工介入 —— 防止"用户已退款、上游已扣费"的资金损失。
func TestUpdateVideoTasksNoRefundFailureRetainsQuota(t *testing.T) {
	truncate(t)

	const userID, tokenID, channelID = 501, 501, 501
	const initialUserQuota, initialTokenQuota, taskQuota = 10_000, 6_000, 2_500

	seedUser(t, userID, initialUserQuota)
	seedToken(t, tokenID, userID, "sk-video-no-refund", initialTokenQuota)
	seedTaskPollingChannel(t, channelID, true)

	task := makeTask(userID, channelID, taskQuota, tokenID, BillingSourceWallet, 0)
	task.TaskID = "video_public_no_refund"
	task.Platform = constant.TaskPlatform("kling")
	task.PrivateData.UpstreamTaskID = "video_upstream_no_refund"
	require.NoError(t, model.DB.Create(task).Error)

	adaptor := &videoFailurePollingAdaptor{noRefund: true, reason: "上游返回未知状态，需人工介入"}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	require.NoError(t, UpdateVideoTasks(context.Background(), constant.TaskPlatform("kling"), map[int][]string{
		channelID: {task.GetUpstreamTaskID()},
	}, map[string]*model.Task{
		task.GetUpstreamTaskID(): task,
	}))

	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, model.TaskStatusFailure, reloaded.Status)
	assert.Contains(t, reloaded.FailReason, "人工介入")
	assert.True(t, reloaded.PrivateData.NoAutomaticRefund)
	assert.Equal(t, model.TaskStorageStatusProviderReview, reloaded.PrivateData.StorageStatus)
	assert.Contains(t, reloaded.PrivateData.StorageLastError, "人工介入")
	// 额度保留在任务上，未退款、无退款日志
	assert.Equal(t, taskQuota, reloaded.Quota)
	assert.Equal(t, initialUserQuota, getUserQuota(t, userID))
	assert.Equal(t, initialTokenQuota, getTokenRemainQuota(t, tokenID))
	assert.Equal(t, int64(0), countLogs(t))
}

func TestUpdateVideoTasksDirectParserHandlesUnknownNestedStatus(t *testing.T) {
	truncate(t)

	const userID, tokenID, channelID = 503, 503, 503
	const initialUserQuota, initialTokenQuota, taskQuota = 10_000, 6_000, 2_500

	seedUser(t, userID, initialUserQuota)
	seedToken(t, tokenID, userID, "sk-video-nested-unknown", initialTokenQuota)
	seedTaskPollingChannel(t, channelID, true)

	task := makeTask(userID, channelID, taskQuota, tokenID, BillingSourceWallet, 0)
	task.TaskID = "video_public_nested_unknown"
	task.Platform = constant.TaskPlatform("kling")
	task.PrivateData.UpstreamTaskID = "video_upstream_nested_unknown"
	require.NoError(t, model.DB.Create(task).Error)

	adaptor := &videoFailurePollingAdaptor{
		noRefund:    true,
		reason:      "上游返回未知状态，需人工介入",
		directParse: true,
	}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	require.NoError(t, UpdateVideoTasks(context.Background(), constant.TaskPlatform("kling"), map[int][]string{
		channelID: {task.GetUpstreamTaskID()},
	}, map[string]*model.Task{
		task.GetUpstreamTaskID(): task,
	}))

	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, model.TaskStatusFailure, reloaded.Status)
	assert.Equal(t, taskQuota, reloaded.Quota)
	assert.Equal(t, initialUserQuota, getUserQuota(t, userID))
}

// 对照：普通失败（NoRefund=false）仍然正常退款，确保新增开关没有反向影响。
func TestUpdateVideoTasksNormalFailureStillRefunds(t *testing.T) {
	truncate(t)

	const userID, tokenID, channelID = 502, 502, 502
	const initialUserQuota, initialTokenQuota, taskQuota = 10_000, 6_000, 2_500

	seedUser(t, userID, initialUserQuota)
	seedToken(t, tokenID, userID, "sk-video-refund", initialTokenQuota)
	seedTaskPollingChannel(t, channelID, true)

	task := makeTask(userID, channelID, taskQuota, tokenID, BillingSourceWallet, 0)
	task.TaskID = "video_public_refund"
	task.Platform = constant.TaskPlatform("kling")
	task.PrivateData.UpstreamTaskID = "video_upstream_refund"
	require.NoError(t, model.DB.Create(task).Error)

	adaptor := &videoFailurePollingAdaptor{noRefund: false, reason: "task failed"}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	require.NoError(t, UpdateVideoTasks(context.Background(), constant.TaskPlatform("kling"), map[int][]string{
		channelID: {task.GetUpstreamTaskID()},
	}, map[string]*model.Task{
		task.GetUpstreamTaskID(): task,
	}))

	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, model.TaskStatusFailure, reloaded.Status)
	assert.Zero(t, reloaded.Quota)
	assert.Equal(t, initialUserQuota+taskQuota, getUserQuota(t, userID))
	assert.Equal(t, int64(1), countLogs(t))
}

func TestUpdateSunoTasksStalePollsRefundExactlyOnce(t *testing.T) {
	truncate(t)

	const userID, tokenID, channelID = 401, 401, 401
	const initialUserQuota, initialTokenQuota, taskQuota = 10_000, 6_000, 2_500
	const publicTaskID, upstreamTaskID = "suno_public_refund_once", "suno_upstream_refund_once"

	seedUser(t, userID, initialUserQuota)
	seedToken(t, tokenID, userID, "sk-suno-refund-once", initialTokenQuota)
	baseURL := "https://suno.invalid"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:      channelID,
		Type:    constant.ChannelTypeSunoAPI,
		Name:    "suno_refund_once",
		Key:     "sk-suno-channel",
		Status:  common.ChannelStatusEnabled,
		BaseURL: &baseURL,
	}).Error)

	task := makeTask(userID, channelID, taskQuota, tokenID, BillingSourceWallet, 0)
	task.TaskID = publicTaskID
	task.Platform = constant.TaskPlatformSuno
	task.Status = model.TaskStatusInProgress
	task.Progress = "50%"
	task.SubmitTime = time.Now().Unix()
	task.PrivateData.UpstreamTaskID = upstreamTaskID
	require.NoError(t, model.DB.Create(task).Error)

	var firstPollTask model.Task
	var staleSecondPollTask model.Task
	require.NoError(t, model.DB.First(&firstPollTask, task.ID).Error)
	require.NoError(t, model.DB.First(&staleSecondPollTask, task.ID).Error)

	adaptor := &sunoFailurePollingAdaptor{failReason: "upstream failed"}
	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return adaptor }
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	require.NoError(t, updateSunoTasks(context.Background(), channelID, []string{upstreamTaskID}, map[string]*model.Task{
		upstreamTaskID: &firstPollTask,
	}))
	require.NoError(t, updateSunoTasks(context.Background(), channelID, []string{upstreamTaskID}, map[string]*model.Task{
		upstreamTaskID: &staleSecondPollTask,
	}))

	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, model.TaskStatusFailure, reloaded.Status)
	assert.Zero(t, reloaded.Quota)
	assert.Equal(t, initialUserQuota+taskQuota, getUserQuota(t, userID))
	assert.Equal(t, initialTokenQuota+taskQuota, getTokenRemainQuota(t, tokenID))
	assert.Equal(t, int64(1), countLogs(t))
}

func TestRunTaskPollingOnceDoesNotRefundHistoricalFailedTask(t *testing.T) {
	truncate(t)

	const userID, initialQuota, taskQuota = 402, 10_000, 1_200
	seedUser(t, userID, initialQuota)

	task := makeTask(userID, 0, taskQuota, 0, BillingSourceWallet, 0)
	task.TaskID = "historical_failed_already_refunded"
	task.Status = model.TaskStatusFailure
	task.Progress = "100%"
	task.SubmitTime = time.Now().Add(-90 * 24 * time.Hour).Unix()
	task.UpdatedAt = time.Now().Add(-time.Minute).Unix()
	require.NoError(t, model.DB.Create(task).Error)

	previousFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor {
		return &taskPollingFetchAdaptor{}
	}
	t.Cleanup(func() { GetTaskAdaptorFunc = previousFactory })

	summary := RunTaskPollingOnce(context.Background(), nil)

	assert.Zero(t, summary.UnfinishedTasks)
	assert.Equal(t, initialQuota, getUserQuota(t, userID))
	assert.Equal(t, taskQuota, getTaskQuota(t, task.ID))
	assert.Equal(t, int64(0), countLogs(t))
}

func TestSweepTimedOutTasksHonorsRefundRolloutBoundary(t *testing.T) {
	truncate(t)

	const (
		userID          = 403
		initialQuota    = 10_000
		legacyTaskQuota = 1_800
		modernTaskQuota = 1_200
	)
	seedUser(t, userID, initialQuota)

	legacyTask := makeTask(userID, 0, legacyTaskQuota, 0, BillingSourceWallet, 0)
	legacyTask.TaskID = "legacy_timeout_without_refund"
	legacyTask.Progress = "50%"
	legacyTask.SubmitTime = 1771718399 // 2026-02-21 23:59:59 UTC
	require.NoError(t, model.DB.Create(legacyTask).Error)

	modernTask := makeTask(userID, 0, modernTaskQuota, 0, BillingSourceWallet, 0)
	modernTask.TaskID = "modern_timeout_with_refund"
	modernTask.Progress = "50%"
	modernTask.SubmitTime = 1771718400 // 2026-02-22 00:00:00 UTC
	require.NoError(t, model.DB.Create(modernTask).Error)

	previousTimeout := constant.TaskTimeoutMinutes
	constant.TaskTimeoutMinutes = 1
	t.Cleanup(func() { constant.TaskTimeoutMinutes = previousTimeout })

	sweepTimedOutTasks(context.Background())

	var reloadedLegacy model.Task
	var reloadedModern model.Task
	require.NoError(t, model.DB.First(&reloadedLegacy, legacyTask.ID).Error)
	require.NoError(t, model.DB.First(&reloadedModern, modernTask.ID).Error)
	assert.EqualValues(t, model.TaskStatusFailure, reloadedLegacy.Status)
	assert.EqualValues(t, model.TaskStatusFailure, reloadedModern.Status)
	assert.Zero(t, reloadedLegacy.Quota)
	assert.Zero(t, reloadedModern.Quota)
	assert.Contains(t, reloadedLegacy.FailReason, "旧系统遗留任务")
	assert.Contains(t, reloadedModern.FailReason, "任务超时")
	assert.Equal(t, initialQuota+modernTaskQuota, getUserQuota(t, userID))
	assert.Equal(t, int64(1), countLogs(t))
}

func TestSweepTimedOutAcceptedSubmissionRequiresReviewWithoutRefund(t *testing.T) {
	truncate(t)

	const userID, initialQuota, taskQuota = 404, 10_000, 1_300
	seedUser(t, userID, initialQuota)
	task := makeTask(userID, 0, taskQuota, 0, BillingSourceWallet, 0)
	task.TaskID = "accepted_submission_timeout"
	task.Status = model.TaskStatusInProgress
	task.Progress = "50%"
	task.SubmitTime = time.Now().Add(-2 * time.Minute).Unix()
	task.PrivateData.UpstreamTaskID = "upstream_accepted"
	task.PrivateData.NoAutomaticRefund = false
	require.NoError(t, model.DB.Create(task).Error)

	previousTimeout := constant.TaskTimeoutMinutes
	constant.TaskTimeoutMinutes = 1
	t.Cleanup(func() { constant.TaskTimeoutMinutes = previousTimeout })

	sweepTimedOutTasks(context.Background())

	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, model.TaskStatusFailure, reloaded.Status)
	assert.Equal(t, model.TaskStorageStatusProviderReview, reloaded.PrivateData.StorageStatus)
	assert.True(t, reloaded.PrivateData.NoAutomaticRefund)
	assert.Equal(t, taskQuota, reloaded.Quota)
	assert.Equal(t, initialQuota, getUserQuota(t, userID))
	assert.Equal(t, int64(0), countLogs(t))
}

func TestSweepTimedOutSubmittingTaskNeverAssumesProviderRejectedIt(t *testing.T) {
	truncate(t)

	const userID, initialQuota, taskQuota = 405, 10_000, 900
	seedUser(t, userID, initialQuota)
	task := makeTask(userID, 0, taskQuota, 0, BillingSourceWallet, 0)
	task.TaskID = "submission_timeout_without_persisted_acceptance"
	task.Status = model.TaskStatusSubmitting
	task.Progress = "0%"
	task.SubmitTime = time.Now().Add(-2 * time.Minute).Unix()
	require.NoError(t, model.DB.Create(task).Error)

	previousTimeout := constant.TaskTimeoutMinutes
	constant.TaskTimeoutMinutes = 1
	t.Cleanup(func() { constant.TaskTimeoutMinutes = previousTimeout })

	sweepTimedOutTasks(context.Background())

	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, model.TaskStatusFailure, reloaded.Status)
	assert.Equal(t, model.TaskStorageStatusProviderReview, reloaded.PrivateData.StorageStatus)
	assert.True(t, reloaded.PrivateData.NoAutomaticRefund)
	assert.Equal(t, taskQuota, reloaded.Quota)
	assert.Equal(t, initialQuota, getUserQuota(t, userID))
	assert.Equal(t, int64(0), countLogs(t))
}
