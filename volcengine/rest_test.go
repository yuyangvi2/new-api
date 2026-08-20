package volcengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
)

func TestPatchRESTActionPayloadInjectsPathID(t *testing.T) {
	body, err := patchRESTActionPayload([]byte(`{"Name":"updated"}`), "Id", "asset-123", nil)

	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	assert.Equal(t, "asset-123", payload["Id"])
	assert.Equal(t, "updated", payload["Name"])
}

func TestPatchRESTActionPayloadAcceptsBytedTokenAlias(t *testing.T) {
	body, err := patchRESTActionPayload([]byte(`{"byted_token":"token-123"}`), "", "", map[string]string{
		"byted_token": "BytedToken",
		"bytedToken":  "BytedToken",
	})

	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	assert.Equal(t, "token-123", payload["BytedToken"])
}
