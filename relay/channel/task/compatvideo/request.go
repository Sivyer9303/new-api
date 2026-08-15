package compatvideo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/task/videocommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/compatvideo_setting"
	"github.com/gin-gonic/gin"
)

const normalizedRequestKey = "compat_video_request"

type normalizedRequest struct {
	request videocommon.VideoGenerateRequest
	profile compatvideo_setting.Profile
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *taskdto.TaskError {
	if c == nil || c.Request == nil {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("request is required"),
			"invalid_request",
			http.StatusBadRequest,
		)
	}
	if info == nil || info.TaskRelayInfo == nil {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("relay task context is required"),
			"invalid_request",
			http.StatusBadRequest,
		)
	}
	path := c.Request.URL.Path
	if path != "/v1/video/generations" && path != "/v1/videos" {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("video generation requires POST /v1/video/generations or POST /v1/videos"),
			"unsupported_route",
			http.StatusBadRequest,
		)
	}
	if c.Request.Method != http.MethodPost {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("video generation requires POST"),
			"unsupported_method",
			http.StatusMethodNotAllowed,
		)
	}
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("video generation requires application/json"),
			"invalid_content_type",
			http.StatusBadRequest,
		)
	}

	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	body, err := storage.Bytes()
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}

	request, profile, err := parseRequest(body, info)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	a.dialect = profile.Dialect
	storeNormalizedRequest(c, info, request, profile)
	return nil
}

func parseRequest(body []byte, info *relaycommon.RelayInfo) (
	videocommon.VideoGenerateRequest,
	compatvideo_setting.Profile,
	error,
) {
	var raw map[string]json.RawMessage
	if err := common.Unmarshal(body, &raw); err != nil {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, fmt.Errorf("invalid JSON request: %w", err)
	}
	if raw == nil {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, fmt.Errorf("request body must be a JSON object")
	}
	if err := rejectUnknownFields(raw); err != nil {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, err
	}

	publicModel, err := requestString(raw, "model")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, err
	}
	modelName := publicModel
	if info != nil {
		if mapped := strings.TrimSpace(info.GetUpstreamModelName()); mapped != "" {
			modelName = mapped
		}
	}
	profile := compatvideo_setting.MatchProfile(modelName)

	prompt, err := requestString(raw, "prompt")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, err
	}
	generationType, err := requestString(raw, "generation_type")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, err
	}
	aspectRatio, err := requestString(raw, "aspect_ratio")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, err
	}
	resolution, _ := requestString(raw, "resolution")

	duration, durationSet, err := requestDuration(raw["duration"])
	if err != nil {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, fmt.Errorf("invalid duration: %w", err)
	}
	seconds, secondsSet, err := requestDuration(raw["seconds"])
	if err != nil {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, fmt.Errorf("invalid seconds: %w", err)
	}
	if !durationSet && !secondsSet {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, fmt.Errorf("duration is required")
	}
	if durationSet && secondsSet && *duration != *seconds {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, fmt.Errorf("duration and seconds conflict")
	}
	if !durationSet {
		duration = seconds
	}
	if *duration <= 0 || *duration > relaycommon.MaxTaskDurationSeconds {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, fmt.Errorf(
			"seconds must be between 1 and %d",
			relaycommon.MaxTaskDurationSeconds,
		)
	}

	generateAudio, err := requestOptionalBool(raw, "generate_audio")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, err
	}
	media, err := requestMedia(raw)
	if err != nil {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, err
	}
	request := videocommon.VideoGenerateRequest{
		Model:          modelName,
		Prompt:         strings.TrimSpace(prompt),
		GenerationType: generationType,
		Duration:       duration,
		Resolution:     resolution,
		AspectRatio:    aspectRatio,
		Media:          media,
		GenerateAudio:  generateAudio,
	}
	if err := validateRequest(request, profile); err != nil {
		return videocommon.VideoGenerateRequest{}, compatvideo_setting.Profile{}, err
	}
	return request, profile, nil
}

func validateRequest(request videocommon.VideoGenerateRequest, profile compatvideo_setting.Profile) error {
	if strings.TrimSpace(request.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	mode, ok := compatvideo_setting.FindGenerationMode(profile, request.GenerationType)
	if !ok {
		return fmt.Errorf("generation_type %q is not supported", request.GenerationType)
	}
	if request.Duration == nil || !compatvideo_setting.DurationAllowed(profile, *request.Duration, mode) {
		return fmt.Errorf("duration is not enabled for this model")
	}
	if request.AspectRatio == "" || !slices.Contains(profile.AspectRatios, request.AspectRatio) {
		return fmt.Errorf("aspect_ratio is not enabled for this model")
	}
	if request.Resolution != "" && !slices.Contains(profile.Resolutions, request.Resolution) {
		return fmt.Errorf("resolution is not enabled for this model")
	}
	if request.GenerateAudio != nil && *request.GenerateAudio && !profile.AllowGenerateAudio {
		return fmt.Errorf("generate_audio is not supported for this model")
	}

	images, videos, audios := 0, 0, 0
	for index, media := range request.Media {
		source := strings.TrimSpace(media.Source)
		if source == "" {
			return fmt.Errorf("media[%d].source is required", index)
		}
		if err := validateMediaSource(source); err != nil {
			return fmt.Errorf("media[%d]: %w", index, err)
		}
		switch media.Type {
		case "", videocommon.VideoMediaImage:
			images++
			if len(mode.ImageRoles) > 0 && media.Role != "" &&
				!slices.Contains(mode.ImageRoles, string(media.Role)) &&
				media.Role != videocommon.VideoMediaRoleReference {
				return fmt.Errorf("media[%d].role is not supported", index)
			}
		case videocommon.VideoMediaVideo:
			videos++
			if !mode.AllowVideo {
				return fmt.Errorf("video media is not supported for this generation_type")
			}
		case videocommon.VideoMediaAudio:
			audios++
			if !mode.AllowAudio {
				return fmt.Errorf("audio media is not supported for this generation_type")
			}
		default:
			return fmt.Errorf("media[%d].type is not supported", index)
		}
	}
	if images < mode.ImagesMin || images > mode.ImagesMax {
		return fmt.Errorf(
			"generation_type %q requires between %d and %d images, got %d",
			mode.Value,
			mode.ImagesMin,
			mode.ImagesMax,
			images,
		)
	}
	if mode.RequireVideo && videos == 0 {
		return fmt.Errorf("generation_type %q requires a reference video", mode.Value)
	}
	if videos > mode.VideosMax && mode.VideosMax == 0 && videos > 0 && !mode.AllowVideo {
		return fmt.Errorf("video media is not supported for this generation_type")
	}
	if mode.VideosMax > 0 && (videos < mode.VideosMin || videos > mode.VideosMax) {
		return fmt.Errorf(
			"generation_type %q requires between %d and %d videos, got %d",
			mode.Value,
			mode.VideosMin,
			mode.VideosMax,
			videos,
		)
	}
	if mode.RequireAudio && audios == 0 {
		return fmt.Errorf("generation_type %q requires reference audio", mode.Value)
	}
	if !mode.AllowAudio && audios > 0 {
		return fmt.Errorf("audio media is not supported for this generation_type")
	}
	return nil
}

func validateMediaSource(source string) error {
	lower := strings.ToLower(source)
	if strings.HasPrefix(lower, "data:") {
		return nil
	}
	parsed, err := url.Parse(source)
	if err != nil || parsed.Host == "" ||
		(!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
		return fmt.Errorf("source must be a data URL or an http(s) URL")
	}
	return nil
}

func rejectUnknownFields(raw map[string]json.RawMessage) error {
	allowed := map[string]struct{}{
		"model":           {},
		"prompt":          {},
		"generation_type": {},
		"duration":        {},
		"seconds":         {},
		"resolution":      {},
		"aspect_ratio":    {},
		"media":           {},
		"generate_audio":  {},
	}
	unknown := make([]string, 0)
	for key := range raw {
		if _, ok := allowed[key]; !ok {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	slices.Sort(unknown)
	return fmt.Errorf("unknown fields: %s", strings.Join(unknown, ", "))
}

func requestString(raw map[string]json.RawMessage, key string) (string, error) {
	value, ok := raw[key]
	if !ok {
		return "", nil
	}
	var text string
	if err := common.Unmarshal(value, &text); err == nil {
		return strings.TrimSpace(text), nil
	}
	return "", fmt.Errorf("%s must be a string", key)
}

func requestOptionalBool(raw map[string]json.RawMessage, key string) (*bool, error) {
	value, ok := raw[key]
	if !ok {
		return nil, nil
	}
	var flag bool
	if err := common.Unmarshal(value, &flag); err != nil {
		return nil, fmt.Errorf("%s must be a boolean", key)
	}
	return &flag, nil
}

func requestDuration(raw json.RawMessage) (*int, bool, error) {
	if len(raw) == 0 {
		return nil, false, nil
	}
	var asInt int
	if err := common.Unmarshal(raw, &asInt); err == nil {
		return &asInt, true, nil
	}
	var asFloat float64
	if err := common.Unmarshal(raw, &asFloat); err == nil {
		if asFloat != float64(int(asFloat)) {
			return nil, false, fmt.Errorf("must be an integer")
		}
		value := int(asFloat)
		return &value, true, nil
	}
	var asString string
	if err := common.Unmarshal(raw, &asString); err == nil {
		asString = strings.TrimSpace(asString)
		if asString == "" {
			return nil, false, nil
		}
		value, err := strconv.Atoi(asString)
		if err != nil {
			return nil, false, fmt.Errorf("must be an integer")
		}
		return &value, true, nil
	}
	return nil, false, fmt.Errorf("must be an integer")
}

func requestMedia(raw map[string]json.RawMessage) ([]videocommon.VideoMedia, error) {
	value, ok := raw["media"]
	if !ok || len(value) == 0 {
		return nil, nil
	}
	var items []videocommon.VideoMedia
	if err := common.Unmarshal(value, &items); err != nil {
		return nil, fmt.Errorf("media must be an array of objects")
	}
	out := make([]videocommon.VideoMedia, 0, len(items))
	for index, item := range items {
		item.Source = strings.TrimSpace(item.Source)
		if item.Source == "" {
			return nil, fmt.Errorf("media[%d].source is required", index)
		}
		out = append(out, item)
	}
	return out, nil
}

func storeNormalizedRequest(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	request videocommon.VideoGenerateRequest,
	profile compatvideo_setting.Profile,
) {
	c.Set(normalizedRequestKey, normalizedRequest{request: request, profile: profile})
	if info == nil || info.TaskRelayInfo == nil {
		return
	}
	media := make([]relaycommon.TaskMediaSnapshot, 0, len(request.Media))
	for _, item := range request.Media {
		media = append(media, relaycommon.TaskMediaSnapshot{
			Type: string(item.Type),
			Role: string(item.Role),
		})
	}
	seconds := 0
	if request.Duration != nil {
		seconds = *request.Duration
	}
	info.Action = constant.TaskActionFromGenerationType(request.GenerationType)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Prompt:   request.Prompt,
		Model:    request.Model,
		Duration: seconds,
		Seconds:  strconv.Itoa(seconds),
	})
	info.SetTaskRequestSnapshot(relaycommon.TaskRequestSnapshot{
		Model:          request.Model,
		Prompt:         request.Prompt,
		GenerationType: request.GenerationType,
		Duration:       seconds,
		Seconds:        strconv.Itoa(seconds),
		Resolution:     request.Resolution,
		AspectRatio:    request.AspectRatio,
		Media:          media,
	})
}

func getNormalizedRequest(c *gin.Context) (normalizedRequest, bool) {
	if c == nil {
		return normalizedRequest{}, false
	}
	value, ok := c.Get(normalizedRequestKey)
	if !ok {
		return normalizedRequest{}, false
	}
	request, ok := value.(normalizedRequest)
	return request, ok
}
