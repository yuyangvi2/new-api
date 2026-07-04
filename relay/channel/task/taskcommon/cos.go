package taskcommon

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// COSConfig describes Tencent COS credentials used for temporary task assets.
type COSConfig struct {
	Bucket       string
	Region       string
	SecretID     string
	SecretKey    string
	PresignHours int64
}

// COSConfigFromEnv reads COS credentials from the first complete env prefix.
// For prefix "VIPEAK_COS", expected variables are VIPEAK_COS_BUCKET,
// VIPEAK_COS_REGION, VIPEAK_COS_SECRET_ID, VIPEAK_COS_SECRET_KEY and optional
// VIPEAK_COS_PRESIGN_HOURS.
func COSConfigFromEnv(prefixes ...string) (COSConfig, bool) {
	for _, prefix := range prefixes {
		prefix = strings.TrimRight(strings.TrimSpace(prefix), "_")
		if prefix == "" {
			continue
		}
		cfg := COSConfig{
			Bucket:    strings.TrimSpace(os.Getenv(prefix + "_BUCKET")),
			Region:    strings.TrimSpace(os.Getenv(prefix + "_REGION")),
			SecretID:  strings.TrimSpace(os.Getenv(prefix + "_SECRET_ID")),
			SecretKey: strings.TrimSpace(os.Getenv(prefix + "_SECRET_KEY")),
		}
		if cfg.Region == "" {
			cfg.Region = "ap-beijing"
		}
		cfg.PresignHours = 168
		if v := strings.TrimSpace(os.Getenv(prefix + "_PRESIGN_HOURS")); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
				cfg.PresignHours = n
			}
		}
		if cfg.Bucket != "" && cfg.SecretID != "" && cfg.SecretKey != "" {
			return cfg, true
		}
	}
	return COSConfig{}, false
}

func HasCOSConfig(prefixes ...string) bool {
	_, ok := COSConfigFromEnv(prefixes...)
	return ok
}

func (c COSConfig) host() string {
	return fmt.Sprintf("%s.cos.%s.myqcloud.com", c.Bucket, c.Region)
}

func hmacSHA1Hex(key, data string) string {
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func sha1Hex(data string) string {
	h := sha1.Sum([]byte(data))
	return hex.EncodeToString(h[:])
}

func cosURLEncode(s string) string {
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.~"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if strings.IndexByte(unreserved, ch) >= 0 {
			b.WriteByte(ch)
		} else {
			fmt.Fprintf(&b, "%%%02X", ch)
		}
	}
	return b.String()
}

func cosSign(method, objectPath string, headers map[string]string, secretID, secretKey string, expireSec int64) string {
	now := time.Now().Unix()
	keyTime := fmt.Sprintf("%d;%d", now, now+expireSec)
	signKey := hmacSHA1Hex(secretKey, keyTime)

	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, strings.ToLower(k))
	}
	sort.Strings(keys)
	headerList := strings.Join(keys, ";")
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+cosURLEncode(headers[k]))
	}
	canonHeaders := strings.Join(parts, "&")

	httpString := strings.ToLower(method) + "\n" + objectPath + "\n\n" + canonHeaders + "\n"
	stringToSign := "sha1\n" + keyTime + "\n" + sha1Hex(httpString) + "\n"
	signature := hmacSHA1Hex(signKey, stringToSign)

	return fmt.Sprintf(
		"q-sign-algorithm=sha1&q-ak=%s&q-sign-time=%s&q-key-time=%s&q-header-list=%s&q-url-param-list=&q-signature=%s",
		secretID, keyTime, keyTime, headerList, signature)
}

// COSPut uploads an object to Tencent COS.
func COSPut(cfg COSConfig, objectKey string, body io.Reader, contentLength int64, contentType string) error {
	host := cfg.host()
	objectPath := "/" + strings.TrimLeft(objectKey, "/")
	auth := cosSign(http.MethodPut, objectPath, map[string]string{"host": host}, cfg.SecretID, cfg.SecretKey, 600)

	req, err := http.NewRequest(http.MethodPut, "https://"+host+objectPath, body)
	if err != nil {
		return err
	}
	req.Host = host
	req.ContentLength = contentLength
	req.Header.Set("Authorization", auth)
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
		return fmt.Errorf("cos put status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

// COSPresignGet returns a signed GET URL for a COS object.
func COSPresignGet(cfg COSConfig, objectKey string) string {
	host := cfg.host()
	objectPath := "/" + strings.TrimLeft(objectKey, "/")
	auth := cosSign(http.MethodGet, objectPath, map[string]string{"host": host}, cfg.SecretID, cfg.SecretKey, cfg.PresignHours*3600)
	return "https://" + host + objectPath + "?" + auth
}

func IsHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// EnsurePublicImageURL returns s unchanged if it is already an http(s) URL.
// Otherwise it treats s as a base64 image, uploads it to COS, and returns a
// signed URL that upstream task providers can download.
func EnsurePublicImageURL(s string, objectPrefix string, prefixes ...string) (string, error) {
	if IsHTTPURL(s) {
		return s, nil
	}
	cfg, ok := COSConfigFromEnv(prefixes...)
	if !ok {
		return "", fmt.Errorf("COS not configured; uploaded/base64 images require a public URL before forwarding upstream")
	}
	raw, contentType := stripDataURLBase64(s)
	decoded, err := decodeBase64(raw)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %w", err)
	}
	if contentType == "" {
		contentType = http.DetectContentType(decoded)
		if !strings.HasPrefix(contentType, "image/") {
			contentType = "image/png"
		}
	}
	objectPrefix = strings.Trim(strings.TrimSpace(objectPrefix), "/")
	if objectPrefix == "" {
		objectPrefix = "task"
	}
	objectKey := fmt.Sprintf("%s/img-%d.%s", objectPrefix, time.Now().UnixNano(), imageExt(contentType))
	if err := COSPut(cfg, objectKey, bytes.NewReader(decoded), int64(len(decoded)), contentType); err != nil {
		return "", fmt.Errorf("COS upload failed: %w", err)
	}
	return COSPresignGet(cfg, objectKey), nil
}

func stripDataURLBase64(s string) (raw string, contentType string) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "data:") {
		return s, ""
	}
	idx := strings.Index(s, ",")
	if idx < 0 {
		return s, ""
	}
	meta := s[len("data:"):idx]
	if !strings.Contains(strings.ToLower(meta), "base64") {
		return s, ""
	}
	parts := strings.Split(meta, ";")
	if len(parts) > 0 && strings.HasPrefix(parts[0], "image/") {
		contentType = parts[0]
	}
	return s[idx+1:], contentType
}

func decodeBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if data, err := base64.StdEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	if data, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	if data, err := base64.URLEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	return base64.RawURLEncoding.DecodeString(s)
}

func imageExt(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		return "png"
	}
}
