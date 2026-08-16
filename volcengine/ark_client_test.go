package volcengine

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalQueryEscapesSpacesAsPercent20(t *testing.T) {
	query := url.Values{
		"q":    []string{"hello world"},
		"plus": []string{"a+b"},
	}

	assert.Equal(t, "plus=a%2Bb&q=hello%20world", canonicalQuery(query))
}
