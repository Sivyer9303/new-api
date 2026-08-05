package newapi

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
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

	modelName := info.OriginModelName
	if modelName == "" {
		modelName = req.Model
	}
	profile, ok := silkroad_setting.MatchProfile(modelName)
	if !ok {
		return nil, fmt.Errorf("model %q does not match any silkroad profile", modelName)
	}

	upstreamModel := info.GetUpstreamModelName()
	if upstreamModel == "" {
		upstreamModel = modelName
	}

	data, err := buildUpstreamBody(req, profile, upstreamModel)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func buildUpstreamBody(req FriendlyRequest, profile *silkroad_setting.Profile, upstreamModel string) ([]byte, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is required")
	}
	gt, ok := silkroad_setting.FindGenerationType(profile, req.GenerationType)
	if !ok {
		return nil, fmt.Errorf("generation_type %q is not enabled", req.GenerationType)
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

	// Apply ExtraOptions first; generation-type UpstreamSets win on key conflicts
	// so recipe fields (e.g. reference_mode=start_end) are never clobbered by client extras.
	for key, val := range req.Extras {
		setNestedValue(body, key, coerceExtraValue(val))
	}

	for _, us := range gt.UpstreamSets {
		if us.UpstreamKey == "" {
			continue
		}
		if us.Value != "" {
			setNestedValue(body, us.UpstreamKey, coerceExtraValue(us.Value))
			continue
		}
		if us.From == "" {
			continue
		}
		val, err := resolveFromPath(us.From, req.Images)
		if err != nil {
			return nil, err
		}
		setNestedValue(body, us.UpstreamKey, val)
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

func resolveFromPath(from string, images []string) (any, error) {
	from = strings.TrimSpace(from)
	switch {
	case from == "images":
		out := make([]string, len(images))
		copy(out, images)
		return out, nil
	case strings.HasPrefix(from, "images[") && strings.HasSuffix(from, "]"):
		idxStr := from[len("images[") : len(from)-1]
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			return nil, fmt.Errorf("invalid from path %q", from)
		}
		if idx < 0 || idx >= len(images) {
			return nil, fmt.Errorf("from path %q out of range (have %d images)", from, len(images))
		}
		return images[idx], nil
	default:
		return nil, fmt.Errorf("unsupported from path %q", from)
	}
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

func coerceExtraValue(v string) any {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true":
		return true
	case "false":
		return false
	default:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		return v
	}
}
