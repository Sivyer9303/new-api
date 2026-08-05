package newapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
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
	TaskID    string `json:"task_id,omitempty"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	VideoURL  string `json:"video_url"`
	URL       string `json:"url"`
	ResultURL string `json:"result_url"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/v1/video/generations", a.baseURL), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if ct := c.Request.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	return nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *taskdto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	var dResp taskResultResponse
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	upstreamID := dResp.ID
	if upstreamID == "" {
		upstreamID = dResp.TaskID
	}
	if upstreamID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	dResp.ID = info.PublicTaskID
	dResp.TaskID = info.PublicTaskID
	c.JSON(http.StatusOK, dResp)
	return upstreamID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/v1/video/generations/%s", baseUrl, taskID)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	res, err := unmarshalTaskResultResponse(respBody)
	if err != nil {
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
			if strings.TrimSpace(u) != "" && !isVideoContentProxyURL(u) {
				taskResult.Url = u
				break
			}
		}
		if taskResult.Url == "" {
			for _, u := range []string{res.VideoURL, res.URL, res.ResultURL} {
				if strings.TrimSpace(u) != "" {
					taskResult.Url = u
					break
				}
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

func unmarshalTaskResultResponse(respBody []byte) (taskResultResponse, error) {
	var res taskResultResponse
	if err := common.Unmarshal(respBody, &res); err != nil {
		return taskResultResponse{}, err
	}
	if strings.TrimSpace(res.Status) != "" {
		return res, nil
	}

	// Upstream New API often wraps TaskDto as {code,data:{status,data:{video_url}}}.
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := common.Unmarshal(respBody, &envelope); err != nil || len(envelope.Data) == 0 {
		return res, nil
	}
	var nested taskResultResponse
	if err := common.Unmarshal(envelope.Data, &nested); err != nil {
		return res, nil
	}
	if nested.VideoURL == "" || nested.URL == "" || nested.ResultURL == "" {
		var nestedPayload struct {
			Data json.RawMessage `json:"data"`
		}
		if err := common.Unmarshal(envelope.Data, &nestedPayload); err == nil && len(nestedPayload.Data) > 0 {
			var leaf taskResultResponse
			if err := common.Unmarshal(nestedPayload.Data, &leaf); err == nil {
				if nested.VideoURL == "" {
					nested.VideoURL = leaf.VideoURL
				}
				if nested.URL == "" {
					nested.URL = leaf.URL
				}
				if nested.ResultURL == "" {
					nested.ResultURL = leaf.ResultURL
				}
			}
		}
	}
	return nested, nil
}

func isVideoContentProxyURL(raw string) bool {
	return strings.Contains(raw, "/v1/videos/") && strings.HasSuffix(strings.TrimRight(raw, "/"), "/content")
}
