package compatvideo

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

func TestDoResponseUsesOpenAIVideoEnvelopeForVideosRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "public-video-model",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_pub_123"},
	}
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(
			`{"data":{"id":"upstream-private-id","status":"queued"}}`,
		)),
	}

	submitResponse, taskErr := (&TaskAdaptor{}).DoResponse(context, response, info)
	require.Nil(t, taskErr)
	require.NotNil(t, submitResponse)
	assert.Equal(t, "upstream-private-id", submitResponse.UpstreamTaskID)
	assert.Contains(t, string(submitResponse.ResponseData), `"id":"task_pub_123"`)
	assert.Contains(t, string(submitResponse.ResponseData), `"object":"video"`)
	assert.Contains(t, string(submitResponse.ResponseData), `"model":"public-video-model"`)
	assert.Contains(t, string(submitResponse.ResponseData), `"status":"queued"`)
	assert.Contains(t, string(submitResponse.ResponseData), `"progress":0`)
	assert.Contains(t, string(submitResponse.ResponseData), `"created_at":`)
	assert.NotContains(t, string(submitResponse.ResponseData), "upstream-private-id")
	assert.JSONEq(t, string(submitResponse.ResponseData), string(submitResponse.TaskData))
}
