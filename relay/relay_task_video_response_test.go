package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOpenAIVideoResponseMapsStorageLifecycleWithoutPrivateURLs(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_public_video",
		Status:     model.TaskStatusStoring,
		Progress:   "99%",
		SubmitTime: 1_800_000_000,
		Properties: model.Properties{OriginModelName: "public-seedance"},
		Data:       []byte(`{"id":"upstream-secret","video_url":"https://cdn.example/private.mp4"}`),
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID:    "upstream-secret",
			UpstreamResultURL: "https://cdn.example/private.mp4",
			StorageStatus:     "pending",
		},
	}

	body, err := buildOpenAIVideoResponse(task)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"id":"task_public_video",
		"object":"video",
		"model":"public-seedance",
		"status":"in_progress",
		"progress":99,
		"created_at":1800000000
	}`, string(body))
	assert.NotContains(t, string(body), "upstream-secret")
	assert.NotContains(t, string(body), "cdn.example")
}

func TestBuildOpenAIVideoResponseMapsDeliveryFailureAndExpiry(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_delivery_failed",
		Status:     model.TaskStatusFailure,
		Progress:   "100%",
		FailReason: service.VideoDeliveryFailureMessage("task_delivery_failed"),
		PrivateData: model.TaskPrivateData{
			StorageStatus:     "failed",
			NoAutomaticRefund: true,
		},
	}

	body, err := buildOpenAIVideoResponse(task)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"status":"failed"`)
	assert.Contains(t, string(body), `"code":"video_delivery_failed"`)
	assert.Contains(t, string(body), "task_delivery_failed")

	task.Status = model.TaskStatusExpired
	task.PrivateData.StorageStatus = "expired"
	body, err = buildOpenAIVideoResponse(task)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"code":"video_expired"`)
}

func TestBuildOpenAIVideoResponsePreservesSafeEstablishedFields(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_compatible_video",
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		SubmitTime: 1_800_000_000,
		FinishTime: 1_800_000_100,
		Data: []byte(`{
			"seconds":"12",
			"size":"1280x720",
			"remixed_from_video_id":"task_parent",
			"metadata":{
				"seed":42,
				"quality":"standard",
				"url":"https://upstream.example/private.mp4",
				"nested_url":"https://upstream.example/private.mp4"
			}
		}`),
		PrivateData: model.TaskPrivateData{
			StorageStatus:    "ready",
			StorageExpiresAt: 1_800_604_800,
		},
	}

	body, err := buildOpenAIVideoResponse(task)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"seconds":"12"`)
	assert.Contains(t, string(body), `"size":"1280x720"`)
	assert.Contains(t, string(body), `"remixed_from_video_id":"task_parent"`)
	assert.Contains(t, string(body), `"seed":42`)
	assert.NotContains(t, string(body), "upstream.example")
}

func TestTaskModel2DtoKeepsStoragePhaseProcessingAndHidesResult(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_legacy_storing",
		Status:   model.TaskStatusStoring,
		Progress: "99%",
		Data:     []byte(`{"task_id":"upstream-secret","url":"https://cdn.example/private.mp4"}`),
		PrivateData: model.TaskPrivateData{
			ResultURL:         "https://video.example.com/v1/videos/task_legacy_storing/content",
			UpstreamResultURL: "https://cdn.example/private.mp4",
			StorageStatus:     "pending",
		},
	}

	dto := TaskModel2Dto(task)
	assert.Equal(t, string(model.TaskStatusInProgress), dto.Status)
	assert.Empty(t, dto.FailReason)
	assert.Empty(t, dto.ResultURL)
	assert.Contains(t, string(dto.Data), task.TaskID)
	assert.NotContains(t, string(dto.Data), "upstream-secret")
	assert.NotContains(t, string(dto.Data), "cdn.example")
}

func TestBuildOpenAIVideoResponseCoversPublicLifecycleStates(t *testing.T) {
	tests := []struct {
		name          string
		status        model.TaskStatus
		storageStatus string
		wantStatus    string
		wantErrorCode string
	}{
		{name: "queued", status: model.TaskStatusQueued, wantStatus: "queued"},
		{name: "generating", status: model.TaskStatusInProgress, wantStatus: "in_progress"},
		{name: "storing", status: model.TaskStatusStoring, storageStatus: "pending", wantStatus: "in_progress"},
		{name: "completed", status: model.TaskStatusSuccess, storageStatus: "ready", wantStatus: "completed"},
		{name: "provider failure", status: model.TaskStatusFailure, wantStatus: "failed", wantErrorCode: "video_generation_failed"},
		{name: "delivery failure", status: model.TaskStatusFailure, storageStatus: "failed", wantStatus: "failed", wantErrorCode: "video_delivery_failed"},
		{name: "expired", status: model.TaskStatusExpired, storageStatus: "expired", wantStatus: "failed", wantErrorCode: "video_expired"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &model.Task{
				TaskID:     "task_public_state",
				Status:     tt.status,
				Progress:   "50%",
				FailReason: "provider failed",
				PrivateData: model.TaskPrivateData{
					StorageStatus:     tt.storageStatus,
					NoAutomaticRefund: tt.name == "delivery failure",
					StorageExpiresAt:  1_900_000_000,
				},
			}
			body, err := buildOpenAIVideoResponse(task)
			require.NoError(t, err)
			assert.Contains(t, string(body), `"status":"`+tt.wantStatus+`"`)
			if tt.wantErrorCode != "" {
				assert.Contains(t, string(body), `"code":"`+tt.wantErrorCode+`"`)
			}
			assert.NotContains(t, string(body), "upstream")
		})
	}
}
