package seedancemasset

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignedQueryMatchesOpenAPISignatureExample(t *testing.T) {
	client := &Client{
		AccessKey:       "testid",
		SecretKey:       "testsecret",
		SignatureMethod: "HmacSHA1",
	}
	now, err := time.Parse("2006-01-02T15:04:05Z", "2017-01-11T15:15:11Z")
	require.NoError(t, err)

	values, err := client.signedQuery(httpMethodGet, "/api/keypair", url.Values{}, now, func() string {
		return "9d81ffbeaaf7477390db5df577bb3299"
	})
	require.NoError(t, err)

	assert.Equal(t, "2976158792407a581305786b2a82c56d9b007362", values.Get("Signature"))
}

const httpMethodGet = "GET"
