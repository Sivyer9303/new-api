package aistarslab_test

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTaskAdaptorAIStarsLab(t *testing.T) {
	adaptor := relay.GetTaskAdaptor(
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeAIStarsLab)),
	)

	require.NotNil(t, adaptor)
	assert.Equal(t, "aistarslab", adaptor.GetChannelName())
}
