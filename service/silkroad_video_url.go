package service

import (
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
)

const maxUpstreamVideoURLSearchDepth = 6

// ShouldHideSilkRoadUpstreamURLs reports whether outbound task payloads must
// never include upstream CDN URLs (ResultURL / Data.video_url / etc.).
func ShouldHideSilkRoadUpstreamURLs(task *model.Task) bool {
	if task == nil {
		return false
	}
	if strings.TrimSpace(task.PrivateData.StorageStatus) != "" {
		return true
	}
	return shouldSilkRoadStore(task) || silkRoadNewAPIAvoidUpstreamResultURL(task)
}

// PublicSilkRoadResultURL returns the only client-visible download URL for a
// SilkRoad-stored (or storage-bound) task: this site's content endpoint.
func PublicSilkRoadResultURL(taskID string) string {
	if strings.TrimSpace(silkroad_setting.GetSilkRoadSetting().Storage.PublicDownloadBaseURL) != "" {
		return BuildSilkRoadPublicURL(taskID)
	}
	return taskcommon.BuildProxyURL(taskID)
}

// SanitizeTaskForClient returns ResultURL and Data safe for API clients.
// When SilkRoad storage policy applies, upstream CDN fields are stripped and
// ResultURL is forced to this site's content path.
func SanitizeTaskForClient(task *model.Task) (resultURL string, data json.RawMessage) {
	if task == nil {
		return "", nil
	}
	resultURL = task.GetResultURL()
	data = task.Data
	if !ShouldHideSilkRoadUpstreamURLs(task) {
		return resultURL, data
	}
	resultURL = PublicSilkRoadResultURL(task.TaskID)
	cleaned, err := applySilkRoadDataRedaction(data)
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
		if isSilkRoadContentProxyURL(u) {
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

func isSilkRoadContentProxyURL(raw string) bool {
	u := strings.TrimSpace(raw)
	if u == "" {
		return false
	}
	return strings.Contains(u, "/v1/videos/") && strings.HasSuffix(strings.TrimRight(u, "/"), "/content")
}

// applySilkRoadSuccessStore queues local ingest, forces public ResultURL, and
// redacts upstream URLs from task.Data. Never writes upstream CDN into ResultURL.
func ApplySilkRoadSuccessStore(task *model.Task, resultURL string, responseBody []byte) {
	applySilkRoadSuccessStore(task, resultURL, responseBody)
}

func applySilkRoadSuccessStore(task *model.Task, resultURL string, responseBody []byte) {
	if task == nil {
		return
	}

	upstream := strings.TrimSpace(resultURL)
	if upstream == "" || strings.HasPrefix(upstream, "data:") || isSilkRoadContentProxyURL(upstream) {
		upstream = ExtractUpstreamVideoURLFromJSON(responseBody)
	}
	if upstream == "" || isSilkRoadContentProxyURL(upstream) {
		upstream = ExtractUpstreamVideoURLFromJSON(task.Data)
	}

	switch task.PrivateData.StorageStatus {
	case "ready":
		task.PrivateData.ResultURL = PublicSilkRoadResultURL(task.TaskID)
	default:
		if upstream != "" && !strings.HasPrefix(upstream, "data:") {
			markSilkRoadPendingStore(task, upstream)
		} else {
			// No downloadable upstream URL yet — still never expose CDN fields.
			task.PrivateData.ResultURL = PublicSilkRoadResultURL(task.TaskID)
		}
	}

	cleaned, err := applySilkRoadDataRedaction(task.Data)
	if err != nil {
		task.Data = []byte("{}")
		return
	}
	task.Data = cleaned
}
