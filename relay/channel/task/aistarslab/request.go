package aistarslab

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/QuantumNous/new-api/setting/aistarslab_setting"
	"github.com/QuantumNous/new-api/setting/video_setting"
	"github.com/gin-gonic/gin"
)

const normalizedRequestKey = "aistarslab_video_request"

type normalizedRequest struct {
	request videocommon.VideoGenerateRequest
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
	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	isJSON := strings.HasPrefix(contentType, "application/json")
	isMultipart := strings.HasPrefix(contentType, "multipart/form-data")
	if !isJSON && (path != "/v1/videos" || !isMultipart) {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("video generation requires application/json%s", map[bool]string{true: " or multipart/form-data", false: ""}[path == "/v1/videos"]),
			"invalid_content_type",
			http.StatusBadRequest,
		)
	}

	var body []byte
	var err error
	if isMultipart {
		body, err = parseOpenAIVideosMultipart(c)
	} else {
		storage, storageErr := common.GetBodyStorage(c)
		if storageErr != nil {
			return service.TaskErrorWrapperLocal(storageErr, "invalid_request", http.StatusBadRequest)
		}
		body, err = storage.Bytes()
	}
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}

	request, err := parseRequestForPath(path, body, info)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	storeNormalizedRequest(c, info, request)
	return nil
}

func parseRequestForPath(_ string, body []byte, info *relaycommon.RelayInfo) (
	videocommon.VideoGenerateRequest,
	error,
) {
	var raw map[string]json.RawMessage
	if err := common.Unmarshal(body, &raw); err != nil {
		return videocommon.VideoGenerateRequest{}, fmt.Errorf("invalid JSON request: %w", err)
	}
	if raw == nil {
		return videocommon.VideoGenerateRequest{}, fmt.Errorf("request body must be a JSON object")
	}
	if err := normalizeOpenAIVideosFields(raw); err != nil {
		return videocommon.VideoGenerateRequest{}, err
	}
	if err := rejectUnknownFields(raw); err != nil {
		return videocommon.VideoGenerateRequest{}, err
	}

	publicModel, err := requestString(raw, "model")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, err
	}
	modelName := publicModel
	if info != nil {
		if mapped := strings.TrimSpace(info.GetUpstreamModelName()); mapped != "" {
			modelName = mapped
		}
	}
	if strings.TrimSpace(modelName) == "" {
		return videocommon.VideoGenerateRequest{}, fmt.Errorf("model is required")
	}

	prompt, err := requestString(raw, "prompt")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, err
	}
	generationType, err := requestString(raw, "generation_type")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, err
	}
	aspectRatio, err := requestString(raw, "aspect_ratio")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, err
	}
	resolution, err := requestString(raw, "resolution")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, err
	}

	duration, durationSet, err := requestDuration(raw["duration"])
	if err != nil {
		return videocommon.VideoGenerateRequest{}, fmt.Errorf("invalid duration: %w", err)
	}
	seconds, secondsSet, err := requestDuration(raw["seconds"])
	if err != nil {
		return videocommon.VideoGenerateRequest{}, fmt.Errorf("invalid seconds: %w", err)
	}
	if !durationSet && !secondsSet {
		return videocommon.VideoGenerateRequest{}, fmt.Errorf("duration is required")
	}
	if durationSet && secondsSet && *duration != *seconds {
		return videocommon.VideoGenerateRequest{}, fmt.Errorf("duration and seconds conflict")
	}
	if !durationSet {
		duration = seconds
	}
	if *duration <= 0 || *duration > relaycommon.MaxTaskDurationSeconds {
		return videocommon.VideoGenerateRequest{}, fmt.Errorf(
			"seconds must be between 1 and %d",
			relaycommon.MaxTaskDurationSeconds,
		)
	}

	media, err := requestMedia(raw)
	if err != nil {
		return videocommon.VideoGenerateRequest{}, err
	}
	if generationType == "" {
		generationType = inferredGenerationType(media)
	}
	if err := aistarslab_setting.ValidateGenerationTypeForPublicModel(
		generationType,
		publicModelForCapabilities(info, publicModel),
	); err != nil {
		return videocommon.VideoGenerateRequest{}, err
	}
	request := videocommon.VideoGenerateRequest{
		Model:          modelName,
		Prompt:         strings.TrimSpace(prompt),
		GenerationType: generationType,
		Duration:       duration,
		Resolution:     resolution,
		AspectRatio:    aspectRatio,
		Media:          media,
	}
	if err := validateRequest(request); err != nil {
		return videocommon.VideoGenerateRequest{}, err
	}
	if err := validateReferenceVideoCount(
		publicModelForCapabilities(info, publicModel),
		request,
	); err != nil {
		return videocommon.VideoGenerateRequest{}, err
	}
	return request, nil
}

func parseOpenAIVideosMultipart(c *gin.Context) ([]byte, error) {
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return nil, err
	}
	raw := make(map[string]any, len(form.Value)+1)
	for key, values := range form.Value {
		if len(values) > 0 {
			raw[key] = values[len(values)-1]
		}
	}
	files := form.File["input_reference"]
	if len(files) > 1 {
		return nil, fmt.Errorf("input_reference accepts at most one file")
	}
	if len(files) == 1 {
		fileHeader := files[0]
		file, err := fileHeader.Open()
		if err != nil {
			return nil, err
		}
		defer file.Close()

		contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
		limits := video_setting.GetVideoSetting().UploadLimits
		video_setting.NormalizeUploadLimitsSetting(&limits)
		maxBytes := limits.MaxBytesForContentType(contentType)
		if fileHeader.Size > maxBytes {
			return nil, fmt.Errorf("input_reference exceeds maximum size")
		}
		content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
		if err != nil {
			return nil, err
		}
		if int64(len(content)) > maxBytes {
			return nil, fmt.Errorf("input_reference exceeds maximum size")
		}
		if contentType == "" || strings.EqualFold(contentType, "application/octet-stream") {
			contentType = http.DetectContentType(content)
		}
		raw["input_reference"] = "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(content)
	}
	return common.Marshal(raw)
}

func normalizeOpenAIVideosFields(raw map[string]json.RawMessage) error {
	if _, hasMedia := raw["media"]; hasMedia {
		if _, hasInput := raw["input_reference"]; hasInput {
			return fmt.Errorf("input_reference and media cannot be used together")
		}
	}
	if input, err := requestString(raw, "input_reference"); err != nil {
		return err
	} else if input != "" {
		media, marshalErr := common.Marshal([]videocommon.VideoMedia{{
			Type:   videocommon.VideoMediaImage,
			Role:   videocommon.VideoMediaRoleReference,
			Source: input,
		}})
		if marshalErr != nil {
			return marshalErr
		}
		raw["media"] = media
	}
	delete(raw, "input_reference")

	if image, err := requestString(raw, "image"); err != nil {
		return err
	} else if image != "" {
		if err := appendImageSources(raw, []string{image}); err != nil {
			return err
		}
	}
	delete(raw, "image")

	if size, err := requestString(raw, "size"); err != nil {
		return err
	} else if size != "" {
		aspectRatio, resolution, ok := parseSize(strings.TrimSpace(size))
		if !ok {
			return fmt.Errorf("size %q is not supported", size)
		}
		if _, set := raw["aspect_ratio"]; !set && aspectRatio != "" {
			value, marshalErr := common.Marshal(aspectRatio)
			if marshalErr != nil {
				return marshalErr
			}
			raw["aspect_ratio"] = value
		}
		if _, set := raw["resolution"]; !set && resolution != "" {
			value, marshalErr := common.Marshal(resolution)
			if marshalErr != nil {
				return marshalErr
			}
			raw["resolution"] = value
		}
	}
	delete(raw, "size")

	if err := mergeMetadata(raw); err != nil {
		return err
	}
	if err := requestOptionalN(raw); err != nil {
		return err
	}
	delete(raw, "metadata")
	delete(raw, "n")
	return nil
}

func mergeMetadata(raw map[string]json.RawMessage) error {
	value, ok := raw["metadata"]
	if !ok || len(value) == 0 {
		return nil
	}
	var metadata map[string]json.RawMessage
	if err := common.Unmarshal(value, &metadata); err != nil {
		return fmt.Errorf("metadata must be an object")
	}
	if mode, err := requestString(metadata, "mode_type"); err != nil {
		return err
	} else if mode != "" {
		if _, set := raw["generation_type"]; !set {
			encoded, marshalErr := common.Marshal(mode)
			if marshalErr != nil {
				return marshalErr
			}
			raw["generation_type"] = encoded
		}
	}
	if resolution, err := requestString(metadata, "resolution"); err != nil {
		return err
	} else if resolution != "" {
		if _, set := raw["resolution"]; !set {
			encoded, marshalErr := common.Marshal(resolution)
			if marshalErr != nil {
				return marshalErr
			}
			raw["resolution"] = encoded
		}
	}
	if size, err := requestString(metadata, "size"); err != nil {
		return err
	} else if size != "" {
		if _, set := raw["aspect_ratio"]; !set {
			encoded, marshalErr := common.Marshal(size)
			if marshalErr != nil {
				return marshalErr
			}
			raw["aspect_ratio"] = encoded
		}
	}
	for _, key := range []string{"images", "videos", "audios"} {
		urls, err := requestStringList(metadata, key)
		if err != nil {
			return err
		}
		if len(urls) == 0 {
			continue
		}
		switch key {
		case "images":
			if err := appendImageSources(raw, urls); err != nil {
				return err
			}
		case "videos":
			if err := appendTypedSources(raw, videocommon.VideoMediaVideo, urls); err != nil {
				return err
			}
		case "audios":
			if err := appendTypedSources(raw, videocommon.VideoMediaAudio, urls); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseSize(size string) (string, string, bool) {
	normalized := strings.ToLower(strings.ReplaceAll(size, " ", ""))
	switch normalized {
	case "1280x720", "1792x1024":
		return "16:9", "720p", true
	case "720x1280", "1024x1792":
		return "9:16", "720p", true
	case "1024x1024":
		return "1:1", "720p", true
	case "1920x1080":
		return "16:9", "1080p", true
	case "1080x1920":
		return "9:16", "1080p", true
	}
	if strings.Contains(normalized, ":") {
		return size, "", true
	}
	return "", "", false
}

func inferredGenerationType(media []videocommon.VideoMedia) string {
	first, last, images := 0, 0, 0
	for _, item := range media {
		if item.Type != "" && item.Type != videocommon.VideoMediaImage {
			continue
		}
		images++
		switch item.Role {
		case videocommon.VideoMediaRoleFirstFrame:
			first++
		case videocommon.VideoMediaRoleLastFrame:
			last++
		}
	}
	if first == 1 && last == 1 && images == 2 {
		return aistarslab_setting.GenerationFrames2Video
	}
	if images > 0 {
		return aistarslab_setting.GenerationImage2Video
	}
	return aistarslab_setting.GenerationText2Video
}

func validateRequest(request videocommon.VideoGenerateRequest) error {
	if strings.TrimSpace(request.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	images, videos, audios := 0, 0, 0
	first, last := 0, 0
	for index, media := range request.Media {
		source := strings.TrimSpace(media.Source)
		if source == "" {
			return fmt.Errorf("media[%d].source is required", index)
		}
		if err := validateMediaSource(media); err != nil {
			return fmt.Errorf("media[%d]: %w", index, err)
		}
		switch media.Type {
		case "", videocommon.VideoMediaImage:
			images++
			switch media.Role {
			case videocommon.VideoMediaRoleFirstFrame:
				first++
			case videocommon.VideoMediaRoleLastFrame:
				last++
			case "", videocommon.VideoMediaRoleReference:
			default:
				return fmt.Errorf("media[%d].role is not supported", index)
			}
		case videocommon.VideoMediaVideo:
			videos++
		case videocommon.VideoMediaAudio:
			audios++
		default:
			return fmt.Errorf("media[%d].type is not supported", index)
		}
	}
	switch request.GenerationType {
	case aistarslab_setting.GenerationText2Video:
		if images > 0 || videos > 0 || audios > 0 {
			return fmt.Errorf("text2video does not accept reference media")
		}
	case aistarslab_setting.GenerationImage2Video:
		if images < 1 {
			return fmt.Errorf("image2video requires at least 1 reference image")
		}
	case aistarslab_setting.GenerationFrames2Video:
		if images != 2 || first != 1 || last != 1 {
			return fmt.Errorf("frames2video requires first_frame and last_frame images")
		}
		if videos > 0 || audios > 0 {
			return fmt.Errorf("frames2video does not accept reference video or audio")
		}
	default:
		return fmt.Errorf("generation_type %q is not supported", request.GenerationType)
	}
	return nil
}

func validateReferenceVideoCount(
	publicModel string,
	request videocommon.VideoGenerateRequest,
) error {
	videos := 0
	for _, media := range request.Media {
		if media.Type == videocommon.VideoMediaVideo {
			videos++
		}
	}
	return aistarslab_setting.ValidateReferenceVideoCount(
		publicModel,
		request.GenerationType,
		videos,
	)
}

func validateMediaSource(media videocommon.VideoMedia) error {
	source := strings.TrimSpace(media.Source)
	lower := strings.ToLower(source)
	if strings.HasPrefix(lower, "data:") {
		switch media.Type {
		case "", videocommon.VideoMediaImage:
			return service.ValidateVideoInputImageDataURL(source)
		case videocommon.VideoMediaVideo:
			return service.ValidateVideoInputVideoDataURL(source)
		case videocommon.VideoMediaAudio:
			return service.ValidateVideoInputAudioDataURL(source)
		default:
			return fmt.Errorf("media type is not supported")
		}
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

func publicModelForCapabilities(info *relaycommon.RelayInfo, requestModel string) string {
	if info != nil {
		if origin := strings.TrimSpace(info.GetOriginModelName()); origin != "" {
			return origin
		}
	}
	return strings.TrimSpace(requestModel)
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

func requestStringList(raw map[string]json.RawMessage, key string) ([]string, error) {
	value, ok := raw[key]
	if !ok || len(value) == 0 {
		return nil, nil
	}
	var items []string
	if err := common.Unmarshal(value, &items); err != nil {
		return nil, fmt.Errorf("%s must be an array of strings", key)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out, nil
}

func requestOptionalN(raw map[string]json.RawMessage) error {
	value, ok := raw["n"]
	if !ok || len(value) == 0 {
		return nil
	}
	var asInt int
	if err := common.Unmarshal(value, &asInt); err == nil {
		if asInt != 1 {
			return fmt.Errorf("n must be 1")
		}
		return nil
	}
	var asString string
	if err := common.Unmarshal(value, &asString); err == nil {
		if strings.TrimSpace(asString) != "1" {
			return fmt.Errorf("n must be 1")
		}
		return nil
	}
	return fmt.Errorf("n must be 1")
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

func appendImageSources(raw map[string]json.RawMessage, urls []string) error {
	media := make([]videocommon.VideoMedia, 0, len(urls))
	if existing, err := requestMedia(raw); err != nil {
		return err
	} else {
		media = append(media, existing...)
	}
	for _, source := range urls {
		media = append(media, videocommon.VideoMedia{
			Type:   videocommon.VideoMediaImage,
			Role:   videocommon.VideoMediaRoleReference,
			Source: source,
		})
	}
	encoded, err := common.Marshal(media)
	if err != nil {
		return err
	}
	raw["media"] = encoded
	return nil
}

func appendTypedSources(raw map[string]json.RawMessage, mediaType videocommon.VideoMediaType, urls []string) error {
	media := make([]videocommon.VideoMedia, 0, len(urls))
	if existing, err := requestMedia(raw); err != nil {
		return err
	} else {
		media = append(media, existing...)
	}
	for _, source := range urls {
		media = append(media, videocommon.VideoMedia{
			Type:   mediaType,
			Source: source,
		})
	}
	encoded, err := common.Marshal(media)
	if err != nil {
		return err
	}
	raw["media"] = encoded
	return nil
}

func storeNormalizedRequest(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	request videocommon.VideoGenerateRequest,
) {
	c.Set(normalizedRequestKey, normalizedRequest{request: request})
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
