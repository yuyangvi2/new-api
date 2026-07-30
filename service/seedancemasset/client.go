package seedancemasset

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"
)

type Client struct {
	Endpoint         string
	PoolID           string
	AccessKey        string
	SecretKey        string
	SignatureMethod  string
	HTTPClient       *http.Client
	MaxResponseBytes int64
}

func New(settings *dto.SeedanceMSettings, proxy string) (*Client, error) {
	if settings == nil {
		return nil, fmt.Errorf("seedance_m settings are required")
	}
	accessKey := strings.TrimSpace(settings.AssetAccessKey)
	secretKey := strings.TrimSpace(settings.AssetSecretKey)
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("seedance_m asset_access_key and asset_secret_key are required")
	}
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(strings.TrimSpace(settings.AssetEndpoint), "/")
	if endpoint == "" {
		endpoint = "https://ecloud.10086.cn"
	}
	poolID := strings.TrimSpace(settings.AssetPoolID)
	if poolID == "" {
		poolID = "CIDC-CORE-00"
	}
	method := strings.TrimSpace(settings.AssetSignatureMethod)
	if method == "" {
		method = "HmacSHA1"
	}
	if method != "HmacSHA1" && method != "HmacSHA256" {
		return nil, fmt.Errorf("unsupported asset_signature_method: %s", method)
	}
	return &Client{
		Endpoint:         endpoint,
		PoolID:           poolID,
		AccessKey:        accessKey,
		SecretKey:        secretKey,
		SignatureMethod:  method,
		HTTPClient:       client,
		MaxResponseBytes: 8 << 20,
	}, nil
}

func (c *Client) Call(method string, path string, query url.Values, payload any) (int, http.Header, []byte, error) {
	var body io.Reader
	if payload != nil {
		data, err := common.Marshal(payload)
		if err != nil {
			return 0, nil, nil, err
		}
		body = bytes.NewReader(data)
	}
	if query == nil {
		query = url.Values{}
	}
	// Mobile Cloud OpenAPI SDK formats China local time with a literal Z suffix.
	now := time.Now()
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		now = now.In(loc)
	} else {
		now = now.In(time.FixedZone("CST", 8*60*60))
	}
	signedQuery, err := c.signedQuery(method, path, query, now, randomNonce)
	if err != nil {
		return 0, nil, nil, err
	}
	req, err := http.NewRequest(method, c.Endpoint+path+"?"+signedQuery.Encode(), body)
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("pool-id", c.PoolID)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	reader := io.Reader(resp.Body)
	if c.MaxResponseBytes > 0 {
		reader = io.LimitReader(resp.Body, c.MaxResponseBytes+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, nil, nil, err
	}
	if c.MaxResponseBytes > 0 && int64(len(data)) > c.MaxResponseBytes {
		return 0, nil, nil, fmt.Errorf("upstream response is too large")
	}
	return resp.StatusCode, resp.Header.Clone(), data, nil
}

func (c *Client) signedQuery(method string, path string, query url.Values, now time.Time, nonceFn func() string) (url.Values, error) {
	values := url.Values{}
	for key, list := range query {
		for _, item := range list {
			values.Add(key, item)
		}
	}
	values.Set("AccessKey", c.AccessKey)
	values.Set("Timestamp", now.Format("2006-01-02T15:04:05Z"))
	values.Set("SignatureMethod", c.SignatureMethod)
	values.Set("SignatureVersion", "V2.0")
	values.Set("SignatureNonce", nonceFn())
	if values.Get("Version") == "" {
		values.Set("Version", "2016-12-05")
	}
	canonical := canonicalQuery(values)
	digest := sha256.Sum256([]byte(canonical))
	stringToSign := strings.ToUpper(method) + "\n" + percentEncode(path) + "\n" + hex.EncodeToString(digest[:])
	signature, err := c.sign(stringToSign)
	if err != nil {
		return nil, err
	}
	values.Set("Signature", signature)
	return values, nil
}

func (c *Client) sign(data string) (string, error) {
	var h func() hash.Hash
	switch c.SignatureMethod {
	case "HmacSHA1":
		h = sha1.New
	case "HmacSHA256":
		h = sha256.New
	default:
		return "", fmt.Errorf("unsupported signature method: %s", c.SignatureMethod)
	}
	mac := hmac.New(h, []byte("BC_SIGNATURE&"+c.SecretKey))
	_, _ = mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func canonicalQuery(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key == "Signature" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		list := append([]string(nil), values[key]...)
		sort.Strings(list)
		for _, value := range list {
			parts = append(parts, percentEncode(key)+"="+percentEncode(value))
		}
	}
	return strings.Join(parts, "&")
}

func percentEncode(value string) string {
	escaped := url.QueryEscape(value)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "*", "%2A")
	escaped = strings.ReplaceAll(escaped, "%7E", "~")
	return escaped
}

func randomNonce() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
