package brioi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/relay/channel/task/videocommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const (
	ChannelName             = "brioi"
	maxProviderResponseSize = 4 << 20
)

type stageInputFunc func(context.Context, int, string) (string, error)

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
	stageInput  stageInputFunc
}

type upstreamRequest struct {
	Model       string        `json:"model"`
	Prompt      string        `json:"prompt"`
	Duration    int           `json:"duration"`
	Resolution  string        `json:"resolution"`
	AspectRatio string        `json:"aspect_ratio"`
	Ref         []upstreamRef `json:"ref,omitempty"`
}

type upstreamRef struct {
	URL  string `json:"url"`
	Type string `json:"type"`
	Role string `json:"role,omitempty"`
}

type publicSubmitResponse struct {
	ID     string `json:"id"`
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	if info == nil || info.ChannelMeta == nil {
		a.ChannelType = 0
		a.apiKey = ""
		a.baseURL = ""
		return
	}
	a.ChannelType = info.ChannelType
	a.apiKey = info.ApiKey
	a.baseURL = strings.TrimRight(strings.TrimSpace(info.ChannelBaseUrl), "/")
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	baseURL, err := normalizeBaseURL(a.baseURL)
	if err != nil {
		return "", err
	}
	return baseURL + "/v1/videos", nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	stored, ok := getNormalizedRequest(c)
	if !ok || stored.request.Duration == nil {
		return nil, fmt.Errorf("validated Brioi request not found")
	}

	stage := a.stageInput
	if stage == nil {
		stage = service.StageVideoInputMedia
	}
	channelID := 0
	if info != nil && info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	refs := make([]upstreamRef, 0, len(stored.request.Media))
	for index, media := range stored.request.Media {
		stagedURL, err := stage(c.Request.Context(), channelID, media.Source)
		if err != nil {
			return nil, fmt.Errorf("stage Brioi media %d: %w", index, err)
		}
		parsed, err := url.Parse(strings.TrimSpace(stagedURL))
		if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" {
			return nil, fmt.Errorf("stage Brioi media %d did not return an HTTPS URL", index)
		}
		ref := upstreamRef{
			URL:  stagedURL,
			Type: string(media.Type),
		}
		if ref.Type == "" {
			ref.Type = string(videocommon.VideoMediaImage)
		}
		if media.Role != "" && media.Role != videocommon.VideoMediaRoleReference {
			ref.Role = string(media.Role)
		}
		refs = append(refs, ref)
	}

	payload := upstreamRequest{
		Model:       stored.request.Model,
		Prompt:      stored.request.Prompt,
		Duration:    *stored.request.Duration,
		Resolution:  stored.request.Resolution,
		AspectRatio: stored.request.AspectRatio,
		Ref:         refs,
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal Brioi request: %w", err)
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	requestBody io.Reader,
) (*http.Response, error) {
	response, err := channel.DoTaskApiRequest(a, c, info, requestBody)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return response, nil
	}
	_ = response.Body.Close()
	response.Body = io.NopCloser(strings.NewReader(
		`{"error":{"message":"Brioi task submission failed"}}`,
	))
	response.ContentLength = -1
	return response, nil
}

func (a *TaskAdaptor) DoResponse(
	_ *gin.Context,
	response *http.Response,
	info *relaycommon.RelayInfo,
) (submitResult *channel.TaskSubmitResponse, taskErr *taskdto.TaskError) {
	if response == nil || response.Body == nil {
		taskErr = service.TaskErrorWrapper(
			fmt.Errorf("Brioi response is empty"),
			"invalid_response",
			http.StatusBadGateway,
		)
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxProviderResponseSize+1))
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusBadGateway)
		return
	}
	if len(body) > maxProviderResponseSize {
		taskErr = service.TaskErrorWrapper(
			fmt.Errorf("Brioi response exceeds 4 MiB"),
			"invalid_response",
			http.StatusBadGateway,
		)
		return
	}
	upstreamID, err := videocommon.ExtractSubmitTaskID(body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(
			fmt.Errorf("Brioi response is missing a task identifier"),
			"invalid_response",
			http.StatusBadGateway,
		)
		return
	}

	publicID := ""
	if info != nil && info.TaskRelayInfo != nil {
		publicID = info.PublicTaskID
	}
	if publicID == "" {
		taskErr = service.TaskErrorWrapper(
			fmt.Errorf("public task identifier is missing"),
			"invalid_response",
			http.StatusInternalServerError,
		)
		return
	}
	publicResponse := publicSubmitResponse{
		ID:     publicID,
		TaskID: publicID,
		Status: "queued",
	}
	responseData, err := common.Marshal(publicResponse)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
		return
	}
	return &channel.TaskSubmitResponse{
		UpstreamTaskID: upstreamID,
		TaskData:       responseData,
		ResponseData:   responseData,
	}, nil
}

func (a *TaskAdaptor) FetchTask(
	baseURL string,
	key string,
	body map[string]any,
	proxy string,
) (*http.Response, error) {
	normalizedBase, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	taskID, ok := body["task_id"].(string)
	taskID = strings.TrimSpace(taskID)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	request, err := http.NewRequest(
		http.MethodGet,
		normalizedBase+"/v1/videos/"+url.PathEscape(taskID),
		nil,
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Accept", "application/json")

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("create Brioi proxy client: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError {
		_ = response.Body.Close()
		return nil, fmt.Errorf("Brioi poll returned retryable status %d", response.StatusCode)
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxProviderResponseSize+1))
	_ = response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read Brioi poll response: %w", err)
	}
	if len(responseBody) > maxProviderResponseSize {
		return nil, fmt.Errorf("Brioi poll response exceeds 4 MiB")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		// A non-retryable polling response must terminate safely without exposing
		// the provider body. Treat its synthetic status as unknown so billing is
		// withheld for administrator review instead of issuing an unsafe refund.
		responseBody, err = common.Marshal(struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}{
			ID:     taskID,
			Status: "poll_http_" + strconv.Itoa(response.StatusCode),
		})
		if err != nil {
			return nil, fmt.Errorf("marshal Brioi poll error: %w", err)
		}
	}
	response.Body = io.NopCloser(bytes.NewReader(responseBody))
	response.ContentLength = int64(len(responseBody))
	return response, nil
}

func (a *TaskAdaptor) ParseTaskResult(body []byte) (*relaycommon.TaskInfo, error) {
	result, err := videocommon.ParseProviderResult(body)
	if err != nil {
		return nil, fmt.Errorf("parse Brioi task result: %w", err)
	}
	if strings.TrimSpace(result.RawStatus) == "" {
		return nil, fmt.Errorf("Brioi task result is missing status")
	}

	taskResult := &relaycommon.TaskInfo{
		Code:   0,
		TaskID: result.UpstreamTaskID,
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
		if strings.TrimSpace(result.ResultURL) == "" {
			taskResult.Status = model.TaskStatusFailure
			taskResult.Progress = taskcommon.ProgressComplete
			taskResult.Reason = "Brioi completed the task without a result URL; administrator review is required"
			taskResult.NoRefund = true
			return taskResult, nil
		}
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Url = result.ResultURL
	case videocommon.ProviderTaskFailed:
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = taskcommon.ProgressComplete
		taskResult.NoRefund = result.NoRefund
		if result.NoRefund {
			taskResult.Reason = "Brioi returned an unknown task status; administrator review is required"
		} else {
			taskResult.Reason = "Brioi task failed"
		}
	default:
		return nil, fmt.Errorf("unsupported Brioi task status %q", result.Status)
	}
	return taskResult, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return append([]string(nil), modelList...)
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) PreferDirectTaskResultParsing() bool {
	return true
}

func normalizeBaseURL(raw string) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(raw), "/")
	if baseURL == "" {
		return "", fmt.Errorf("Brioi base URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" ||
		parsed.RawQuery != "" || parsed.Fragment != "" ||
		(!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
		return "", fmt.Errorf("Brioi base URL must be an absolute HTTP(S) URL")
	}
	return baseURL, nil
}
