package brioi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/task/videocommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/gin-gonic/gin"
)

const normalizedRequestKey = "brioi_video_request"

type normalizedRequest struct {
	request videocommon.VideoGenerateRequest
	profile resolvedProfile
}

type resolvedProfile struct {
	hard       modelProfile
	configured brioi_setting.Profile
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
	if c.Request.URL.Path != "/v1/video/generations" {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("Brioi supports only POST /v1/video/generations"),
			"unsupported_route",
			http.StatusBadRequest,
		)
	}
	if c.Request.Method != http.MethodPost {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("Brioi video generation requires POST"),
			"unsupported_method",
			http.StatusMethodNotAllowed,
		)
	}
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("Brioi video generation requires application/json"),
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

	req, profile, err := parseRequest(body, info)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}

	info.Action = constant.TaskActionFromGenerationType(req.GenerationType)
	c.Set(normalizedRequestKey, normalizedRequest{request: req, profile: profile})
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Prompt:   req.Prompt,
		Model:    req.Model,
		Duration: *req.Duration,
		Seconds:  strconv.Itoa(*req.Duration),
	})
	media := make([]relaycommon.TaskMediaSnapshot, 0, len(req.Media))
	for _, item := range req.Media {
		media = append(media, relaycommon.TaskMediaSnapshot{
			Type: string(item.Type),
			Role: string(item.Role),
		})
	}
	info.SetTaskRequestSnapshot(relaycommon.TaskRequestSnapshot{
		Model:          req.Model,
		Prompt:         req.Prompt,
		GenerationType: req.GenerationType,
		Duration:       *req.Duration,
		Seconds:        strconv.Itoa(*req.Duration),
		Resolution:     req.Resolution,
		AspectRatio:    req.AspectRatio,
		Media:          media,
	})
	return nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	stored, ok := getNormalizedRequest(c)
	if !ok || stored.request.Duration == nil {
		return nil
	}
	return map[string]float64{"seconds": float64(*stored.request.Duration)}
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

func parseRequest(body []byte, info *relaycommon.RelayInfo) (videocommon.VideoGenerateRequest, resolvedProfile, error) {
	var raw map[string]json.RawMessage
	if err := common.Unmarshal(body, &raw); err != nil {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, fmt.Errorf("invalid JSON request: %w", err)
	}
	if raw == nil {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, fmt.Errorf("request body must be a JSON object")
	}
	if err := rejectUnknownFields(raw); err != nil {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, err
	}
	// Singular video_url is reserved/rejected; use media type=video or reference_videos.
	if _, exists := raw["video_url"]; exists {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, fmt.Errorf("video_url is not supported; use media with type \"video\" or reference_videos")
	}

	publicModel, err := requestString(raw, "model")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, err
	}
	modelName := publicModel
	if info != nil {
		if mapped := strings.TrimSpace(info.GetUpstreamModelName()); mapped != "" {
			modelName = mapped
		}
	}
	hardProfile, ok := modelProfiles[modelName]
	if !ok {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, fmt.Errorf(
			"model %q is not supported by Brioi",
			modelName,
		)
	}
	configuredProfile, ok := brioi_setting.ResolveProfile(modelName)
	if !ok {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, fmt.Errorf(
			"model %q is disabled or has no enabled Brioi profile",
			modelName,
		)
	}
	profile := resolvedProfile{hard: hardProfile, configured: configuredProfile}

	prompt, err := requestString(raw, "prompt")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, err
	}
	generationType, err := requestString(raw, "generation_type")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, err
	}
	originModel := publicModel
	if info != nil {
		if name := strings.TrimSpace(info.OriginModelName); name != "" {
			originModel = name
		}
	}
	resolution, err := resolveResolution(raw, originModel, publicModel)
	if err != nil {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, err
	}
	aspectRatio, err := requestString(raw, "aspect_ratio")
	if err != nil {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, err
	}

	duration, durationSet, err := requestDuration(raw["duration"])
	if err != nil {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, fmt.Errorf("invalid duration: %w", err)
	}
	seconds, secondsSet, err := requestDuration(raw["seconds"])
	if err != nil {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, fmt.Errorf("invalid seconds: %w", err)
	}
	if !durationSet && !secondsSet {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, fmt.Errorf("duration is required")
	}
	if durationSet && secondsSet && *duration != *seconds {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, fmt.Errorf("duration and seconds conflict")
	}
	if !durationSet {
		duration = seconds
	}

	media, err := requestMedia(raw, generationType)
	if err != nil {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, err
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
	if err := validateRequest(request, profile); err != nil {
		return videocommon.VideoGenerateRequest{}, resolvedProfile{}, err
	}
	return request, profile, nil
}

func rejectUnknownFields(raw map[string]json.RawMessage) error {
	allowed := map[string]struct{}{
		"model":            {},
		"prompt":           {},
		"generation_type":  {},
		"duration":         {},
		"seconds":          {},
		"resolution":       {},
		"aspect_ratio":     {},
		"images":           {},
		"media":            {},
		"audio_url":        {},
		"video_url":        {},
		"reference_videos": {},
		"reference_audios": {},
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
	sort.Strings(unknown)
	return fmt.Errorf("unknown field %q", unknown[0])
}

func requestString(raw map[string]json.RawMessage, field string) (string, error) {
	value, exists := raw[field]
	if !exists {
		return "", fmt.Errorf("%s is required", field)
	}
	var text string
	if err := common.Unmarshal(value, &text); err != nil {
		return "", fmt.Errorf("%s must be a string", field)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return text, nil
}

// resolveResolution prefers an explicit request field, otherwise derives it from
// the public/origin model name (e.g. seedance-2-0-480p → 480p) so channel
// mappings can keep the upstream model as seedance-2-0 while encoding price tiers
// in local aliases.
func resolveResolution(
	raw map[string]json.RawMessage,
	originModel string,
	publicModel string,
) (string, error) {
	derived := resolutionFromModelName(originModel)
	if derived == "" {
		derived = resolutionFromModelName(publicModel)
	}

	value, exists := raw["resolution"]
	if !exists {
		if derived == "" {
			return "", fmt.Errorf("resolution is required")
		}
		return derived, nil
	}
	var text string
	if err := common.Unmarshal(value, &text); err != nil {
		return "", fmt.Errorf("resolution must be a string")
	}
	text = canonicalizeResolution(text)
	if text == "" {
		if derived == "" {
			return "", fmt.Errorf("resolution is required")
		}
		return derived, nil
	}
	if derived != "" && text != derived {
		source := strings.TrimSpace(originModel)
		if source == "" {
			source = strings.TrimSpace(publicModel)
		}
		return "", fmt.Errorf("resolution %q conflicts with model name %q", text, source)
	}
	return text, nil
}

func resolutionFromModelName(modelName string) string {
	name := strings.ToLower(strings.TrimSpace(modelName))
	if name == "" {
		return ""
	}
	name = strings.TrimSuffix(name, "-ref")
	for _, candidate := range []struct {
		suffix string
		value  string
	}{
		{suffix: "-1080p", value: "1080p"},
		{suffix: "-720p", value: "720p"},
		{suffix: "-480p", value: "480p"},
		{suffix: "-4k", value: "4K"},
	} {
		if strings.HasSuffix(name, candidate.suffix) {
			return candidate.value
		}
	}
	return ""
}

func canonicalizeResolution(value string) string {
	trimmed := strings.TrimSpace(value)
	switch strings.ToLower(trimmed) {
	case "480p", "720p", "1080p":
		return strings.ToLower(trimmed)
	case "4k":
		return "4K"
	default:
		return trimmed
	}
}

func requestDuration(raw json.RawMessage) (*int, bool, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" {
		return nil, false, nil
	}
	if value == "null" {
		return nil, true, fmt.Errorf("must be an integer")
	}
	if strings.HasPrefix(value, `"`) {
		var text string
		if err := common.Unmarshal(raw, &text); err != nil {
			return nil, true, fmt.Errorf("must be an integer")
		}
		value = strings.TrimSpace(text)
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, true, fmt.Errorf("must be an integer")
	}
	if parsed < 0 || parsed > int64(relaycommon.MaxTaskDurationSeconds) {
		return nil, true, fmt.Errorf(
			"must be between 0 and %d",
			relaycommon.MaxTaskDurationSeconds,
		)
	}
	duration := int(parsed)
	return &duration, true, nil
}

func requestMedia(
	raw map[string]json.RawMessage,
	generationType string,
) ([]videocommon.VideoMedia, error) {
	imagesRaw, hasImages := raw["images"]
	mediaRaw, hasMedia := raw["media"]
	videosRaw, hasVideos := raw["reference_videos"]
	audiosRaw, hasAudios := raw["reference_audios"]
	_, hasAudioURL := raw["audio_url"]
	if hasImages && hasMedia {
		return nil, fmt.Errorf("images and media cannot be used together")
	}
	if hasVideos && hasMedia {
		return nil, fmt.Errorf("reference_videos and media cannot be used together")
	}
	if (hasAudios || hasAudioURL) && hasMedia {
		return nil, fmt.Errorf("audio fields and media cannot be used together")
	}
	if hasAudios && hasAudioURL {
		return nil, fmt.Errorf("audio_url and reference_audios cannot be used together")
	}

	var media []videocommon.VideoMedia
	var err error
	if hasMedia {
		media, err = parseExplicitMedia(mediaRaw)
		if err != nil {
			return nil, err
		}
		return media, nil
	}

	if hasImages {
		var images []string
		if err := common.Unmarshal(imagesRaw, &images); err != nil {
			return nil, fmt.Errorf("images must be an array of strings")
		}
		media = make([]videocommon.VideoMedia, 0, len(images))
		for index, image := range images {
			role := videocommon.VideoMediaRoleReference
			switch generationType {
			case GenerationFirstFrame:
				role = videocommon.VideoMediaRoleFirstFrame
			case GenerationStartEnd:
				if index == 0 {
					role = videocommon.VideoMediaRoleFirstFrame
				} else if index == 1 {
					role = videocommon.VideoMediaRoleLastFrame
				}
			}
			media = append(media, videocommon.VideoMedia{
				Type:   videocommon.VideoMediaImage,
				Role:   role,
				Source: strings.TrimSpace(image),
			})
		}
	}

	if hasVideos {
		videos, err := parseStringList(videosRaw, "reference_videos")
		if err != nil {
			return nil, err
		}
		for _, video := range videos {
			media = append(media, videocommon.VideoMedia{
				Type:   videocommon.VideoMediaVideo,
				Role:   videocommon.VideoMediaRoleReference,
				Source: video,
			})
		}
	}

	if hasAudios {
		audios, err := parseStringList(audiosRaw, "reference_audios")
		if err != nil {
			return nil, err
		}
		for _, audio := range audios {
			media = append(media, videocommon.VideoMedia{
				Type:   videocommon.VideoMediaAudio,
				Role:   videocommon.VideoMediaRoleReference,
				Source: audio,
			})
		}
	}
	if hasAudioURL {
		audioURL, err := requestString(raw, "audio_url")
		if err != nil {
			return nil, fmt.Errorf("audio_url: %w", err)
		}
		media = append(media, videocommon.VideoMedia{
			Type:   videocommon.VideoMediaAudio,
			Role:   videocommon.VideoMediaRoleReference,
			Source: audioURL,
		})
	}
	return media, nil
}

func parseStringList(raw json.RawMessage, field string) ([]string, error) {
	var values []string
	if err := common.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("%s must be an array of strings", field)
	}
	out := make([]string, 0, len(values))
	for index, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%s[%d] is empty", field, index)
		}
		out = append(out, value)
	}
	return out, nil
}

func parseExplicitMedia(raw json.RawMessage) ([]videocommon.VideoMedia, error) {
	var values []map[string]json.RawMessage
	if err := common.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("media must be an array of objects")
	}
	media := make([]videocommon.VideoMedia, 0, len(values))
	for index, value := range values {
		for field := range value {
			switch field {
			case "type", "role", "source":
			default:
				return nil, fmt.Errorf("media[%d] has unknown field %q", index, field)
			}
		}
		mediaType, err := requestString(value, "type")
		if err != nil {
			return nil, fmt.Errorf("media[%d]: %w", index, err)
		}
		source, err := requestString(value, "source")
		if err != nil {
			return nil, fmt.Errorf("media[%d]: %w", index, err)
		}
		role := string(videocommon.VideoMediaRoleReference)
		if _, exists := value["role"]; exists {
			role, err = requestString(value, "role")
			if err != nil {
				return nil, fmt.Errorf("media[%d]: %w", index, err)
			}
		}
		media = append(media, videocommon.VideoMedia{
			Type:   videocommon.VideoMediaType(mediaType),
			Role:   videocommon.VideoMediaRole(role),
			Source: source,
		})
	}
	return media, nil
}

func validateRequest(request videocommon.VideoGenerateRequest, profile resolvedProfile) error {
	if utf8.RuneCountInString(request.Prompt) > 5000 {
		return fmt.Errorf("prompt must not exceed 5000 characters")
	}
	if request.Duration == nil {
		return fmt.Errorf("duration is required")
	}
	if *request.Duration < profile.hard.minDuration || *request.Duration > profile.hard.maxDuration {
		return fmt.Errorf(
			"duration must be between %d and %d for model %q",
			profile.hard.minDuration,
			profile.hard.maxDuration,
			request.Model,
		)
	}
	if !slices.Contains(profile.configured.Durations, *request.Duration) {
		return fmt.Errorf("duration %d is not enabled for model %q", *request.Duration, request.Model)
	}
	if _, ok := profile.hard.resolutions[request.Resolution]; !ok {
		return fmt.Errorf("resolution %q is not supported for model %q", request.Resolution, request.Model)
	}
	if !slices.Contains(profile.configured.Resolutions, request.Resolution) {
		return fmt.Errorf("resolution %q is not enabled for model %q", request.Resolution, request.Model)
	}
	if _, ok := profile.hard.aspectRatios[request.AspectRatio]; !ok {
		return fmt.Errorf("aspect_ratio %q is not supported for model %q", request.AspectRatio, request.Model)
	}
	if !slices.Contains(profile.configured.AspectRatios, request.AspectRatio) {
		return fmt.Errorf("aspect_ratio %q is not enabled for model %q", request.AspectRatio, request.Model)
	}
	mode, ok := brioi_setting.FindGenerationMode(profile.configured, request.GenerationType)
	if !ok {
		return fmt.Errorf(
			"generation_type %q is not enabled for model %q",
			request.GenerationType,
			request.Model,
		)
	}

	references := 0
	videos := 0
	audios := 0
	firstFrames := 0
	lastFrames := 0
	for index, media := range request.Media {
		switch media.Type {
		case videocommon.VideoMediaImage:
			if !isInlineImageDataURL(media.Source) {
				return fmt.Errorf("media[%d] must be an inline image data URL", index)
			}
			if err := service.ValidateVideoInputImageDataURL(media.Source); err != nil {
				return fmt.Errorf("media[%d] is invalid: %w", index, err)
			}
		case videocommon.VideoMediaVideo:
			if request.GenerationType != GenerationReferenceVideos {
				return fmt.Errorf("media[%d] type %q is only supported for generation_type %q", index, media.Type, GenerationReferenceVideos)
			}
			if !isInlineVideoDataURL(media.Source) {
				return fmt.Errorf("media[%d] must be an inline MP4 or MOV video data URL", index)
			}
			if err := service.ValidateVideoInputVideoDataURL(media.Source); err != nil {
				return fmt.Errorf("media[%d] is invalid: %w", index, err)
			}
		case videocommon.VideoMediaAudio:
			if request.GenerationType != GenerationReferenceVideos {
				return fmt.Errorf("media[%d] type %q is only supported for generation_type %q", index, media.Type, GenerationReferenceVideos)
			}
			if !isInlineAudioDataURL(media.Source) {
				return fmt.Errorf("media[%d] must be an inline MP3 or WAV audio data URL", index)
			}
			if err := service.ValidateVideoInputAudioDataURL(media.Source); err != nil {
				return fmt.Errorf("media[%d] is invalid: %w", index, err)
			}
		default:
			return fmt.Errorf("media[%d] type %q is not supported", index, media.Type)
		}
		switch media.Role {
		case "", videocommon.VideoMediaRoleReference:
			switch media.Type {
			case videocommon.VideoMediaVideo:
				videos++
			case videocommon.VideoMediaAudio:
				audios++
			default:
				references++
			}
		case videocommon.VideoMediaRoleFirstFrame:
			if media.Type != videocommon.VideoMediaImage {
				return fmt.Errorf("media[%d] first_frame must be an image", index)
			}
			firstFrames++
		case videocommon.VideoMediaRoleLastFrame:
			if media.Type != videocommon.VideoMediaImage {
				return fmt.Errorf("media[%d] last_frame must be an image", index)
			}
			lastFrames++
		default:
			return fmt.Errorf("media[%d] role %q is not supported", index, media.Role)
		}
	}
	if firstFrames > 1 {
		return fmt.Errorf("duplicate first_frame media is not allowed")
	}
	if lastFrames > 1 {
		return fmt.Errorf("duplicate last_frame media is not allowed")
	}
	if lastFrames > 0 && firstFrames == 0 {
		return fmt.Errorf("last_frame requires first_frame")
	}
	if (references > 0 || videos > 0 || audios > 0) && firstFrames+lastFrames > 0 {
		return fmt.Errorf("ordinary references cannot be mixed with strict frame media")
	}

	switch request.GenerationType {
	case GenerationTextToVideo:
		if len(request.Media) != 0 {
			return fmt.Errorf("text2video does not accept reference media")
		}
	case GenerationImageToVideo:
		if references != 1 || len(request.Media) != 1 {
			return fmt.Errorf("image2video requires exactly one ordinary reference image")
		}
	case GenerationMultiImage:
		maxItems := min(mode.ImagesMax, profile.hard.maxReferenceItems)
		if references < 2 || references > maxItems ||
			references != len(request.Media) {
			return fmt.Errorf(
				"multi_image requires between 2 and %d ordinary reference images",
				maxItems,
			)
		}
	case GenerationFirstFrame:
		if firstFrames != 1 || len(request.Media) != 1 {
			return fmt.Errorf("first_frame requires exactly one first-frame image")
		}
	case GenerationStartEnd:
		if firstFrames != 1 || lastFrames != 1 || len(request.Media) != 2 {
			return fmt.Errorf("first/last-frame generation requires one first_frame and one last_frame")
		}
	case GenerationReferenceVideos:
		maxVideos := min(brioi_setting.ReferenceVideosMax, profile.hard.maxReferenceItems)
		if videos < brioi_setting.ReferenceVideosMin || videos > maxVideos {
			return fmt.Errorf(
				"reference_videos requires between %d and %d reference videos",
				brioi_setting.ReferenceVideosMin,
				maxVideos,
			)
		}
		maxCompanion := min(mode.ImagesMax, brioi_setting.ReferenceMixImagesMax)
		if references > maxCompanion || firstFrames+lastFrames != 0 {
			return fmt.Errorf(
				"reference_videos allows up to %d companion reference images",
				maxCompanion,
			)
		}
		if audios > brioi_setting.ReferenceAudiosMax {
			return fmt.Errorf(
				"reference_videos allows up to %d companion reference audios",
				brioi_setting.ReferenceAudiosMax,
			)
		}
		if videos+references+audios != len(request.Media) {
			return fmt.Errorf("reference_videos only accepts ordinary reference media")
		}
		if len(request.Media) > min(profile.hard.maxReferenceItems, brioi_setting.ReferenceMixTotalMax) {
			return fmt.Errorf(
				"reference_videos allows at most %d mixed reference items",
				min(profile.hard.maxReferenceItems, brioi_setting.ReferenceMixTotalMax),
			)
		}
	default:
		return fmt.Errorf("generation_type %q is not supported by Brioi", request.GenerationType)
	}
	return nil
}

func isInlineImageDataURL(source string) bool {
	source = strings.TrimSpace(source)
	comma := strings.IndexByte(source, ',')
	if comma <= 0 || comma == len(source)-1 {
		return false
	}
	metadata := strings.ToLower(source[:comma])
	return strings.HasPrefix(metadata, "data:image/") && strings.HasSuffix(metadata, ";base64")
}

func isInlineVideoDataURL(source string) bool {
	source = strings.TrimSpace(source)
	comma := strings.IndexByte(source, ',')
	if comma <= 0 || comma == len(source)-1 {
		return false
	}
	metadata := strings.ToLower(source[:comma])
	if !strings.Contains(metadata, ";base64") {
		return false
	}
	return strings.HasPrefix(metadata, "data:video/mp4") ||
		strings.HasPrefix(metadata, "data:video/quicktime")
}

func isInlineAudioDataURL(source string) bool {
	source = strings.TrimSpace(source)
	comma := strings.IndexByte(source, ',')
	if comma <= 0 || comma == len(source)-1 {
		return false
	}
	metadata := strings.ToLower(source[:comma])
	if !strings.Contains(metadata, ";base64") {
		return false
	}
	return strings.HasPrefix(metadata, "data:audio/mpeg") ||
		strings.HasPrefix(metadata, "data:audio/mp3") ||
		strings.HasPrefix(metadata, "data:audio/wav") ||
		strings.HasPrefix(metadata, "data:audio/x-wav") ||
		strings.HasPrefix(metadata, "data:audio/wave")
}
