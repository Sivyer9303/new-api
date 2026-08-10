package silkroad_setting

import (
	"fmt"
	"strings"
)

// Hardcoded SilkRoad / Seedance generation modes (not configurable).
const (
	GenerationText2Video      = "text2video"
	GenerationImage2Video     = "image2video"
	GenerationMultiImage      = "multi_image"
	GenerationStartEnd        = "start_end"
	GenerationReferenceAudio  = "reference_audio"
)

// GenerationMode is a fixed generation recipe for the video tool / NewAPI adaptor.
type GenerationMode struct {
	Label           string
	Value           string
	Sort            int
	RequireRefModel bool
	RequireAudio    bool // reference_audio must provide audio_url
	AllowAudio      bool // only reference_audio may send audio_url
	ImagesMin       int
	ImagesMax       int
}

var hardcodedGenerationModes = []GenerationMode{
	{
		Label:           "文生视频",
		Value:           GenerationText2Video,
		Sort:            1,
		RequireRefModel: false,
		ImagesMin:       0,
		ImagesMax:       0,
	},
	{
		Label:           "图生视频",
		Value:           GenerationImage2Video,
		Sort:            2,
		RequireRefModel: true,
		ImagesMin:       1,
		ImagesMax:       1,
	},
	{
		Label:           "多图参考",
		Value:           GenerationMultiImage,
		Sort:            3,
		RequireRefModel: true,
		ImagesMin:       2,
		ImagesMax:       9,
	},
	{
		Label:           "首尾帧",
		Value:           GenerationStartEnd,
		Sort:            4,
		RequireRefModel: true,
		ImagesMin:       2,
		ImagesMax:       2,
	},
	{
		Label:           "参考音频",
		Value:           GenerationReferenceAudio,
		Sort:            5,
		RequireRefModel: true,
		RequireAudio:    true,
		AllowAudio:      true,
		ImagesMin:       0,
		ImagesMax:       9, // optional companion images
	},
}

// HardcodedGenerationModes returns the fixed five generation modes.
func HardcodedGenerationModes() []GenerationMode {
	out := make([]GenerationMode, len(hardcodedGenerationModes))
	copy(out, hardcodedGenerationModes)
	return out
}

// FindGenerationMode returns the hardcoded mode matching value.
func FindGenerationMode(value string) (*GenerationMode, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, false
	}
	for i := range hardcodedGenerationModes {
		m := &hardcodedGenerationModes[i]
		if m.Value == value {
			cp := *m
			return &cp, true
		}
	}
	return nil, false
}

// ApplyGenerationMedia writes upstream media fields for the given mode.
// Friendly client fields (images / audio_url) are never passed through as-is
// except when the recipe intentionally uses the same upstream key name.
func ApplyGenerationMedia(body map[string]any, mode *GenerationMode, images []string, audioURL string) error {
	if body == nil {
		return fmt.Errorf("body is required")
	}
	if mode == nil {
		return fmt.Errorf("generation mode is required")
	}

	switch mode.Value {
	case GenerationText2Video:
		// prompt-only
	case GenerationImage2Video:
		if len(images) != 1 {
			return fmt.Errorf("image2video requires exactly 1 image")
		}
		body["image"] = images[0]
	case GenerationMultiImage:
		if len(images) < 2 {
			return fmt.Errorf("multi_image requires at least 2 images")
		}
		out := make([]string, len(images))
		copy(out, images)
		body["images"] = out
	case GenerationStartEnd:
		if len(images) != 2 {
			return fmt.Errorf("start_end requires exactly 2 images")
		}
		body["first_frame"] = images[0]
		body["last_frame"] = images[1]
	case GenerationReferenceAudio:
		audioURL = strings.TrimSpace(audioURL)
		if audioURL == "" {
			return fmt.Errorf("reference_audio requires audio_url")
		}
		body["audio_url"] = audioURL
		switch len(images) {
		case 0:
			// audio only
		case 1:
			body["image"] = images[0]
		default:
			out := make([]string, len(images))
			copy(out, images)
			body["images"] = out
		}
	default:
		return fmt.Errorf("unsupported generation_type %q", mode.Value)
	}
	return nil
}
