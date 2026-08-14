package constant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTaskActionFromGenerationType(t *testing.T) {
	tests := []struct {
		generationType string
		want           string
	}{
		{generationType: "text2video", want: TaskActionTextGenerate},
		{generationType: "image2video", want: TaskActionGenerate},
		{generationType: "multi_image", want: TaskActionGenerate},
		{generationType: "first_frame", want: TaskActionFirstTailGenerate},
		{generationType: "start_end", want: TaskActionFirstTailGenerate},
		{generationType: "reference_videos", want: TaskActionReferenceGenerate},
		{generationType: "reference_audio", want: TaskActionReferenceGenerate},
		{generationType: "", want: TaskActionGenerate},
		{generationType: "unknown", want: TaskActionGenerate},
	}
	for _, test := range tests {
		t.Run(test.generationType, func(t *testing.T) {
			assert.Equal(t, test.want, TaskActionFromGenerationType(test.generationType))
		})
	}
}
