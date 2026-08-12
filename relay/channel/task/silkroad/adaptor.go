package silkroad

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/relay/channel/task/videocommon"
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

type openAIVideoSubmitResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Model     string `json:"model"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	CreatedAt int64  `json:"created_at"`
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

	upstreamID, err := videocommon.ExtractSubmitTaskID(responseBody)
	if err != nil {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	dResp.ID = info.PublicTaskID
	dResp.TaskID = info.PublicTaskID
	// Submit responses never expose upstream URLs. Polling and query paths
	// apply the same redaction before returning task data to clients.
	dResp.VideoURL = ""
	dResp.URL = ""
	dResp.ResultURL = ""
	publicResponse := any(dResp)
	if c.Request.URL.Path == "/v1/videos" {
		publicResponse = openAIVideoSubmitResponse{
			ID:        info.PublicTaskID,
			Object:    "video",
			Model:     info.OriginModelName,
			Status:    "queued",
			Progress:  0,
			CreatedAt: time.Now().Unix(),
		}
	}
	c.JSON(http.StatusOK, publicResponse)
	taskData, err = common.Marshal(publicResponse)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
		return
	}
	return upstreamID, taskData, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/v1/video/generations/%s", baseURL, taskID)
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

// PreferDirectTaskResultParsing keeps nested SilkRoad envelopes inside the
// provider parser instead of treating them as this application's TaskResponse.
func (a *TaskAdaptor) PreferDirectTaskResultParsing() bool {
	return true
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	result, err := videocommon.ParseProviderResult(respBody)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code:     0,
		TaskID:   result.UpstreamTaskID,
		Reason:   result.FailureReason,
		Url:      result.ResultURL,
		NoRefund: result.NoRefund,
	}
	if result.Progress > 0 {
		taskResult.Progress = strconv.Itoa(result.Progress)
	}

	switch result.Status {
	case videocommon.ProviderTaskSubmitted:
		taskResult.Status = model.TaskStatusSubmitted
	case videocommon.ProviderTaskQueued:
		taskResult.Status = model.TaskStatusQueued
	case videocommon.ProviderTaskRunning:
		taskResult.Status = model.TaskStatusInProgress
	case videocommon.ProviderTaskSucceeded:
		taskResult.Status = model.TaskStatusSuccess
	case videocommon.ProviderTaskFailed:
		taskResult.Status = model.TaskStatusFailure
	default:
		return nil, fmt.Errorf("unsupported normalized provider status %q", result.Status)
	}

	return &taskResult, nil
}
