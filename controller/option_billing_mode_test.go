package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// per_second 计费模式只在模型匹配视频渠道档案（适配器会提供 seconds 乘数）
// 时才会真正按秒扣费；绑定错误会导致按次仅收一次单价的严重少收。
// 保存 billing_mode 时必须对这类错配给出警告。
func TestPerSecondBindingWarningFlagsUnmatchedModels(t *testing.T) {
	warning, err := perSecondBindingWarning(`{
		"seedance-2.0-720": "per_second",
		"seedance-2-5": "per_second",
		"gpt-4o": "per_second",
		"some-video-model": "per_second",
		"expr-model": "tiered_expr"
	}`)
	require.NoError(t, err)
	require.NotEmpty(t, warning)
	assert.Contains(t, warning, "gpt-4o")
	assert.Contains(t, warning, "some-video-model")
	assert.NotContains(t, warning, "seedance-2.0-720")
	assert.NotContains(t, warning, "seedance-2-5")
	assert.NotContains(t, warning, "expr-model")
}

func TestPerSecondBindingWarningEmptyWhenAllMatched(t *testing.T) {
	warning, err := perSecondBindingWarning(`{
		"seedance-2.0-720": "per_second",
		"dreamina-seedance-2-0-720": "per_second",
		"seedance-2-0-fast": "per_second",
		"chat-model": "tiered_expr"
	}`)
	require.NoError(t, err)
	assert.Empty(t, warning)
}

func TestPerSecondBindingWarningEmptyValueAndInvalidJSON(t *testing.T) {
	warning, err := perSecondBindingWarning("")
	require.NoError(t, err)
	assert.Empty(t, warning)

	_, err = perSecondBindingWarning(`{not-json`)
	require.Error(t, err)
}
