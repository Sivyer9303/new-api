package controller

import (
	"testing"

	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/stretchr/testify/assert"
)

func TestFinalizeSubmittingTaskFailureLeavesAcceptedTasksPollable(t *testing.T) {
	task := &model.Task{
		TaskID: "accepted_unsettled",
		Status: model.TaskStatusSubmitting,
		Quota:  100,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID:    "upstream-accepted",
			NoAutomaticRefund: true,
		},
	}
	taskErr := &taskdto.TaskError{Code: "settle_task_billing_failed"}

	assert.False(t, finalizeSubmittingTaskFailure(&relay.TaskSubmitResult{
		Task:             task,
		ProviderAccepted: true,
	}, taskErr))
	assert.Equal(t, model.TaskStatusSubmitting, task.Status)
	assert.Equal(t, "upstream-accepted", task.PrivateData.UpstreamTaskID)
	assert.True(t, task.PrivateData.NoAutomaticRefund)

	assert.False(t, finalizeSubmittingTaskFailure(&relay.TaskSubmitResult{
		Task:                 task,
		HasDurableUpstreamID: true,
	}, taskErr))
	assert.Equal(t, model.TaskStatusSubmitting, task.Status)
}
