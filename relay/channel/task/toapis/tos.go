package toapis

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/QuantumNous/new-api/common"
)

const (
	tosAlgorithm       = "TOS4-HMAC-SHA256"
	tosService         = "tos"
	tosRequest         = "request"
	tosUnsignedPayload = "UNSIGNED-PAYLOAD"
)

type tosConfig struct {
	Bucket        string
	Region        string
	Endpoint      string
	AccessKey     string
	SecretKey     string
	SecurityToken string
	Prefix        string
	PresignHours  int64
}

func toapisTOSConfig() (tosConfig, bool) {
	cfg := tosConfig{
		Bucket:        strings.TrimSpace(os.Getenv("TOAPIS_TOS_BUCKET")),
		Region:        strings.TrimSpace(os.Getenv("TOAPIS_TOS_REGION")),
		Endpoint:      strings.TrimSpace(os.Getenv("TOAPIS_TOS_ENDPOINT")),
		AccessKey:     strings.TrimSpace(os.Getenv("TOAPIS_TOS_ACCESS_KEY")),
		SecretKey:     strings.TrimSpace(os.Getenv("TOAPIS_TOS_SECRET_KEY")),
		SecurityToken: strings.TrimSpace(os.Getenv("TOAPIS_TOS_SECURITY_TOKEN")),
		Prefix:        strings.Trim(strings.TrimSpace(os.Getenv("TOAPIS_TOS_PREFIX")), "/"),
		PresignHours:  24,
	}
	if cfg.Endpoint == "" && cfg.Region != "" {
		cfg.Endpoint = "tos-" + cfg.Region + ".volces.com"
	}
	if cfg.Prefix == "" {
		cfg.Prefix = "toapis"
	}
	if v := strings.TrimSpace(os.Getenv("TOAPIS_TOS_PRESIGN_HOURS")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			cfg.PresignHours = n
		}
	}
	if cfg.Bucket == "" || cfg.Region == "" || cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return cfg, false
	}
	return cfg, true
}

func (c tosConfig) host() string {
	endpoint := strings.TrimRight(strings.TrimSpace(c.Endpoint), "/")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	return c.Bucket + "." + endpoint
}

func transferToTOS(cfg tosConfig, rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	extension := path.Ext(sanitizeObjectFilename(path.Base(parsed.Path)))
	if extension == "" {
		extension = ".mp4"
	}
	filename := uuid.NewString() + extension
	objectKey := cfg.Prefix + "/" + filename

	tmp, err := os.CreateTemp("", "toapis-video-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		tmp.Close()
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		tmp.Close()
		return "", fmt.Errorf("download upstream status %d", resp.StatusCode)
	}

	hash := sha256.New()
	written, err := io.Copy(tmp, io.TeeReader(resp.Body, hash))
	if err != nil {
		tmp.Close()
		return "", err
	}
	if _, err = tmp.Seek(0, io.SeekStart); err != nil {
		tmp.Close()
		return "", err
	}
	defer tmp.Close()

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "video/mp4"
	}
	payloadHash := hex.EncodeToString(hash.Sum(nil))
	if err = tosPut(cfg, objectKey, tmp, written, contentType, payloadHash, time.Now().UTC()); err != nil {
		return "", err
	}
	return tosPresignGet(cfg, objectKey, time.Now().UTC()), nil
}

func tosPut(cfg tosConfig, objectKey string, body io.Reader, contentLength int64, contentType string, payloadHash string, now time.Time) error {
	host := cfg.host()
	objectPath := "/" + tosPathEscape(strings.TrimLeft(objectKey, "/"))
	xDate := now.Format("20060102T150405Z")
	shortDate := now.Format("20060102")
	headers := map[string]string{
		"host":                 host,
		"x-tos-content-sha256": payloadHash,
		"x-tos-date":           xDate,
	}
	if cfg.SecurityToken != "" {
		headers["x-tos-security-token"] = cfg.SecurityToken
	}
	auth := tosAuthorization(cfg, http.MethodPut, objectPath, "", headers, payloadHash, xDate, shortDate)

	req, err := http.NewRequest(http.MethodPut, "https://"+host+objectPath, body)
	if err != nil {
		return err
	}
	req.Host = host
	req.ContentLength = contentLength
	req.Header.Set("Authorization", auth)
	req.Header.Set("x-tos-content-sha256", payloadHash)
	req.Header.Set("x-tos-date", xDate)
	if cfg.SecurityToken != "" {
		req.Header.Set("x-tos-security-token", cfg.SecurityToken)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("tos put status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

func tosPresignGet(cfg tosConfig, objectKey string, now time.Time) string {
	host := cfg.host()
	objectPath := "/" + tosPathEscape(strings.TrimLeft(objectKey, "/"))
	xDate := now.Format("20060102T150405Z")
	shortDate := now.Format("20060102")
	expires := cfg.PresignHours * 3600
	if expires <= 0 {
		expires = 24 * 3600
	}
	if expires > 30*24*3600 {
		expires = 30 * 24 * 3600
	}

	query := map[string]string{
		"X-Tos-Algorithm":     tosAlgorithm,
		"X-Tos-Credential":    cfg.AccessKey + "/" + shortDate + "/" + cfg.Region + "/" + tosService + "/" + tosRequest,
		"X-Tos-Date":          xDate,
		"X-Tos-Expires":       strconv.FormatInt(expires, 10),
		"X-Tos-SignedHeaders": "host",
	}
	if cfg.SecurityToken != "" {
		query["X-Tos-Security-Token"] = cfg.SecurityToken
	}
	canonicalQuery := tosCanonicalQuery(query)
	headers := map[string]string{"host": host}
	signature := tosSignature(cfg, http.MethodGet, objectPath, canonicalQuery, headers, tosUnsignedPayload, xDate, shortDate)
	query["X-Tos-Signature"] = signature
	return "https://" + host + objectPath + "?" + tosCanonicalQuery(query)
}

func tosAuthorization(cfg tosConfig, method, objectPath, canonicalQuery string, headers map[string]string, payloadHash, xDate, shortDate string) string {
	signedHeaders := tosSignedHeaders(headers)
	signature := tosSignature(cfg, method, objectPath, canonicalQuery, headers, payloadHash, xDate, shortDate)
	return fmt.Sprintf(
		"%s Credential=%s/%s/%s/%s/%s, SignedHeaders=%s, Signature=%s",
		tosAlgorithm,
		cfg.AccessKey,
		shortDate,
		cfg.Region,
		tosService,
		tosRequest,
		signedHeaders,
		signature,
	)
}

func tosSignature(cfg tosConfig, method, objectPath, canonicalQuery string, headers map[string]string, payloadHash, xDate, shortDate string) string {
	canonicalRequest := strings.Join([]string{
		method,
		objectPath,
		canonicalQuery,
		tosCanonicalHeaders(headers),
		tosSignedHeaders(headers),
		payloadHash,
	}, "\n")
	scope := strings.Join([]string{shortDate, cfg.Region, tosService, tosRequest}, "/")
	requestHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := strings.Join([]string{
		tosAlgorithm,
		xDate,
		scope,
		hex.EncodeToString(requestHash[:]),
	}, "\n")
	key := tosSigningKey(cfg.SecretKey, shortDate, cfg.Region)
	return hex.EncodeToString(tosHMACSHA256(key, []byte(stringToSign)))
}

func tosSigningKey(secretKey, shortDate, region string) []byte {
	dateKey := tosHMACSHA256([]byte(secretKey), []byte(shortDate))
	regionKey := tosHMACSHA256(dateKey, []byte(region))
	serviceKey := tosHMACSHA256(regionKey, []byte(tosService))
	return tosHMACSHA256(serviceKey, []byte(tosRequest))
}

func tosHMACSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func tosCanonicalHeaders(headers map[string]string) string {
	keys := tosHeaderKeys(headers)
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		b.WriteByte(':')
		b.WriteString(strings.TrimSpace(headers[key]))
		b.WriteByte('\n')
	}
	return b.String()
}

func tosSignedHeaders(headers map[string]string) string {
	return strings.Join(tosHeaderKeys(headers), ";")
}

func tosHeaderKeys(headers map[string]string) []string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, strings.ToLower(key))
	}
	sort.Strings(keys)
	return keys
}

func tosCanonicalQuery(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, tosQueryEscape(key)+"="+tosQueryEscape(values[key]))
	}
	return strings.Join(parts, "&")
}

func tosQueryEscape(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func tosPathEscape(s string) string {
	parts := strings.Split(s, "/")
	for i, part := range parts {
		parts[i] = tosQueryEscape(part)
	}
	return strings.Join(parts, "/")
}

func sanitizeObjectFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" || filename == "." || filename == "/" {
		return ""
	}
	filename = strings.Trim(filename, "/")
	if filename == "" || filename == "." {
		return ""
	}
	return filename
}

func maybeTransferToTOS(rawURL string) string {
	if strings.TrimSpace(rawURL) == "" {
		return rawURL
	}
	cfg, ok := toapisTOSConfig()
	if !ok {
		return rawURL
	}
	newURL, err := transferToTOS(cfg, rawURL)
	if err != nil {
		common.SysError("toapis TOS transfer failed: " + err.Error())
		return rawURL
	}
	return newURL
}
