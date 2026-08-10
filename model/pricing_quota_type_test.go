package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/stretchr/testify/require"
)

func TestResolveModelQuotaType(t *testing.T) {
	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{
			"seedance-per-second":"per_second",
			"expr-model":"tiered_expr"
		}`,
	}))

	require.Equal(t, 0, resolveModelQuotaType("any-token-model", false))
	require.Equal(t, 1, resolveModelQuotaType("plain-price-model", true))
	require.Equal(t, 2, resolveModelQuotaType("seedance-per-second", true))
	// tiered_expr with ModelPrice still surfaces as per-request quota_type;
	// BillingMode/BillingExpr are attached separately in updatePricing.
	require.Equal(t, 1, resolveModelQuotaType("expr-model", true))
	require.Equal(t, billing_setting.BillingModePerSecond, billing_setting.GetBillingMode("seedance-per-second"))
}
