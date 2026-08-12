package service

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestCanUseChannelProxyForVideoResultRejectsArbitraryProviderURL(t *testing.T) {
	baseURL := "https://provider.example"
	channel := &model.Channel{
		Type:    constant.ChannelTypeSilkRoad,
		BaseURL: &baseURL,
	}

	assert.True(t, canUseChannelProxyForVideoResult(
		"https://provider.example/v1/videos/task/content",
		channel,
		false,
	))
	assert.False(t, canUseChannelProxyForVideoResult(
		"https://provider-controlled.example/internal-target",
		channel,
		false,
	))
}
