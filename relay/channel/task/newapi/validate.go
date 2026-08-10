package newapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	taskdto "github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/gin-gonic/gin"
)

const friendlyRequestKey = "silkroad_friendly_request"

// FriendlyRequest is the client-facing video generation payload (no upstream-native keys).
type FriendlyRequest struct {
	Model           string
	Prompt          string
	GenerationType  string
	DurationValue   string // normalized option value, e.g. "10" or "5"
	DurationSeconds int    // billing multiplier (always seconds count)
	AspectRatio     string
	Images          []string
	AudioURL        string // reference audio data URL (data:audio/mpeg;base64,...)
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *taskdto.TaskError {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	rawBytes, err := storage.Bytes()
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}

	var raw map[string]any
	if err := common.Unmarshal(rawBytes, &raw); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_json", http.StatusBadRequest)
	}

	req, err := parseFriendlyRequest(raw)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}

	modelName := info.OriginModelName
	if modelName == "" {
		modelName = req.Model
	}
	if modelName == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("model is required"), "missing_model", http.StatusBadRequest)
	}
	req.Model = modelName

	profile, ok := silkroad_setting.MatchProfile(modelName)
	if !ok {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("model %q does not match any silkroad profile", modelName),
			"unknown_model_profile",
			http.StatusBadRequest,
		)
	}

	if err := validateFriendlyRequest(&req, profile, raw); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}

	mode, ok := silkroad_setting.FindGenerationMode(req.GenerationType)
	if !ok {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("generation_type %q is not supported", req.GenerationType),
			"invalid_generation_type",
			http.StatusBadRequest,
		)
	}
	if err := checkRequireRefModel(mode, info.GetUpstreamModelName()); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_model", http.StatusBadRequest)
	}

	durOpt, ok := silkroad_setting.FindEnabledOption(profile.Durations, req.DurationValue)
	if !ok {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("duration %q is not enabled for this profile", req.DurationValue),
			"invalid_duration",
			http.StatusBadRequest,
		)
	}
	seconds, err := strconv.Atoi(durOpt.Value)
	if err != nil || seconds <= 0 || seconds > relaycommon.MaxTaskDurationSeconds {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("seconds must be between 1 and %d", relaycommon.MaxTaskDurationSeconds),
			"invalid_seconds",
			http.StatusBadRequest,
		)
	}
	req.DurationSeconds = seconds

	storeFriendlyRequest(c, info, req)
	return nil
}

func storeFriendlyRequest(c *gin.Context, info *relaycommon.RelayInfo, req FriendlyRequest) {
	info.Action = constant.TaskActionGenerate
	c.Set(friendlyRequestKey, req)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Prompt:   req.Prompt,
		Model:    req.Model,
		Images:   req.Images,
		Seconds:  strconv.Itoa(req.DurationSeconds),
		Duration: req.DurationSeconds,
	})
}

func getFriendlyRequest(c *gin.Context) (FriendlyRequest, bool) {
	v, ok := c.Get(friendlyRequestKey)
	if !ok {
		return FriendlyRequest{}, false
	}
	req, ok := v.(FriendlyRequest)
	return req, ok
}

func parseFriendlyRequest(raw map[string]any) (FriendlyRequest, error) {
	var req FriendlyRequest
	if v, ok := raw["model"].(string); ok {
		req.Model = strings.TrimSpace(v)
	}
	if v, ok := raw["prompt"].(string); ok {
		req.Prompt = v
	}
	if v, ok := raw["generation_type"].(string); ok {
		req.GenerationType = strings.TrimSpace(v)
	}
	if v, ok := raw["aspect_ratio"].(string); ok {
		req.AspectRatio = strings.TrimSpace(v)
	}

	if secs, ok := raw["seconds"]; ok {
		s, err := scalarToString(secs)
		if err != nil {
			return req, fmt.Errorf("invalid seconds: %w", err)
		}
		req.DurationValue = s
	}
	if dur, ok := raw["duration"]; ok {
		s, err := scalarToString(dur)
		if err != nil {
			return req, fmt.Errorf("invalid duration: %w", err)
		}
		if req.DurationValue != "" && req.DurationValue != s {
			return req, fmt.Errorf("seconds and duration conflict: %q vs %q", req.DurationValue, s)
		}
		req.DurationValue = s
	}

	if imgs, ok := raw["images"]; ok && imgs != nil {
		switch v := imgs.(type) {
		case []any:
			req.Images = make([]string, 0, len(v))
			for i, item := range v {
				s, ok := item.(string)
				if !ok {
					return req, fmt.Errorf("images[%d] must be a string", i)
				}
				s = strings.TrimSpace(s)
				if s != "" {
					req.Images = append(req.Images, s)
				}
			}
		case []string:
			req.Images = append([]string(nil), v...)
		default:
			return req, fmt.Errorf("images must be an array of strings")
		}
	}

	if audio, ok := raw["audio_url"]; ok && audio != nil {
		s, err := scalarToString(audio)
		if err != nil {
			return req, fmt.Errorf("invalid audio_url: %w", err)
		}
		req.AudioURL = s
	}

	return req, nil
}

func scalarToString(v any) (string, error) {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t), nil
	case float64:
		if t != float64(int(t)) {
			return "", fmt.Errorf("must be an integer")
		}
		return strconv.Itoa(int(t)), nil
	case bool:
		if t {
			return "true", nil
		}
		return "false", nil
	case int:
		return strconv.Itoa(t), nil
	case int64:
		return strconv.FormatInt(t, 10), nil
	default:
		return "", fmt.Errorf("unsupported type %T", v)
	}
}

func validateFriendlyRequest(req *FriendlyRequest, profile *silkroad_setting.Profile, raw map[string]any) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if profile == nil {
		return fmt.Errorf("profile is required")
	}
	if err := rejectUnknownTopLevelKeys(raw); err != nil {
		return err
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if req.GenerationType == "" {
		return fmt.Errorf("generation_type is required")
	}
	if req.AspectRatio == "" {
		return fmt.Errorf("aspect_ratio is required")
	}
	if req.DurationValue == "" {
		return fmt.Errorf("seconds or duration is required")
	}

	mode, ok := silkroad_setting.FindGenerationMode(req.GenerationType)
	if !ok {
		return fmt.Errorf("generation_type %q is not supported", req.GenerationType)
	}
	if _, ok := silkroad_setting.FindEnabledOption(profile.AspectRatios, req.AspectRatio); !ok {
		return fmt.Errorf("aspect_ratio %q is not enabled for this profile", req.AspectRatio)
	}
	if _, ok := silkroad_setting.FindEnabledOption(profile.Durations, req.DurationValue); !ok {
		return fmt.Errorf("duration %q is not enabled for this profile", req.DurationValue)
	}

	n := len(req.Images)
	if n < mode.ImagesMin || n > mode.ImagesMax {
		return fmt.Errorf("generation_type %q requires between %d and %d images, got %d", req.GenerationType, mode.ImagesMin, mode.ImagesMax, n)
	}

	if err := validateAudioURL(req.AudioURL, mode); err != nil {
		return err
	}
	return nil
}

// maxAudioDataURLBytes caps reference audio payload size (~8MiB decoded MP3).
const maxAudioDataURLBytes = 12 << 20

func validateAudioURL(audioURL string, mode *silkroad_setting.GenerationMode) error {
	audioURL = strings.TrimSpace(audioURL)
	if audioURL == "" {
		if mode != nil && mode.RequireAudio {
			return fmt.Errorf("audio_url is required for generation_type %q", mode.Value)
		}
		return nil
	}
	if mode == nil || !mode.AllowAudio {
		return fmt.Errorf("audio_url is not supported for this generation_type")
	}
	lower := strings.ToLower(audioURL)
	if !strings.HasPrefix(lower, "data:audio/mpeg;base64,") &&
		!strings.HasPrefix(lower, "data:audio/mp3;base64,") {
		return fmt.Errorf("audio_url must be an MP3 data URL (data:audio/mpeg;base64,...)")
	}
	if len(audioURL) > maxAudioDataURLBytes {
		return fmt.Errorf("audio_url exceeds maximum size")
	}
	return nil
}

func rejectUnknownTopLevelKeys(raw map[string]any) error {
	allowed := map[string]struct{}{
		"model":           {},
		"prompt":          {},
		"generation_type": {},
		"seconds":         {},
		"duration":        {},
		"aspect_ratio":    {},
		"images":          {},
		"audio_url":       {},
	}
	for k := range raw {
		if _, ok := allowed[k]; !ok {
			return fmt.Errorf("unknown field %q", k)
		}
	}
	return nil
}

func checkRequireRefModel(mode *silkroad_setting.GenerationMode, upstreamModel string) error {
	if mode == nil || !mode.RequireRefModel {
		return nil
	}
	if !strings.Contains(upstreamModel, "-ref") {
		return fmt.Errorf("generation_type %q requires upstream model name containing -ref, got %q", mode.Value, upstreamModel)
	}
	return nil
}
