package volcengine

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
)

const (
	region      = "cn-beijing"
	serviceName = "ark"
	version     = "2024-01-01"
)

func callArk[Result any](ctx context.Context, action string, req any) (*Result, error) {
	baseURL, err := arkRequestBaseURL()
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if reader, ok := req.(io.Reader); ok {
		reqBody = reader
	} else {
		body, err := common.Marshal(req)
		if err != nil {
			return nil, errors.Wrap(err, "marshal ark request failed")
		}
		reqBody = bytes.NewReader(body)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "create ark request failed")
	}
	query := request.URL.Query()
	query.Set("Action", action)
	query.Set("Version", version)
	request.URL.RawQuery = query.Encode()

	if err = signRequest(request, arkSetting.AK, arkSetting.SK); err != nil {
		return nil, errors.Wrap(err, "sign ark request failed")
	}

	httpClient, err := service.GetHttpClientWithProxy(arkSetting.ProxyURL)
	if err != nil {
		return nil, errors.Wrap(err, "create ark http client failed")
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, "send ark request failed")
	}

	defer func() { _ = response.Body.Close() }()
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read ark response failed")
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf("ark returned http %d: %s", response.StatusCode, string(respBody))
	}

	resp := gjson.ParseBytes(respBody)
	if resp.Get("success").Type == gjson.False {
		return nil, newError(resp.Get("code").String(), resp.Get("message").String())
	}
	if errorPart := resp.Get("ResponseMetadata.Error"); errorPart.IsObject() {
		return nil, newError(errorPart.Get("Code").String(), errorPart.Get("Message").String())
	}
	rstPart := resp.Get("Result")
	if !rstPart.IsObject() {
		return nil, errors.Errorf("unknown ark result: %s", string(respBody))
	}
	var rst Result
	if err = common.Unmarshal([]byte(rstPart.Raw), &rst); err != nil {
		return nil, errors.Wrap(err, "unmarshal ark result failed")
	}
	return &rst, nil
}

func arkRequestBaseURL() (string, error) {
	baseURL := strings.TrimSpace(arkSetting.BaseURL)
	if baseURL == "" {
		return "", newError("InvalidConfiguration", "ark_setting.base_url is required.")
	}
	parsedURL, err := url.ParseRequestURI(baseURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return "", newError("InvalidConfiguration", "ark_setting.base_url must be an absolute http(s) URL.")
	}
	if strings.TrimSpace(arkSetting.AK) == "" || strings.TrimSpace(arkSetting.SK) == "" {
		return "", newError("InvalidConfiguration", "ark_setting.ak and ark_setting.sk are required.")
	}
	return baseURL, nil
}

func signRequest(req *http.Request, accessKey, secretKey string) error {
	var bodyBytes []byte
	var err error
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return errors.Wrap(err, "read request body failed")
		}
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	payloadHash := sha256.Sum256(bodyBytes)
	hexPayloadHash := hex.EncodeToString(payloadHash[:])
	now := time.Now().UTC()
	xDate := now.Format("20060102T150405Z")
	shortDate := now.Format("20060102")

	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Date", xDate)
	req.Header.Set("X-Content-Sha256", hexPayloadHash)
	req.Header.Set("Content-Type", "application/json")

	canonicalQueryString := canonicalQuery(req.URL.Query())
	headersToSign := map[string]string{
		"content-type":     "application/json",
		"host":             req.URL.Host,
		"x-content-sha256": hexPayloadHash,
		"x-date":           xDate,
	}
	headerKeys := make([]string, 0, len(headersToSign))
	for key := range headersToSign {
		headerKeys = append(headerKeys, key)
	}
	sort.Strings(headerKeys)

	var canonicalHeaders strings.Builder
	for _, key := range headerKeys {
		canonicalHeaders.WriteString(key)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(strings.TrimSpace(headersToSign[key]))
		canonicalHeaders.WriteString("\n")
	}
	signedHeaders := strings.Join(headerKeys, ";")
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		req.URL.Path,
		canonicalQueryString,
		canonicalHeaders.String(),
		signedHeaders,
		hexPayloadHash,
	)
	hashedCanonicalRequest := sha256.Sum256([]byte(canonicalRequest))
	credentialScope := fmt.Sprintf("%s/%s/%s/request", shortDate, region, serviceName)
	stringToSign := fmt.Sprintf("HMAC-SHA256\n%s\n%s\n%s", xDate, credentialScope, hex.EncodeToString(hashedCanonicalRequest[:]))

	kDate := hmacSHA256([]byte(secretKey), []byte(shortDate))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(serviceName))
	kSigning := hmacSHA256(kService, []byte("request"))
	signature := hex.EncodeToString(hmacSHA256(kSigning, []byte(stringToSign)))

	req.Header.Set("Authorization", fmt.Sprintf(
		"HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey,
		credentialScope,
		signedHeaders,
		signature,
	))
	return nil
}

func canonicalQuery(query url.Values) string {
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(query))
	for _, key := range keys {
		values := query[key]
		sort.Strings(values)
		for _, value := range values {
			parts = append(parts, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(value)))
		}
	}
	return strings.Join(parts, "&")
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(data)
	return h.Sum(nil)
}
