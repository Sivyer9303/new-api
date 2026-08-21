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
	GenerationReferenceVideos = "reference_videos"
)

// GenerationMode is a fixed generation recipe for the video tool / NewAPI adaptor.
type GenerationMode struct {
	Label           string
	Value           string
	Sort            int
	RequireRefModel bool
	RequireAudio    bool // reference_audio must provide audio_url
	AllowAudio      bool // only reference_audio may send audio_url
	RequireVideo    bool // reference_videos must provide reference_videos
	AllowVideo      bool // only reference_videos may send reference_videos
	ImagesMin       int
	ImagesMax       int
	VideosMin       int
	VideosMax       int
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
		ImagesMin:       1, // upstream rejects audio without at least one image
		ImagesMax:       9,
	},
	{
		Label:           "参考视频",
		Value:           GenerationReferenceVideos,
		Sort:            6,
		RequireRefModel: true,
		RequireVideo:    true,
		AllowVideo:      true,
		ImagesMin:       0, // companion images optional; prompt uses @ImageN / @VideoN
		ImagesMax:       9,
		VideosMin:       1,
		VideosMax:       3, // upstream ≤3 videos, each ideally ≤15s
	},
}

// HardcodedGenerationModes returns the fixed generation modes.
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

// ApplyGenerationMedia writes Elucid Seedance media fields for the given mode.
// Top-level `images` is used for image-to-video / multi-image / companion
// stills. Control inputs (first/last frame, audios, reference videos) go
// under `metadata` so the gateway actually reads them.
func ApplyGenerationMedia(
	body map[string]any,
	mode *GenerationMode,
	images []string,
	audioURL string,
	videos []string,
) error {
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
		setTopLevelImages(body, images)
	case GenerationMultiImage:
		if len(images) < 2 {
			return fmt.Errorf("multi_image requires at least 2 images")
		}
		setTopLevelImages(body, images)
	case GenerationStartEnd:
		if len(images) != 2 {
			return fmt.Errorf("start_end requires exactly 2 images")
		}
		meta := metadataObject(body)
		meta["first_frame"] = images[0]
		meta["last_frame"] = images[1]
	case GenerationReferenceAudio:
		audioURL = strings.TrimSpace(audioURL)
		if audioURL == "" {
			return fmt.Errorf("reference_audio requires audio_url")
		}
		if len(images) < 1 {
			return fmt.Errorf("reference_audio requires at least 1 image")
		}
		meta := metadataObject(body)
		meta["audios"] = []string{audioURL}
		setTopLevelImages(body, images)
	case GenerationReferenceVideos:
		if len(videos) < mode.VideosMin || len(videos) > mode.VideosMax {
			return fmt.Errorf(
				"reference_videos requires between %d and %d videos",
				mode.VideosMin,
				mode.VideosMax,
			)
		}
		meta := metadataObject(body)
		meta["reference_videos"] = copyStrings(videos)
		setTopLevelImages(body, images)
	default:
		return fmt.Errorf("unsupported generation_type %q", mode.Value)
	}
	return nil
}

func metadataObject(body map[string]any) map[string]any {
	raw, _ := body["metadata"].(map[string]any)
	if raw == nil {
		raw = map[string]any{}
		body["metadata"] = raw
	}
	return raw
}

func setTopLevelImages(body map[string]any, images []string) {
	if len(images) == 0 {
		return
	}
	body["images"] = copyStrings(images)
}

func copyStrings(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}
