package silkroad_test

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTaskAdaptorSilkRoad(t *testing.T) {
	a := relay.GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSilkRoad)))
	require.NotNil(t, a)
	assert.Equal(t, "silkroad", a.GetChannelName())
}
