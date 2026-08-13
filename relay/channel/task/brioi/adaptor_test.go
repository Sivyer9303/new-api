package brioi

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdaptorBuildsNormalizedURLsAndBearerHeaders(t *testing.T) {
	adaptor := &TaskAdaptor{}
	adaptor.Init(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: " https://brioi.example/root/ ",
			ApiKey:         "brioi-key",
		},
	})

	requestURL, err := adaptor.BuildRequestURL(nil)
	require.NoError(t, err)
	assert.Equal(t, "https://brioi.example/root/v1/videos", requestURL)

	request := httptest.NewRequest(http.MethodPost, requestURL, nil)
	require.NoError(t, adaptor.BuildRequestHeader(nil, request, nil))
	assert.Equal(t, "Bearer brioi-key", request.Header.Get("Authorization"))
	assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
}

func TestDoResponseRewritesIdentifiersAndDropsProviderData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public_123"},
	}
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"task_id":"provider-secret-id",
			"status":"queued",
			"result_url":"https://provider.example/private.mp4"
		}`)),
	}

	submitResponse, taskErr := (&TaskAdaptor{}).DoResponse(context, response, info)

	require.Nil(t, taskErr)
	require.NotNil(t, submitResponse)
	assert.Equal(t, "provider-secret-id", submitResponse.UpstreamTaskID)
	assert.JSONEq(t, `{
		"id":"task_public_123",
		"task_id":"task_public_123",
		"status":"queued"
	}`, string(submitResponse.TaskData))
	assert.JSONEq(t, string(submitResponse.TaskData), string(submitResponse.ResponseData))
	assert.Empty(t, recorder.Body.String())
	assert.NotContains(t, string(submitResponse.ResponseData), "provider-secret-id")
	assert.NotContains(t, string(submitResponse.ResponseData), "provider.example")
}

func TestParseTaskResultNormalizesStatusesAndResults(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		status     model.TaskStatus
		resultURL  string
		noRefund   bool
		reasonPart string
	}{
		{name: "queued", body: `{"id":"up-1","status":"queued"}`, status: model.TaskStatusQueued},
		{name: "pending", body: `{"id":"up-1","status":"pending"}`, status: model.TaskStatusQueued},
		{name: "processing", body: `{"id":"up-1","status":"processing"}`, status: model.TaskStatusInProgress},
		{name: "in progress", body: `{"id":"up-1","status":"in_progress"}`, status: model.TaskStatusInProgress},
		{
			name:      "completed metadata URL",
			body:      `{"id":"up-1","status":"completed","metadata":{"url":"https://cdn.example/result.mp4"}}`,
			status:    model.TaskStatusSuccess,
			resultURL: "https://cdn.example/result.mp4",
		},
		{
			name:      "completed root result URL",
			body:      `{"task_id":"up-2","status":"completed","result_url":"https://cdn.example/result-2.mp4"}`,
			status:    model.TaskStatusSuccess,
			resultURL: "https://cdn.example/result-2.mp4",
		},
		{
			name:       "completed without result",
			body:       `{"id":"up-1","status":"completed","metadata":{}}`,
			status:     model.TaskStatusFailure,
			noRefund:   true,
			reasonPart: "without a result URL",
		},
		{
			name:       "failed",
			body:       `{"id":"up-1","status":"failed","error":{"message":"provider detail"}}`,
			status:     model.TaskStatusFailure,
			reasonPart: "Brioi task failed",
		},
		{
			name:       "cancelled",
			body:       `{"id":"up-1","status":"cancelled"}`,
			status:     model.TaskStatusFailure,
			reasonPart: "Brioi task failed",
		},
		{
			name:       "unknown status",
			body:       `{"id":"up-1","status":"paid_but_unrecognized"}`,
			status:     model.TaskStatusFailure,
			noRefund:   true,
			reasonPart: "administrator review",
		},
	}

	adaptor := &TaskAdaptor{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := adaptor.ParseTaskResult([]byte(test.body))

			require.NoError(t, err)
			assert.Equal(t, string(test.status), result.Status)
			assert.Equal(t, test.resultURL, result.Url)
			assert.Equal(t, test.noRefund, result.NoRefund)
			if test.reasonPart != "" {
				assert.Contains(t, result.Reason, test.reasonPart)
			}
		})
	}
}

func TestParseTaskResultRejectsMissingStatus(t *testing.T) {
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{"id":"up-1"}`))

	require.ErrorContains(t, err, "missing status")
	assert.Nil(t, result)
}

func TestFetchTaskUsesPollPathAndRetriesServerErrors(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, "/v1/videos/provider-task", r.URL.Path)
		assert.Equal(t, "Bearer poll-key", r.Header.Get("Authorization"))
		if requests == 1 {
			http.Error(w, "temporary failure", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"id":"provider-task","status":"processing"}`))
	}))
	t.Cleanup(server.Close)

	adaptor := &TaskAdaptor{}
	response, err := adaptor.FetchTask(
		server.URL+"/",
		"poll-key",
		map[string]any{"task_id": "provider-task"},
		"",
	)
	require.ErrorContains(t, err, "retryable status 503")
	assert.Nil(t, response)

	response, err = adaptor.FetchTask(
		server.URL,
		"poll-key",
		map[string]any{"task_id": "provider-task"},
		"",
	)
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, 2, requests)
}

func TestFetchTaskConvertsNonRetryableHTTPErrorToManualReviewFailure(t *testing.T) {
	const providerSecret = "https://r2.example/input.png?X-Amz-Signature=secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"missing ` + providerSecret + `"}}`))
	}))
	t.Cleanup(server.Close)

	response, err := (&TaskAdaptor{}).FetchTask(
		server.URL,
		"poll-key",
		map[string]any{"task_id": "provider-task"},
		"",
	)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.NotContains(t, string(body), providerSecret)

	result, err := (&TaskAdaptor{}).ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, string(model.TaskStatusFailure), result.Status)
	assert.True(t, result.NoRefund)
	assert.Contains(t, result.Reason, "administrator review")
}

func TestDoRequestRedactsUpstreamErrorBody(t *testing.T) {
	const signedInput = "https://r2.example/input.png?X-Amz-Signature=secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"could not fetch ` + signedInput + `"}}`))
	}))
	t.Cleanup(server.Close)

	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: server.URL,
			ApiKey:         "secret-key",
		},
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)

	response, err := adaptor.DoRequest(context, info, bytes.NewBufferString(`{"ref":[]}`))
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
	assert.NotContains(t, string(body), signedInput)
	assert.NotContains(t, string(body), "X-Amz-Signature")
}
