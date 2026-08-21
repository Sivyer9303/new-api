package silkroad

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/gin-gonic/gin"
)

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	req, ok := getFriendlyRequest(c)
	if !ok || req.DurationSeconds <= 0 {
		return nil
	}
	seconds := min(req.DurationSeconds, relaycommon.MaxTaskDurationSeconds)
	return map[string]float64{"seconds": float64(seconds)}
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, ok := getFriendlyRequest(c)
	if !ok {
		return nil, fmt.Errorf("friendly request not found in context")
	}

	modelName := silkRoadProfileModelName(info, req.Model)
	profile, ok := silkroad_setting.MatchProfile(modelName)
	if !ok {
		return nil, fmt.Errorf("model %q is not available for video generation", modelName)
	}

	upstreamModel := info.GetUpstreamModelName()
	if upstreamModel == "" {
		upstreamModel = modelName
	}

	if err := stageChannelInputMedia(c, info, &req); err != nil {
		return nil, err
	}

	data, err := buildUpstreamBody(req, profile, upstreamModel)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// stageChannelInputMedia rewrites inline reference media into presigned R2 URLs
// for channels whose upstream rejects base64 payloads. Channels left on the
// default inline mode are untouched.
func stageChannelInputMedia(c *gin.Context, info *relaycommon.RelayInfo, req *FriendlyRequest) error {
	if info == nil || info.ChannelMeta == nil {
		return nil
	}
	channelSetting := info.ChannelSetting
	if !channelSetting.UsesR2VideoInputMedia() {
		return nil
	}
	ctx := c.Request.Context()
	images, err := service.StageVideoInputMediaList(ctx, info.ChannelId, req.Images)
	if err != nil {
		return err
	}
	audioURL, err := service.StageVideoInputMedia(ctx, info.ChannelId, req.AudioURL)
	if err != nil {
		return err
	}
	videos, err := service.StageVideoInputMediaList(ctx, info.ChannelId, req.ReferenceVideos)
	if err != nil {
		return err
	}
	req.Images = images
	req.AudioURL = audioURL
	req.ReferenceVideos = videos
	return nil
}

func buildUpstreamBody(req FriendlyRequest, profile *silkroad_setting.Profile, upstreamModel string) ([]byte, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is required")
	}
	mode, ok := silkroad_setting.FindGenerationMode(req.GenerationType)
	if !ok {
		return nil, fmt.Errorf("generation_type %q is not supported", req.GenerationType)
	}
	durOpt, ok := silkroad_setting.FindEnabledOption(profile.Durations, req.DurationValue)
	if !ok {
		return nil, fmt.Errorf("duration %q is not enabled", req.DurationValue)
	}
	aspectOpt, ok := silkroad_setting.FindEnabledOption(profile.AspectRatios, req.AspectRatio)
	if !ok {
		return nil, fmt.Errorf("aspect_ratio %q is not enabled", req.AspectRatio)
	}

	seconds, err := strconv.Atoi(durOpt.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid duration value %q", durOpt.Value)
	}

	body := map[string]any{
		"model":    upstreamModel,
		"prompt":   req.Prompt,
		"duration": seconds,
	}
	if resolution, ok := resolveUpstreamResolution(req.Resolution, upstreamModel); ok {
		body["resolution"] = resolution
	}

	metadata := map[string]any{
		"ratio": aspectOpt.Value,
	}
	if req.GenerateAudio != nil {
		metadata["generate_audio"] = *req.GenerateAudio
	}
	if req.CameraFixed != nil {
		metadata["camera_fixed"] = *req.CameraFixed
	}
	if req.Seed != nil {
		metadata["seed"] = *req.Seed
	}
	body["metadata"] = metadata

	if err := silkroad_setting.ApplyGenerationMedia(
		body,
		mode,
		req.Images,
		req.AudioURL,
		req.ReferenceVideos,
	); err != nil {
		return nil, err
	}

	return common.Marshal(body)
}

func resolveUpstreamResolution(explicit, upstreamModel string) (string, bool) {
	value := strings.TrimSpace(explicit)
	if value == "" {
		value = resolutionFromUpstreamModel(upstreamModel)
	}
	if value == "" {
		return "", false
	}
	normalized, err := normalizeSeedanceResolution(value)
	if err != nil {
		return "", false
	}
	return normalized, true
}

func resolutionFromUpstreamModel(modelName string) string {
	name := strings.ToLower(strings.TrimSpace(modelName))
	name = strings.TrimSuffix(name, "-ref")
	name = strings.TrimSuffix(name, "-promax")
	name = strings.TrimSuffix(name, "-global")
	for _, spec := range []struct {
		suffix string
		value  string
	}{
		{"-1080p", "1080p"},
		{"-720p", "720p"},
		{"-480p", "480p"},
		{"-4k", "4k"},
		{"-1080", "1080p"},
		{"-720", "720p"},
		{"-480", "480p"},
	} {
		if strings.HasSuffix(name, spec.suffix) {
			return spec.value
		}
	}
	return ""
}
