package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTasksToDtoHidesFailReasonFromUserLogs(t *testing.T) {
	tasks := []*model.Task{
		{
			TaskID:     "task_submit_failed",
			UserId:     1,
			FailReason: "Task submission did not complete.",
		},
	}

	userItems := tasksToDto(tasks, false)
	require.Len(t, userItems, 1)
	assert.Equal(t, "task_submit_failed", userItems[0].TaskID)
	assert.Empty(t, userItems[0].FailReason)
}
