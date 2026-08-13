package service

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/setting"
)

const maxUpstreamVideoURLSearchDepth = 6

// ShouldHideVideoUpstreamURLs reports whether outbound task payloads must
// never include upstream CDN URLs (ResultURL / Data.video_url / etc.).
func ShouldHideVideoUpstreamURLs(task *model.Task) bool {
	if task == nil {
		return false
	}
	return IsVideoTask(task)
}

// ShouldHideSilkRoadUpstreamURLs is retained for callers compiled against the
// legacy provider-named entry point.
func ShouldHideSilkRoadUpstreamURLs(task *model.Task) bool {
	return ShouldHideVideoUpstreamURLs(task)
}

func IsVideoTask(task *model.Task) bool {
	if task == nil {
		return false
	}
	if task.PrivateData.VideoTask ||
		strings.TrimSpace(task.PrivateData.StorageStatus) != "" ||
		strings.TrimSpace(task.PrivateData.UpstreamResultURL) != "" ||
		strings.TrimSpace(task.PrivateData.StorageObjectKey) != "" ||
		strings.TrimSpace(task.PrivateData.StoragePath) != "" ||
		task.PrivateData.StorageReadyAt > 0 ||
		task.PrivateData.StorageExpiresAt > 0 {
		return true
	}
	// Historical tasks predate the provider-neutral marker. Keep the original
	// action/provider fallbacks until those rows have been backfilled, otherwise
	// their signed upstream result URLs can become client-visible again.
	switch task.Action {
	case constant.TaskActionGenerate,
		constant.TaskActionTextGenerate,
		constant.TaskActionFirstTailGenerate,
		constant.TaskActionReferenceGenerate,
		constant.TaskActionRemix:
		return true
	}
	switch strings.ToLower(string(task.Platform)) {
	case "kling", "jimeng", "sora", "vidu", "doubao":
		return true
	}
	return task.Platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI)) ||
		task.Platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSilkRoad))
}

// PublicVideoResultURL returns the only client-visible download URL for a
// stored (or storage-bound) video task: this site's content endpoint.
func PublicVideoResultURL(taskID string) string {
	if strings.TrimSpace(setting.GetEffectiveVideoSetting().Storage.PublicDownloadBaseURL) != "" {
		return BuildVideoPublicURL(taskID)
	}
	return taskcommon.BuildProxyURL(taskID)
}

func PublicSilkRoadResultURL(taskID string) string {
	return PublicVideoResultURL(taskID)
}

// SanitizeTaskForClient returns ResultURL and Data safe for API clients.
// Video tasks never expose upstream CDN fields; ready results use this site's
// content path.
func SanitizeTaskForClient(task *model.Task) (resultURL string, data json.RawMessage) {
	if task == nil {
		return "", nil
	}
	resultURL = task.GetResultURL()
	data = task.Data
	if !ShouldHideVideoUpstreamURLs(task) {
		return resultURL, data
	}
	if task.Status == model.TaskStatusSuccess && task.PrivateData.StorageStatus == "ready" {
		resultURL = PublicVideoResultURL(task.TaskID)
	} else {
		resultURL = ""
	}
	cleaned, err := applyVideoDataRedaction(data, task.TaskID)
	if err != nil {
		return resultURL, json.RawMessage(`{}`)
	}
	return resultURL, cleaned
}

// ExtractUpstreamVideoURLFromJSON walks nested JSON for a playable http(s) URL,
// skipping self-referential /v1/videos/.../content proxy paths.
func ExtractUpstreamVideoURLFromJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var payload any
	if err := common.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	for _, candidate := range collectUpstreamVideoURLCandidates(payload, 0) {
		u := strings.TrimSpace(candidate)
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			continue
		}
		if isVideoContentProxyURL(u) {
			continue
		}
		return u
	}
	return ""
}

func collectUpstreamVideoURLCandidates(node any, depth int) []string {
	if node == nil || depth > maxUpstreamVideoURLSearchDepth {
		return nil
	}
	switch typed := node.(type) {
	case map[string]any:
		var out []string
		for _, key := range []string{"video_url", "url", "result_url"} {
			if s, ok := typed[key].(string); ok {
				out = append(out, s)
			}
		}
		for _, value := range typed {
			out = append(out, collectUpstreamVideoURLCandidates(value, depth+1)...)
		}
		return out
	case []any:
		var out []string
		for _, value := range typed {
			out = append(out, collectUpstreamVideoURLCandidates(value, depth+1)...)
		}
		return out
	default:
		return nil
	}
}

func isVideoContentProxyURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil ||
		!strings.HasPrefix(parsed.Path, "/v1/videos/") ||
		!strings.HasSuffix(strings.TrimRight(parsed.Path, "/"), "/content") {
		return false
	}
	if !parsed.IsAbs() {
		return strings.HasPrefix(raw, "/")
	}
	publicBase := strings.TrimSpace(
		setting.GetEffectiveVideoSetting().Storage.PublicDownloadBaseURL,
	)
	base, err := url.Parse(publicBase)
	if err != nil || !base.IsAbs() {
		return false
	}
	return strings.EqualFold(parsed.Scheme, base.Scheme) &&
		strings.EqualFold(parsed.Host, base.Host)
}

func isSilkRoadContentProxyURL(raw string) bool {
	return isVideoContentProxyURL(raw)
}

// ApplyVideoSuccessStore queues result ingest, forces a public ResultURL, and
// redacts upstream URLs from task.Data. Never writes upstream CDN into ResultURL.
func ApplyVideoSuccessStore(task *model.Task, resultURL string, responseBody []byte) {
	applyVideoSuccessStore(task, resultURL, responseBody)
}

func ApplySilkRoadSuccessStore(task *model.Task, resultURL string, responseBody []byte) {
	ApplyVideoSuccessStore(task, resultURL, responseBody)
}

func applyVideoSuccessStore(task *model.Task, resultURL string, responseBody []byte) {
	if task == nil {
		return
	}

	upstream := strings.TrimSpace(resultURL)
	if upstream == "" || isVideoContentProxyURL(upstream) {
		upstream = ExtractUpstreamVideoURLFromJSON(responseBody)
	}
	if upstream == "" || isVideoContentProxyURL(upstream) {
		upstream = ExtractUpstreamVideoURLFromJSON(task.Data)
	}

	switch task.PrivateData.StorageStatus {
	case "ready":
		task.PrivateData.ResultURL = PublicVideoResultURL(task.TaskID)
	default:
		markVideoPendingStore(task, upstream)
	}

	cleaned, err := applyVideoDataRedaction(task.Data, task.TaskID)
	if err != nil {
		task.Data = []byte("{}")
		return
	}
	task.Data = cleaned
}

func applySilkRoadSuccessStore(task *model.Task, resultURL string, responseBody []byte) {
	applyVideoSuccessStore(task, resultURL, responseBody)
}
