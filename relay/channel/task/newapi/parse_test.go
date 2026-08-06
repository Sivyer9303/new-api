package newapi

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTaskResultCompletedWithVideoURL(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{"id":"cgt-1","status":"completed","progress":100,"video_url":"https://cdn.example/a.mp4"}`)
	info, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, info.Status)
	assert.Equal(t, "https://cdn.example/a.mp4", info.Url)
}

func TestParseTaskResultSUCCESS(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{"status":"SUCCESS","video_url":"https://cdn.example/b.mp4"}`)
	info, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, info.Status)
}

// 未知终态必须保守处理：标记失败但不自动退款，等待人工介入，
// 防止"用户已退款、上游已扣费"的资金损失。
func TestParseTaskResultUnknownStatusFailsWithoutRefund(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{"status":"weird"}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusFailure, info.Status)
	assert.True(t, info.NoRefund)
	assert.Contains(t, info.Reason, "weird")
	assert.Contains(t, info.Reason, "人工")
}

// 空 status（如限流错误响应体）保持进行中，交给下一轮轮询处理，不判定终态。
func TestParseTaskResultEmptyStatusKeepsInProgress(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{"id":"cgt-1"}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusInProgress, info.Status)
	assert.False(t, info.NoRefund)
}

func TestParseTaskResultSucceededSynonym(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{"status":"succeeded","video_url":"https://cdn.example/c.mp4"}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, info.Status)
}

// canceled（美式拼写）是明确的失败终态，正常退款，不进入人工介入路径。
func TestParseTaskResultCanceledSynonymRefundable(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{"status":"canceled"}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusFailure, info.Status)
	assert.False(t, info.NoRefund)
}

func TestParseTaskResultFailed(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{"status":"failed"}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusFailure, info.Status)
}

func TestParseTaskResultWrappedTaskDtoWithNestedVideoURL(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{"code":"success","data":{"status":"SUCCESS","result_url":"http://localhost:3000/v1/videos/task_x/content","data":{"video_url":"https://cdn.example/nested.mp4","url":"https://cdn.example/nested.mp4"}}}`)
	info, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, info.Status)
	assert.Equal(t, "https://cdn.example/nested.mp4", info.Url)
}
