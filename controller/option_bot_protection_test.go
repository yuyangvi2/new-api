package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAndValidateCapOptionUpdateProtectsEnabledConfiguration(t *testing.T) {
	originalPublicEndpoint := common.CapPublicEndpoint
	originalVerifyEndpoint := common.CapVerifyEndpoint
	originalRegisterEnabled := common.CapRegisterCheckEnabled
	originalRegisterSiteKey := common.CapRegisterSiteKey
	originalRegisterSecretKey := common.CapRegisterSecretKey
	originalLoginEnabled := common.CapLoginCheckEnabled
	originalLoginSiteKey := common.CapLoginSiteKey
	originalLoginSecretKey := common.CapLoginSecretKey
	t.Cleanup(func() {
		common.CapPublicEndpoint = originalPublicEndpoint
		common.CapVerifyEndpoint = originalVerifyEndpoint
		common.CapRegisterCheckEnabled = originalRegisterEnabled
		common.CapRegisterSiteKey = originalRegisterSiteKey
		common.CapRegisterSecretKey = originalRegisterSecretKey
		common.CapLoginCheckEnabled = originalLoginEnabled
		common.CapLoginSiteKey = originalLoginSiteKey
		common.CapLoginSecretKey = originalLoginSecretKey
	})

	common.CapPublicEndpoint = "https://captcha.example.com/cap"
	common.CapVerifyEndpoint = "http://cap:3000"
	common.CapRegisterCheckEnabled = true
	common.CapRegisterSiteKey = "register-site"
	common.CapRegisterSecretKey = "register-secret"
	common.CapLoginCheckEnabled = false
	common.CapLoginSiteKey = ""
	common.CapLoginSecretKey = ""

	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "empty public endpoint", key: "CapPublicEndpoint", value: ""},
		{name: "endpoint with query", key: "CapPublicEndpoint", value: "https://captcha.example.com/cap?target=other"},
		{name: "unsupported endpoint scheme", key: "CapVerifyEndpoint", value: "file:///tmp/cap"},
		{name: "empty enabled site key", key: "CapRegisterSiteKey", value: "   "},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := normalizeAndValidateCapOptionUpdate(test.key, test.value)
			assert.Error(t, err)
		})
	}
}

func TestNormalizeAndValidateCapOptionUpdateKeepsExistingSecretOnBlankInput(t *testing.T) {
	originalSecret := common.CapLoginSecretKey
	t.Cleanup(func() { common.CapLoginSecretKey = originalSecret })
	common.CapLoginSecretKey = "existing-secret"

	value, err := normalizeAndValidateCapOptionUpdate("CapLoginSecretKey", "  ")

	require.NoError(t, err)
	assert.Equal(t, "existing-secret", value)
}

func TestNormalizeAndValidateCapOptionUpdateValidatesBeforeEnablingScene(t *testing.T) {
	originalPublicEndpoint := common.CapPublicEndpoint
	originalVerifyEndpoint := common.CapVerifyEndpoint
	originalEnabled := common.CapLoginCheckEnabled
	originalSiteKey := common.CapLoginSiteKey
	originalSecretKey := common.CapLoginSecretKey
	t.Cleanup(func() {
		common.CapPublicEndpoint = originalPublicEndpoint
		common.CapVerifyEndpoint = originalVerifyEndpoint
		common.CapLoginCheckEnabled = originalEnabled
		common.CapLoginSiteKey = originalSiteKey
		common.CapLoginSecretKey = originalSecretKey
	})

	common.CapPublicEndpoint = "https://captcha.example.com/cap"
	common.CapVerifyEndpoint = "http://cap:3000"
	common.CapLoginCheckEnabled = false
	common.CapLoginSiteKey = "login-site"
	common.CapLoginSecretKey = "login-secret"

	value, err := normalizeAndValidateCapOptionUpdate("CapLoginCheckEnabled", "true")

	require.NoError(t, err)
	assert.Equal(t, "true", value)
}

func TestCapSiteKeysRemainVisibleWhileSecretsStayHidden(t *testing.T) {
	assert.False(t, isSensitiveOptionKey("CapRegisterSiteKey"))
	assert.False(t, isSensitiveOptionKey("CapLoginSiteKey"))
	assert.True(t, isSensitiveOptionKey("CapRegisterSecretKey"))
	assert.True(t, isSensitiveOptionKey("CapLoginSecretKey"))
}
