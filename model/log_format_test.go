package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/require"
)

// TestFormatUserLogsStripsQuotaSaturation verifies the admin-only quota
// saturation marker (nested under other.admin_info) is removed for non-admin
// log views, since formatUserLogs strips the whole admin_info object.
func TestFormatUserLogsStripsQuotaSaturation(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"model_price": 0.004,
		"admin_info": map[string]interface{}{
			"quota_saturation": map[string]interface{}{
				"op":      "QuotaFromDecimal",
				"kind":    "overflow",
				"clamped": common.MaxQuota,
			},
		},
	})
	logs := []*Log{{Other: other}}

	formatUserLogs(logs, 0)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	_, hasAdminInfo := parsed["admin_info"]
	require.False(t, hasAdminInfo, "admin_info (and nested quota_saturation) must be stripped for non-admin views")
	// Non-admin billing fields remain visible.
	require.Contains(t, parsed, "model_price")
}

// TestFormatUserLogsStripsChannelIdentity verifies channel name/type/id are not
// visible in non-admin log views. Newer error logs nest these under admin_info;
// historical rows may still have them at the top level of other.
func TestFormatUserLogsStripsChannelIdentity(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"error_code":   "bad_response_status_code",
		"status_code":  502,
		"channel_id":   42,
		"channel_name": "openai-prod-secret-name",
		"channel_type": 1,
		"admin_info": map[string]interface{}{
			"channel_id":   42,
			"channel_name": "openai-prod-secret-name",
			"channel_type": 1,
			"use_channel":  []string{"42"},
		},
	})
	logs := []*Log{
		{
			ChannelId:   42,
			ChannelName: "openai-prod-secret-name",
			Other:       other,
		},
	}

	formatUserLogs(logs, 0)

	require.Empty(t, logs[0].ChannelName, "top-level channel_name must be cleared for non-admin views")

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	_, hasAdminInfo := parsed["admin_info"]
	require.False(t, hasAdminInfo, "admin_info (including nested channel identity) must be stripped")
	for _, key := range []string{"channel_id", "channel_name", "channel_type"} {
		_, present := parsed[key]
		require.Falsef(t, present, "other.%s must not be visible to non-admin viewers", key)
	}
	// Non-admin error diagnostics remain visible.
	require.Equal(t, "bad_response_status_code", parsed["error_code"])
	require.EqualValues(t, 502, parsed["status_code"])
}
