package videocommon

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const maxProviderResponseDepth = 8

// ExtractSubmitTaskID extracts an upstream task identifier from flat or
// nested provider response envelopes.
func ExtractSubmitTaskID(raw []byte) (string, error) {
	payload, err := decodeProviderResponse(raw)
	if err != nil {
		return "", err
	}
	id := extractUpstreamTaskID(orderedResponseNodes(payload, 0, false, false))
	if id != "" {
		return id, nil
	}
	return "", fmt.Errorf("task_id is empty")
}

// ParseProviderResult parses common task fields without depending on a
// provider package or response envelope shape.
func ParseProviderResult(raw []byte) (VideoProviderResult, error) {
	payload, err := decodeProviderResponse(raw)
	if err != nil {
		return VideoProviderResult{}, err
	}
	nodes := orderedResponseNodes(payload, 0, false, false)

	result := VideoProviderResult{UpstreamTaskID: extractUpstreamTaskID(nodes)}
	for _, node := range nodes {
		if result.RawStatus == "" {
			if value, ok := node.values["status"].(string); ok {
				result.RawStatus = strings.TrimSpace(value)
			}
		}
		if result.Progress == 0 {
			result.Progress = parseProgress(node.values["progress"])
		}
		if result.FailureReason == "" {
			result.FailureReason = extractFailureReason(node.values)
		}
	}

	result.Status, result.NoRefund = NormalizeProviderStatus(result.RawStatus)
	if result.Status == ProviderTaskSucceeded {
		result.ResultURL = extractResultURL(nodes)
	}
	if result.Status == ProviderTaskFailed && result.FailureReason == "" {
		if result.NoRefund {
			result.FailureReason = fmt.Sprintf(
				"上游返回未知状态 %q，额度未退还，请管理员人工核实后处理",
				result.RawStatus,
			)
		} else {
			result.FailureReason = "task failed"
		}
	}
	return result, nil
}

// NormalizeProviderStatus maps common provider synonyms. Unknown statuses are
// terminal but non-refundable because the provider may already have charged.
func NormalizeProviderStatus(raw string) (ProviderTaskStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return ProviderTaskRunning, false
	case "submitted", "not_start":
		return ProviderTaskSubmitted, false
	case "queued", "pending":
		return ProviderTaskQueued, false
	case "in_progress", "processing", "running":
		return ProviderTaskRunning, false
	case "completed", "success", "succeeded":
		return ProviderTaskSucceeded, false
	case "failed", "failure", "cancelled", "canceled":
		return ProviderTaskFailed, false
	default:
		return ProviderTaskFailed, true
	}
}

// IsContentProxyURL reports whether a URL points back to the public local
// video content endpoint instead of an upstream result.
func IsContentProxyURL(raw string) bool {
	value := strings.TrimSpace(raw)
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return false
	}
	return strings.Contains(value, "/v1/videos/") &&
		strings.HasSuffix(strings.TrimRight(value, "/"), "/content")
}

func decodeProviderResponse(raw []byte) (any, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("provider response is empty")
	}
	var payload any
	if err := common.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

type responseNode struct {
	values          map[string]any
	resultContext   bool
	metadataContext bool
}

func orderedResponseNodes(node any, depth int, resultContext, metadataContext bool) []responseNode {
	if node == nil || depth > maxProviderResponseDepth {
		return nil
	}
	switch typed := node.(type) {
	case map[string]any:
		out := []responseNode{{
			values:          typed,
			resultContext:   resultContext,
			metadataContext: metadataContext,
		}}
		for _, key := range []string{"data", "result", "output", "metadata"} {
			value, ok := typed[key]
			if !ok {
				continue
			}
			childResultContext := resultContext || key == "result" || key == "output" || key == "metadata"
			childMetadataContext := metadataContext || key == "metadata"
			out = append(out, orderedResponseNodes(value, depth+1, childResultContext, childMetadataContext)...)
		}
		return out
	case []any:
		var out []responseNode
		for _, value := range typed {
			out = append(out, orderedResponseNodes(value, depth+1, resultContext, metadataContext)...)
		}
		return out
	default:
		return nil
	}
}

func extractUpstreamTaskID(nodes []responseNode) string {
	for _, key := range []string{"task_id", "id"} {
		for _, node := range nodes {
			if node.metadataContext {
				continue
			}
			if value, ok := node.values[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func extractResultURL(nodes []responseNode) string {
	for _, node := range nodes {
		if value := validResultURL(node.values["video_url"]); value != "" {
			return value
		}
	}
	for _, node := range nodes {
		if !node.resultContext {
			continue
		}
		for _, key := range []string{"url", "result_url"} {
			if value := validResultURL(node.values[key]); value != "" {
				return value
			}
		}
	}
	for _, node := range nodes {
		for _, key := range []string{"result_url", "url"} {
			value := validResultURL(node.values[key])
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func validResultURL(raw any) string {
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	value = strings.TrimSpace(value)
	if value == "" || IsContentProxyURL(value) {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return ""
}

func extractFailureReason(node map[string]any) string {
	for _, key := range []string{"fail_reason", "failure_reason", "reason", "message"} {
		if value, ok := node[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	errorValue, ok := node["error"]
	if !ok {
		return ""
	}
	switch typed := errorValue.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		for _, key := range []string{"message", "reason", "detail"} {
			if value, ok := typed[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func parseProgress(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case json.Number:
		parsed, err := strconv.Atoi(typed.String())
		if err == nil {
			return parsed
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(typed), "%"))
		if err == nil {
			return parsed
		}
	}
	return 0
}
