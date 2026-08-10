package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectiveBillingPreference_GroupDisallowsSubscription(t *testing.T) {
	setting := ratio_setting.GetGroupRatioSetting()
	require.NotNil(t, setting.GroupAllowSubscription)
	setting.GroupAllowSubscription.Clear()
	setting.GroupAllowSubscription.Set("video", false)

	assert.Equal(t, "wallet_only", EffectiveBillingPreference("subscription_first", "video"))
	assert.Equal(t, "wallet_only", EffectiveBillingPreference("subscription_only", "video"))
	assert.Equal(t, "wallet_only", EffectiveBillingPreference("wallet_first", "video"))
	assert.Equal(t, "wallet_only", EffectiveBillingPreference("wallet_only", "video"))

	assert.Equal(t, "subscription_first", EffectiveBillingPreference("subscription_first", "default"))
	assert.Equal(t, "subscription_only", EffectiveBillingPreference("subscription_only", "default"))
	assert.Equal(t, "wallet_first", EffectiveBillingPreference("wallet_first", "default"))
}
