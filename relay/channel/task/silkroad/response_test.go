package silkroad

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
	submitResponse, taskErr := a.DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	require.NotNil(t, submitResponse)
	assert.Equal(t, "cgt-upstream-1", submitResponse.UpstreamTaskID)
	assert.Contains(t, string(submitResponse.ResponseData), "task_pub_123")
	assert.NotContains(t, string(submitResponse.ResponseData), "cgt-upstream-1")
	assert.NotContains(t, string(submitResponse.ResponseData), "cdn.example")

	assert.Empty(t, recorder.Body.String())
}

func TestDoResponseUsesFlatOpenAIVideoEnvelopeForVideosRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)

	info := &relaycommon.RelayInfo{
		OriginModelName: "public-seedance",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_pub_openai"},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(
			`{"id":"upstream-secret","status":"queued","url":"https://cdn.example/private.mp4"}`,
		)),
	}

	submitResponse, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	require.NotNil(t, submitResponse)
	assert.Equal(t, "upstream-secret", submitResponse.UpstreamTaskID)
	assert.Contains(t, string(submitResponse.ResponseData), `"id":"task_pub_openai"`)
	assert.Contains(t, string(submitResponse.ResponseData), `"object":"video"`)
	assert.Contains(t, string(submitResponse.ResponseData), `"model":"public-seedance"`)
	assert.Contains(t, string(submitResponse.ResponseData), `"status":"queued"`)
	assert.Contains(t, string(submitResponse.ResponseData), `"progress":0`)
	assert.Contains(t, string(submitResponse.ResponseData), `"created_at":`)
	assert.JSONEq(t, string(submitResponse.ResponseData), string(submitResponse.TaskData))
	assert.NotContains(t, string(submitResponse.ResponseData), "upstream-secret")
	assert.NotContains(t, string(submitResponse.ResponseData), "cdn.example")
	assert.Empty(t, recorder.Body.String())
}
