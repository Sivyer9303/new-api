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

	body := map[string]any{
		"model":  upstreamModel,
		"prompt": req.Prompt,
	}

	if err := setDurationField(body, durOpt); err != nil {
		return nil, err
	}
	setNestedValue(body, aspectOpt.UpstreamKey, aspectOpt.Value)

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

func setDurationField(body map[string]any, durOpt *silkroad_setting.OptionItem) error {
	key := durOpt.UpstreamKey
	if key == "" {
		return fmt.Errorf("duration upstream_key is empty")
	}
	switch key {
	case "seconds":
		setNestedValue(body, key, durOpt.Value) // JSON string
	case "duration":
		n, err := strconv.Atoi(durOpt.Value)
		if err != nil {
			return fmt.Errorf("invalid duration value %q", durOpt.Value)
		}
		setNestedValue(body, key, n) // JSON number
	default:
		setNestedValue(body, key, durOpt.Value)
	}
	return nil
}

func setNestedValue(body map[string]any, key string, value any) {
	parts := strings.Split(key, ".")
	if len(parts) == 1 {
		body[key] = value
		return
	}
	cur := body
	for i := 0; i < len(parts)-1; i++ {
		p := parts[i]
		next, ok := cur[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[p] = next
		}
		cur = next
	}
	cur[parts[len(parts)-1]] = value
}
