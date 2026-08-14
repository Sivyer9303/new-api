package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotFromTaskSubmitReqOmitsMediaPayloads(t *testing.T) {
	snapshot := SnapshotFromTaskSubmitReq(TaskSubmitReq{
		Prompt: "a cat",
		Model:  "sora-2",
		Images: []string{"data:image/png;base64,aaaaaaaa"},
		Size:   "1280x720",
	})

	assert.Equal(t, "sora-2", snapshot.Model)
	assert.Equal(t, "a cat", snapshot.Prompt)
	assert.Equal(t, []TaskMediaSnapshot{{Type: "image"}}, snapshot.Media)
	assert.NotContains(t, snapshot.Prompt, "data:")
}

func TestSetTaskRequestSnapshotSkipsEmpty(t *testing.T) {
	info := &RelayInfo{}
	info.SetTaskRequestSnapshot(TaskRequestSnapshot{})
	assert.Nil(t, info.TaskRelayInfo)

	info.SetTaskRequestSnapshot(TaskRequestSnapshot{Prompt: "hello"})
	require.NotNil(t, info.RequestSnapshot)
	assert.Equal(t, "hello", info.RequestSnapshot.Prompt)
}
