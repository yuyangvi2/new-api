package vclm

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
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
)

// cosCfg 自有 COS 转存配置（来自环境变量）。
type cosCfg struct {
	Bucket       string // 形如 xtoapi-media-1329970810
	Region       string // 形如 ap-beijing
	SecretID     string
	SecretKey    string
	PresignHours int64
}

// cosConfig 读取并校验 env；任一必填项为空则视为未启用(ok=false)。
func cosConfig() (cosCfg, bool) {
	c := cosCfg{
		Bucket:    strings.TrimSpace(os.Getenv("VCLM_COS_BUCKET")),
		Region:    strings.TrimSpace(os.Getenv("VCLM_COS_REGION")),
		SecretID:  strings.TrimSpace(os.Getenv("VCLM_COS_SECRET_ID")),
		SecretKey: strings.TrimSpace(os.Getenv("VCLM_COS_SECRET_KEY")),
	}
	if c.Region == "" {
		c.Region = "ap-beijing"
	}
	c.PresignHours = 168 // 7 天
	if v := strings.TrimSpace(os.Getenv("VCLM_COS_PRESIGN_HOURS")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			c.PresignHours = n
		}
	}
	if c.Bucket == "" || c.SecretID == "" || c.SecretKey == "" {
		return c, false
	}
	return c, true
}

func (c cosCfg) host() string {
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

// cosURLEncode 等价 python urllib.parse.quote(safe='')：仅保留 RFC3986 unreserved。
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

// cosSign 生成 COS 请求签名(sha1 体系)，返回完整 Authorization 串。
// headers 的 key 必须为小写。path 为对象路径(含前导 /)。
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

	// httpString: method\npath\nquery\nheaders\n （query 始终为空）
	httpString := strings.ToLower(method) + "\n" + objectPath + "\n\n" + canonHeaders + "\n"
	stringToSign := "sha1\n" + keyTime + "\n" + sha1Hex(httpString) + "\n"
	signature := hmacSHA1Hex(signKey, stringToSign)

	return fmt.Sprintf(
		"q-sign-algorithm=sha1&q-ak=%s&q-sign-time=%s&q-key-time=%s&q-header-list=%s&q-url-param-list=&q-signature=%s",
		secretID, keyTime, keyTime, headerList, signature)
}

// cosPut 上传对象到 COS。签名 header-list 仅含 host，其余头不参与签名但照常发送。
func cosPut(cfg cosCfg, objectKey string, body io.Reader, contentLength int64, contentType string) error {
	host := cfg.host()
	objectPath := "/" + objectKey
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

// cosPresignGet 生成对象的预签名 GET URL（有效期 PresignHours）。
func cosPresignGet(cfg cosCfg, objectKey string) string {
	host := cfg.host()
	objectPath := "/" + objectKey
	auth := cosSign(http.MethodGet, objectPath, map[string]string{"host": host}, cfg.SecretID, cfg.SecretKey, cfg.PresignHours*3600)
	return "https://" + host + objectPath + "?" + auth
}

// transferToCOS 下载上游视频并转存到自有 COS，返回预签名 GET URL。
func transferToCOS(cfg cosCfg, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	objectKey := "vclm/" + path.Base(u.Path)

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("download upstream status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}
	var body io.Reader = resp.Body
	contentLength := resp.ContentLength
	if contentLength <= 0 {
		// 上游未给 Content-Length：缓冲到内存以便确定长度
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		body = bytes.NewReader(buf)
		contentLength = int64(len(buf))
	}

	if err := cosPut(cfg, objectKey, body, contentLength, contentType); err != nil {
		return "", err
	}
	return cosPresignGet(cfg, objectKey), nil
}
