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

func TestParseTaskResultUnknownKeepsInProgress(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{"status":"weird"}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusInProgress, info.Status)
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
