package newapi

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/pkg/errors"
)

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

type taskResultResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	VideoURL  string `json:"video_url"`
	URL       string `json:"url"`
	ResultURL string `json:"result_url"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var res taskResultResponse
	if err := common.Unmarshal(respBody, &res); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}

	switch strings.ToLower(strings.TrimSpace(res.Status)) {
	case "queued", "pending":
		taskResult.Status = model.TaskStatusQueued
	case "in_progress", "processing", "running":
		taskResult.Status = model.TaskStatusInProgress
	case "completed", "success":
		taskResult.Status = model.TaskStatusSuccess
		for _, u := range []string{res.VideoURL, res.URL, res.ResultURL} {
			if u != "" {
				taskResult.Url = u
				break
			}
		}
	case "failed", "failure", "cancelled":
		taskResult.Status = model.TaskStatusFailure
		if res.Error != nil && res.Error.Message != "" {
			taskResult.Reason = res.Error.Message
		} else {
			taskResult.Reason = "task failed"
		}
	default:
		taskResult.Status = model.TaskStatusInProgress
	}

	return &taskResult, nil
}
