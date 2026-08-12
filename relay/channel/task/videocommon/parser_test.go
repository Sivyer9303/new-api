package videocommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProviderResultSupportsFlatAndNestedResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
		url  string
	}{
		{
			name: "flat",
			body: `{"id":"upstream-1","status":"completed","progress":100,"video_url":"https://cdn.example/flat.mp4"}`,
			url:  "https://cdn.example/flat.mp4",
		},
		{
			name: "nested envelope",
			body: `{"code":"success","data":{"id":"upstream-2","status":"SUCCESS","progress":100,"result_url":"http://localhost/v1/videos/public/content","data":{"video_url":"https://cdn.example/nested.mp4"}}}`,
			url:  "https://cdn.example/nested.mp4",
		},
		{
			name: "metadata url",
			body: `{"data":{"task_id":"upstream-3","status":"succeeded","progress":100,"metadata":{"url":"https://cdn.example/metadata.mp4"}}}`,
			url:  "https://cdn.example/metadata.mp4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseProviderResult([]byte(tt.body))
			require.NoError(t, err)
			assert.Equal(t, ProviderTaskSucceeded, result.Status)
			assert.Equal(t, 100, result.Progress)
			assert.Equal(t, tt.url, result.ResultURL)
		})
	}
}

func TestParseProviderResultNormalizesStatusesAndErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		status     ProviderTaskStatus
		noRefund   bool
		reasonPart string
	}{
		{name: "empty remains active", body: `{"id":"task-1"}`, status: ProviderTaskRunning},
		{name: "submitted", body: `{"status":"not_start"}`, status: ProviderTaskSubmitted},
		{name: "queued", body: `{"status":"pending"}`, status: ProviderTaskQueued},
		{name: "running", body: `{"status":"processing"}`, status: ProviderTaskRunning},
		{name: "failed with nested error", body: `{"status":"failed","error":{"message":"provider rejected prompt"}}`, status: ProviderTaskFailed, reasonPart: "provider rejected prompt"},
		{name: "failed with wrapped fail reason", body: `{"code":"success","data":{"status":"FAILURE","fail_reason":"provider moderation rejected prompt"}}`, status: ProviderTaskFailed, reasonPart: "provider moderation rejected prompt"},
		{name: "unknown requires review", body: `{"status":"mystery"}`, status: ProviderTaskFailed, noRefund: true, reasonPart: "mystery"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseProviderResult([]byte(tt.body))
			require.NoError(t, err)
			assert.Equal(t, tt.status, result.Status)
			assert.Equal(t, tt.noRefund, result.NoRefund)
			if tt.reasonPart != "" {
				assert.Contains(t, result.FailureReason, tt.reasonPart)
			}
		})
	}
}

func TestExtractSubmitTaskIDSupportsIDAndTaskID(t *testing.T) {
	id, err := ExtractSubmitTaskID([]byte(`{"id":"upstream-id"}`))
	require.NoError(t, err)
	assert.Equal(t, "upstream-id", id)

	id, err = ExtractSubmitTaskID([]byte(`{"data":{"task_id":"upstream-task-id"}}`))
	require.NoError(t, err)
	assert.Equal(t, "upstream-task-id", id)
}

func TestExtractSubmitTaskIDPrefersExplicitTaskIDAndIgnoresMetadataID(t *testing.T) {
	id, err := ExtractSubmitTaskID([]byte(`{
		"id":"request-id",
		"data":{
			"task_id":"upstream-task-id",
			"metadata":{"id":"asset-id"}
		}
	}`))
	require.NoError(t, err)
	assert.Equal(t, "upstream-task-id", id)

	result, err := ParseProviderResult([]byte(`{
		"id":"request-id",
		"data":{
			"task_id":"upstream-task-id",
			"status":"queued",
			"metadata":{"id":"asset-id"}
		}
	}`))
	require.NoError(t, err)
	assert.Equal(t, "upstream-task-id", result.UpstreamTaskID)
}

func TestParseProviderResultPrefersOutputURLOverEchoedInputURL(t *testing.T) {
	result, err := ParseProviderResult([]byte(`{
		"status":"completed",
		"data":{
			"url":"https://input.example/reference.mp4",
			"output":{"url":"https://cdn.example/result.mp4"}
		}
	}`))
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example/result.mp4", result.ResultURL)
}

func TestParseProviderResultKeepsAbsoluteUpstreamContentLikeURL(t *testing.T) {
	result, err := ParseProviderResult([]byte(`{
		"status":"completed",
		"url":"https://upstream.example/v1/videos/provider-task/content"
	}`))
	require.NoError(t, err)
	assert.Equal(t, "https://upstream.example/v1/videos/provider-task/content", result.ResultURL)
}
