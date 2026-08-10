package newapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 提交响应必须隐藏上游 URL 并用公开 task ID 替换上游 ID：
// 上游可能在提交时就返回 video_url，客户端拿到的 JSON 中不能出现任何上游地址。
func TestDoResponseHidesUpstreamURLsAndRewritesTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)

	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_pub_123"},
	}
	upstreamBody := `{"id":"cgt-upstream-1","status":"queued","video_url":"https://cdn.example/leak.mp4","url":"https://cdn.example/leak2.mp4","result_url":"https://cdn.example/leak3.mp4"}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}

	a := &TaskAdaptor{}
	upstreamID, taskData, taskErr := a.DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "cgt-upstream-1", upstreamID)
	// taskData 是原始上游响应，供内部存储；客户端响应必须已脱敏
	assert.Contains(t, string(taskData), "cgt-upstream-1")

	clientBody := recorder.Body.String()
	assert.Contains(t, clientBody, "task_pub_123")
	assert.NotContains(t, clientBody, "cgt-upstream-1")
	assert.NotContains(t, clientBody, "cdn.example")
}
