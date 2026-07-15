package model_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsGeminiModelSupportImagineRecognizesImageModelNames(t *testing.T) {
	t.Parallel()

	assert.True(t, IsGeminiModelSupportImagine("gemini-3.1-flash-image"))
	assert.True(t, IsGeminiModelSupportImagine("gemini-3-pro-image-preview"))
	assert.True(t, IsGeminiModelSupportImagine("models/gemini-2.0-flash-exp-image-generation"))
	assert.True(t, IsGeminiModelSupportImagine("nano-banana-pro-preview"))
	assert.False(t, IsGeminiModelSupportImagine("gemini-3-pro-preview"))
}
