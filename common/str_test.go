package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSensitiveInfoMasksCredentialsAndEmail(t *testing.T) {
	input := "Authorization: Bearer token.payload.signature api_key:secret-value " +
		"Authorization: Basic dXNlcjpwYXNzd29yZA== Cookie: session=private-session; " +
		"sk-proj-abcdefghijklmnop AIzaSyABCDEFGHIJKLMNOPQRSTUV user@example.com"

	masked := MaskSensitiveInfo(input)

	assert.NotContains(t, masked, "token.payload.signature")
	assert.NotContains(t, masked, "secret-value")
	assert.NotContains(t, masked, "sk-proj-abcdefghijklmnop")
	assert.NotContains(t, masked, "AIzaSyABCDEFGHIJKLMNOPQRSTUV")
	assert.NotContains(t, masked, "user@example.com")
	assert.NotContains(t, masked, "dXNlcjpwYXNzd29yZA==")
	assert.NotContains(t, masked, "private-session")
	assert.Contains(t, masked, "Bearer ***")
}

func TestMaskSensitiveInfoMasksNamedSecretsAndIPURLs(t *testing.T) {
	input := `x-api-key=plain-secret access_token:"access-secret" password='password-secret' ` +
		`Authorization: custom-secret http://10.20.30.40:8080/private`

	masked := MaskSensitiveInfo(input)

	assert.NotContains(t, masked, "plain-secret")
	assert.NotContains(t, masked, "access-secret")
	assert.NotContains(t, masked, "password-secret")
	assert.NotContains(t, masked, "custom-secret")
	assert.NotContains(t, masked, "10.20.30.40")
	assert.NotContains(t, masked, ".40")
}

func TestMaskSensitiveInfoMasksAuthorizationSchemeCredential(t *testing.T) {
	input := "Authorization: Token upstream-account-secret request rejected"

	masked := MaskSensitiveInfo(input)

	assert.NotContains(t, masked, "upstream-account-secret")
	assert.Contains(t, masked, "Authorization: Token ***")
	assert.Contains(t, masked, "request rejected")
}

func TestMaskSensitiveInfoMasksCommonCloudAccessKeyPrefixes(t *testing.T) {
	input := "credentials AKIAABCDEFGHIJKLMNOP and ASIAABCDEFGHIJKLMNOP"

	masked := MaskSensitiveInfo(input)

	assert.NotContains(t, masked, "AKIAABCDEFGHIJKLMNOP")
	assert.NotContains(t, masked, "ASIAABCDEFGHIJKLMNOP")
}

func TestMaskSensitiveErrorTextPreservesDiagnosticsAndMasksSecrets(t *testing.T) {
	input := "Invalid reasoning.effort in tools.0.function.name for schema.json; " +
		"Authorization: Bearer token.payload.signature; key sk-proj-abcdefghijklmnop; " +
		"endpoint https://internal.example.com/v1; host 10.20.30.40; user user@example.com"

	masked := MaskSensitiveErrorText(input)

	assert.Contains(t, masked, "reasoning.effort")
	assert.Contains(t, masked, "tools.0.function.name")
	assert.Contains(t, masked, "schema.json")
	assert.NotContains(t, masked, "token.payload.signature")
	assert.NotContains(t, masked, "sk-proj-abcdefghijklmnop")
	assert.NotContains(t, masked, "internal.example.com")
	assert.NotContains(t, masked, "10.20.30.40")
	assert.NotContains(t, masked, "user@example.com")
}

func TestMaskSensitiveJSONRedactsSecretFieldsRecursively(t *testing.T) {
	input := []byte(`{
		"message":"Width must be between 300px and 6000px.",
		"code":"InvalidParameter",
		"metadata":{
			"api_key":"plain-secret",
			"nested":{"authorization":"custom-secret"}
		}
	}`)

	masked, err := MaskSensitiveJSON(input)

	assert.NoError(t, err)
	assert.Contains(t, string(masked), "Width must be between 300px and 6000px.")
	assert.Contains(t, string(masked), "InvalidParameter")
	assert.NotContains(t, string(masked), "plain-secret")
	assert.NotContains(t, string(masked), "custom-secret")
}

func TestMaskSensitiveJSONPreservesLargeIntegerPrecision(t *testing.T) {
	input := []byte(`{"request_id":18446744073709551615,"api_key":"plain-secret"}`)

	masked, err := MaskSensitiveJSON(input)

	assert.NoError(t, err)
	assert.Contains(t, string(masked), `18446744073709551615`)
	assert.NotContains(t, string(masked), "plain-secret")
}

func TestMaskSensitiveErrorJSONPreservesDottedDiagnostics(t *testing.T) {
	input := []byte(`{
		"message":"Invalid reasoning.effort in schema.json",
		"metadata":{
			"field":"tools.0.function.name",
			"api_key":"plain-secret",
			"token":"private-token",
			"endpoint":"https://internal.example.com/v1",
			"usage":{"token":42}
		}
	}`)

	masked, err := MaskSensitiveErrorJSON(input)

	assert.NoError(t, err)
	assert.Contains(t, string(masked), "reasoning.effort")
	assert.Contains(t, string(masked), "schema.json")
	assert.Contains(t, string(masked), "tools.0.function.name")
	assert.NotContains(t, string(masked), "plain-secret")
	assert.NotContains(t, string(masked), "private-token")
	assert.NotContains(t, string(masked), "internal.example.com")
	assert.Contains(t, string(masked), `"token":42`)
}
