package constant

type TaskPlatform string

const (
	TaskPlatformSuno       TaskPlatform = "suno"
	TaskPlatformMidjourney              = "mj"
)

const (
	SunoActionMusic  = "MUSIC"
	SunoActionLyrics = "LYRICS"

	TaskActionGenerate          = "generate"
	TaskActionTextGenerate      = "textGenerate"
	TaskActionFirstTailGenerate = "firstTailGenerate"
	TaskActionReferenceGenerate = "referenceGenerate"
	TaskActionRemix             = "remixGenerate"
)

var SunoModel2Action = map[string]string{
	"suno_music":  SunoActionMusic,
	"suno_lyrics": SunoActionLyrics,
}

// TaskActionFromGenerationType maps a public video generation_type onto the
// stored task action used by logs, filters, and video-task detection.
func TaskActionFromGenerationType(generationType string) string {
	switch generationType {
	case "text2video":
		return TaskActionTextGenerate
	case "first_frame", "start_end", "frames2video":
		return TaskActionFirstTailGenerate
	case "reference_videos", "reference_audio":
		return TaskActionReferenceGenerate
	case "image2video", "multi_image":
		return TaskActionGenerate
	default:
		return TaskActionGenerate
	}
}
