package common

import "strings"

// TaskMediaSnapshot records attached media kind/role without payloads or URLs.
type TaskMediaSnapshot struct {
	Type string `json:"type"`
	Role string `json:"role,omitempty"`
}

// TaskRequestSnapshot is the user-visible request summary stored on a task.
type TaskRequestSnapshot struct {
	Model          string              `json:"model,omitempty"`
	Prompt         string              `json:"prompt,omitempty"`
	GenerationType string              `json:"generation_type,omitempty"`
	Duration       int                 `json:"duration,omitempty"`
	Seconds        string              `json:"seconds,omitempty"`
	Resolution     string              `json:"resolution,omitempty"`
	AspectRatio    string              `json:"aspect_ratio,omitempty"`
	Size           string              `json:"size,omitempty"`
	Media          []TaskMediaSnapshot `json:"media,omitempty"`
}

func (s *TaskRequestSnapshot) empty() bool {
	if s == nil {
		return true
	}
	return s.Model == "" &&
		s.Prompt == "" &&
		s.GenerationType == "" &&
		s.Duration == 0 &&
		s.Seconds == "" &&
		s.Resolution == "" &&
		s.AspectRatio == "" &&
		s.Size == "" &&
		len(s.Media) == 0
}

func (info *RelayInfo) SetTaskRequestSnapshot(snapshot TaskRequestSnapshot) {
	if info == nil || snapshot.empty() {
		return
	}
	if info.TaskRelayInfo == nil {
		info.TaskRelayInfo = &TaskRelayInfo{}
	}
	copy := snapshot
	info.TaskRelayInfo.RequestSnapshot = &copy
}

func SnapshotFromTaskSubmitReq(req TaskSubmitReq) TaskRequestSnapshot {
	snapshot := TaskRequestSnapshot{
		Model:    strings.TrimSpace(req.Model),
		Prompt:   req.Prompt,
		Duration: req.Duration,
		Seconds:  strings.TrimSpace(req.Seconds),
		Size:     strings.TrimSpace(req.Size),
	}
	imageCount := len(req.Images)
	if imageCount == 0 && strings.TrimSpace(req.Image) != "" {
		imageCount = 1
	}
	if imageCount == 0 && strings.TrimSpace(req.InputReference) != "" {
		imageCount = 1
	}
	for i := 0; i < imageCount; i++ {
		snapshot.Media = append(snapshot.Media, TaskMediaSnapshot{Type: "image"})
	}
	return snapshot
}
