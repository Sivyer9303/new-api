package compatvideo_test

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTaskAdaptorCompatVideo(t *testing.T) {
	adaptor := relay.GetTaskAdaptor(
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeCompatVideo)),
	)

	require.NotNil(t, adaptor)
	assert.Equal(t, "compat_video", adaptor.GetChannelName())
}
