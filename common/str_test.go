package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSensitiveInfoMasksCredentialsAndEmail(t *testing.T) {
	input := "Authorization: Bearer token.payload.signature api_key:secret-value " +
		"sk-proj-abcdefghijklmnop AIzaSyABCDEFGHIJKLMNOPQRSTUV user@example.com"

	masked := MaskSensitiveInfo(input)

	assert.NotContains(t, masked, "token.payload.signature")
	assert.NotContains(t, masked, "secret-value")
	assert.NotContains(t, masked, "sk-proj-abcdefghijklmnop")
	assert.NotContains(t, masked, "AIzaSyABCDEFGHIJKLMNOPQRSTUV")
	assert.NotContains(t, masked, "user@example.com")
	assert.Contains(t, masked, "Bearer ***")
}
