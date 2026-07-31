package seedancem

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/relay/channel/task/seedancem/securechannel"
	"github.com/QuantumNous/new-api/service"
)

const aiccSDKVersion = "0.1.0"

type aiccHTTPClient struct {
	client *http.Client
	sc     *securechannel.Client
}

func newAICCHTTPClient(base, apiKey, _ string) (*aiccHTTPClient, error) {
	// The official AICC SDK uses a direct client for attestation and encrypted
	// video task calls; some generic channel proxies reject the RA handshake.
	httpClient, err := service.GetHttpClientWithProxy("")
	if err != nil {
		return nil, fmt.Errorf("new http client failed: %w", err)
	}

	trueValue := true
	sc := securechannel.NewClient(securechannel.ClientConfig{
		RaURL:            seedanceRAURL(base),
		RaType:           securechannel.RA_TYPE_LOCAL,
		RaServiceName:    "AICC.ConfidentialInference",
		RaPolicyId:       "router_policy",
		RaAuthToken:      strings.TrimSpace(apiKey),
		RaKeyNegotiation: &trueValue,
		RaNeedToken:      &trueValue,
		HTTPClient:       httpClient,
	})
	if err := sc.AttestServer(); err != nil {
		return nil, fmt.Errorf("attest upstream secure channel failed: %w", err)
	}

	return &aiccHTTPClient{
		client: httpClient,
		sc:     sc,
	}, nil
}

func (c *aiccHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var requestBody []byte
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		requestBody = body
		_ = req.Body.Close()
	}

	encrypted, err := c.sc.EncryptBytesWithResponse(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encrypt request body: %w", err)
	}

	req.Header.Set("X-AICC-Encryption-Enable", "true")
	req.Header.Set("X-AICC-Encryption-SDK", "aicc")
	req.Header.Set("X-AICC-Encryption-Version", aiccSDKVersion)
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(encrypted.Ciphertext))
	req.ContentLength = int64(len(encrypted.Ciphertext))
	req.Header.Del("Content-Length")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("read encrypted response body: %w", err)
	}
	_ = resp.Body.Close()

	decryptedContent, err := encrypted.ResponseKey.DecryptBytesString(string(content))
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(content))
		return resp, nil
	}
	resp.Body = io.NopCloser(bytes.NewReader(decryptedContent))
	resp.ContentLength = int64(len(decryptedContent))
	resp.Header.Del("Content-Length")
	return resp, nil
}

func seedanceRAURL(base string) string {
	parsed, err := url.Parse(baseURL(base))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(baseURL(base), "/") + "/v1/security/token"
	}
	return parsed.Scheme + "://" + parsed.Host + "/v1/security/token"
}
