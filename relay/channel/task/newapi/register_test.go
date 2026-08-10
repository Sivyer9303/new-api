package newapi_test

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTaskAdaptorNewAPI(t *testing.T) {
	a := relay.GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI)))
	require.NotNil(t, a)
	assert.Equal(t, "newapi", a.GetChannelName())
}
