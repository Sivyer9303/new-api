package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGroupAllowSubscription(t *testing.T) {
	setting := GetGroupRatioSetting()
	require.NotNil(t, setting.GroupAllowSubscription)
	setting.GroupAllowSubscription.Clear()
	setting.GroupAllowSubscription.Set("video", false)
	setting.GroupAllowSubscription.Set("vip", true)

	assert.True(t, IsGroupAllowSubscription(""))
	assert.True(t, IsGroupAllowSubscription("default"))
	assert.True(t, IsGroupAllowSubscription("vip"))
	assert.False(t, IsGroupAllowSubscription("video"))
	assert.False(t, IsGroupAllowSubscription(" video "))
}
