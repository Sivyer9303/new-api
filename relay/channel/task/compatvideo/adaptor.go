package compatvideo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/relay/channel/task/videocommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/compatvideo_setting"
	"github.com/gin-gonic/gin"
)

const (
	ChannelName             = "compat_video"
	maxProviderResponseSize = 4 << 20
)

type stageInputFunc func(context.Context, int, string) (string, error)

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
	dialect     compatvideo_setting.Dialect
	stageInput  stageInputFunc
}

type publicSubmitResponse struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id,omitempty"`
	Object    string `json:"object,omitempty"`
	Model     string `json:"model,omitempty"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	CreatedAt int64  `json:"created_at,omitempty"`
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
	return baseURL + submitPath(a.dialect), nil
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
		return nil, fmt.Errorf("validated video request not found")
	}

	stage := a.stageInput
	if stage == nil {
		stage = service.StageVideoInputMedia
	}
	channelID := 0
	useR2 := false
	if info != nil && info.ChannelMeta != nil {
		channelID = info.ChannelId
		useR2 = info.ChannelSetting.UsesR2VideoInputMedia()
	}

	mediaSources := make([]string, 0, len(stored.request.Media))
	stagedMedia := make([]videocommon.VideoMedia, 0, len(stored.request.Media))
	for index, media := range stored.request.Media {
		source := strings.TrimSpace(media.Source)
		if useR2 {
			staged, err := stage(c.Request.Context(), channelID, source)
			if err != nil {
				return nil, fmt.Errorf("stage media %d: %w", index, err)
			}
			source = staged
		}
		mediaSources = append(mediaSources, source)
		item := media
		item.Source = source
		stagedMedia = append(stagedMedia, item)
	}

	payload, err := serializeUpstreamRequest(stored.profile, stored.request, stagedMedia, mediaSources)
	if err != nil {
		return nil, err
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal video request: %w", err)
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
		`{"error":{"message":"Video task submission failed"}}`,
	))
	response.ContentLength = -1
	return response, nil
}

func (a *TaskAdaptor) DoResponse(
	c *gin.Context,
	response *http.Response,
	info *relaycommon.RelayInfo,
) (submitResult *channel.TaskSubmitResponse, taskErr *taskdto.TaskError) {
	if response == nil || response.Body == nil {
		taskErr = service.TaskErrorWrapper(
			fmt.Errorf("video response is empty"),
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
			fmt.Errorf("video response exceeds 4 MiB"),
			"invalid_response",
			http.StatusBadGateway,
		)
		return
	}
	upstreamID, err := videocommon.ExtractSubmitTaskID(body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(
			fmt.Errorf("video response is missing a task identifier"),
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
	if c != nil && c.Request != nil && c.Request.URL != nil && c.Request.URL.Path == "/v1/videos" {
		publicResponse = publicSubmitResponse{
			ID:        publicID,
			Object:    "video",
			Status:    "queued",
			Progress:  0,
			CreatedAt: time.Now().Unix(),
		}
		if info != nil {
			publicResponse.Model = info.OriginModelName
		}
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
	taskID, _ := body["task_id"].(string)
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	modelName, _ := body["model"].(string)
	paths := pollPaths(compatvideo_setting.MatchProfile(modelName).Dialect)
	var lastErr error
	for _, path := range paths {
		response, fetchErr := pollOnce(normalizedBase+path+url.PathEscape(taskID), key, proxy)
		if fetchErr != nil {
			lastErr = fetchErr
			continue
		}
		if response.StatusCode == http.StatusNotFound && len(paths) > 1 {
			_ = response.Body.Close()
			continue
		}
		return finishPollResponse(response, taskID)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("video poll returned not found")
}

func (a *TaskAdaptor) ParseTaskResult(body []byte) (*relaycommon.TaskInfo, error) {
	result, err := videocommon.ParseProviderResult(body)
	if err != nil {
		return nil, fmt.Errorf("parse video task result: %w", err)
	}
	if strings.TrimSpace(result.RawStatus) == "" {
		return nil, fmt.Errorf("video task result is missing status")
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
			taskResult.Reason = "Provider completed the task without a usable result; administrator review is required"
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
			taskResult.Reason = "Provider returned an unknown task status; administrator review is required"
		} else {
			taskResult.Reason = "Provider task failed"
		}
	default:
		return nil, fmt.Errorf("unsupported video task status %q", result.Status)
	}
	return taskResult, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{
		"seedance-2-0",
		"grok-image-video",
		"grok-video-1.5",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) PreferDirectTaskResultParsing() bool {
	return true
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	stored, ok := getNormalizedRequest(c)
	if !ok || stored.request.Duration == nil {
		return nil
	}
	return map[string]float64{"seconds": float64(*stored.request.Duration)}
}

func serializeUpstreamRequest(
	profile compatvideo_setting.Profile,
	request videocommon.VideoGenerateRequest,
	media []videocommon.VideoMedia,
	sources []string,
) (map[string]any, error) {
	if request.Duration == nil {
		return nil, fmt.Errorf("duration is required")
	}
	payload := map[string]any{
		"model":  request.Model,
		"prompt": request.Prompt,
	}
	if request.AspectRatio != "" {
		payload["aspect_ratio"] = request.AspectRatio
	}
	if request.Resolution != "" {
		payload["resolution"] = request.Resolution
	}
	switch profile.Dialect {
	case compatvideo_setting.DialectOpenAIVideos:
		payload["seconds"] = strconv.Itoa(*request.Duration)
		if request.GenerateAudio != nil {
			payload["generate_audio"] = *request.GenerateAudio
		} else if profile.GenerateAudioDefault {
			payload["generate_audio"] = true
		}
		if len(media) > 0 {
			content := make([]map[string]any, 0, len(media))
			for index, item := range media {
				source := sources[index]
				entry := map[string]any{"type": string(item.Type)}
				switch item.Type {
				case videocommon.VideoMediaVideo:
					entry["video_url"] = source
				case videocommon.VideoMediaAudio:
					entry["audio_url"] = source
				default:
					entry["type"] = "image"
					entry["image_url"] = source
				}
				if item.Role != "" && item.Role != videocommon.VideoMediaRoleReference {
					entry["role"] = string(item.Role)
				}
				content = append(content, entry)
			}
			payload["content"] = content
		}
	default:
		payload["seconds"] = *request.Duration
		imageURLs := make([]string, 0, len(media))
		for index, item := range media {
			if item.Type != "" && item.Type != videocommon.VideoMediaImage {
				continue
			}
			imageURLs = append(imageURLs, sources[index])
		}
		if len(imageURLs) > 0 {
			payload["image_urls"] = imageURLs
		}
	}
	return payload, nil
}

func submitPath(dialect compatvideo_setting.Dialect) string {
	if dialect == compatvideo_setting.DialectOpenAIVideos {
		return "/v1/videos"
	}
	return "/v1/video/generations"
}

func pollPaths(dialect compatvideo_setting.Dialect) []string {
	if dialect == compatvideo_setting.DialectOpenAIVideos {
		return []string{"/v1/videos/", "/v1/video/generations/"}
	}
	return []string{"/v1/video/generations/", "/v1/videos/"}
}

func pollOnce(targetURL, key, proxy string) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Accept", "application/json")
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("create video proxy client: %w", err)
	}
	return client.Do(request)
}

func finishPollResponse(response *http.Response, taskID string) (*http.Response, error) {
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError {
		_ = response.Body.Close()
		return nil, fmt.Errorf("video poll returned retryable status %d", response.StatusCode)
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxProviderResponseSize+1))
	_ = response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read video poll response: %w", err)
	}
	if len(responseBody) > maxProviderResponseSize {
		return nil, fmt.Errorf("video poll response exceeds 4 MiB")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, err = common.Marshal(struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}{
			ID:     taskID,
			Status: "poll_http_" + strconv.Itoa(response.StatusCode),
		})
		if err != nil {
			return nil, fmt.Errorf("marshal video poll error: %w", err)
		}
	}
	response.Body = io.NopCloser(bytes.NewReader(responseBody))
	response.ContentLength = int64(len(responseBody))
	return response, nil
}

func normalizeBaseURL(raw string) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(raw), "/")
	if baseURL == "" {
		return "", fmt.Errorf("channel base URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" ||
		parsed.RawQuery != "" || parsed.Fragment != "" ||
		(!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
		return "", fmt.Errorf("channel base URL must be an absolute HTTP(S) URL")
	}
	return baseURL, nil
}
