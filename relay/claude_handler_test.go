package relay

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldUseDefaultClaudeMaxTokens(t *testing.T) {
	t.Run("missing max_tokens uses default", func(t *testing.T) {
		assert.True(t, shouldUseDefaultClaudeMaxTokens(nil))
	})

	t.Run("explicit zero max_tokens is preserved for cache prewarming", func(t *testing.T) {
		zero := uint(0)
		assert.False(t, shouldUseDefaultClaudeMaxTokens(&zero))
	})
}
