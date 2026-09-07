package middleware

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/gin-gonic/gin"
)

type BotProtectionScene string

const (
	BotProtectionSceneRegister BotProtectionScene = "register"
	BotProtectionSceneLogin    BotProtectionScene = "login"
	CaptchaTokenHeader                            = "X-Captcha-Token"

	turnstileVerifyEndpoint     = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	maxVerificationResponseSize = 64 * 1024
)

var botProtectionHTTPClient = &http.Client{
	Timeout: 5 * time.Second,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

type botProtectionConfig struct {
	provider       string
	verifyEndpoint string
	siteKey        string
	secretKey      string
}

type botProtectionResponse struct {
	Success bool `json:"success"`
}

func BotProtectionCheck(scene BotProtectionScene) gin.HandlerFunc {
	return func(c *gin.Context) {
		config, enabled := botProtectionConfigForScene(scene)
		if !enabled {
			c.Next()
			return
		}

		token := strings.TrimSpace(c.GetHeader(CaptchaTokenHeader))
		if token == "" && config.provider == "turnstile" {
			// Keep legacy clients working while new clients move CAPTCHA tokens out
			// of URLs and into X-Captcha-Token.
			token = strings.TrimSpace(c.Query("turnstile"))
		}
		if token == "" {
			abortBotProtection(c, http.StatusBadRequest, "Please complete human verification.")
			return
		}

		valid, err := verifyBotProtection(c, config, token)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("%s bot protection verification failed: %v", config.provider, err))
			abortBotProtection(c, http.StatusServiceUnavailable, "Human verification service is temporarily unavailable. Please try again later.")
			return
		}
		if !valid {
			abortBotProtection(c, http.StatusBadRequest, "Human verification failed. Please try again.")
			return
		}

		c.Next()
	}
}

// TurnstileCheck is retained for non-authentication features that still use
// the legacy global Turnstile setting. Verification is deliberately performed
// for every request; successful checks are never persisted in the session.
func TurnstileCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.TurnstileCheckEnabled {
			c.Next()
			return
		}

		token := strings.TrimSpace(c.GetHeader(CaptchaTokenHeader))
		if token == "" {
			token = strings.TrimSpace(c.Query("turnstile"))
		}
		if token == "" {
			abortBotProtection(c, http.StatusBadRequest, "Please complete human verification.")
			return
		}

		valid, err := verifyBotProtection(c, botProtectionConfig{
			provider:  "turnstile",
			secretKey: common.TurnstileSecretKey,
		}, token)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("turnstile verification failed: %v", err))
			abortBotProtection(c, http.StatusServiceUnavailable, "Human verification service is temporarily unavailable. Please try again later.")
			return
		}
		if !valid {
			abortBotProtection(c, http.StatusBadRequest, "Human verification failed. Please try again.")
			return
		}

		c.Next()
	}
}

func botProtectionConfigForScene(scene BotProtectionScene) (botProtectionConfig, bool) {
	switch scene {
	case BotProtectionSceneRegister:
		if common.CapRegisterCheckEnabled {
			return botProtectionConfig{
				provider:       "cap",
				verifyEndpoint: common.CapVerifyEndpoint,
				siteKey:        common.CapRegisterSiteKey,
				secretKey:      common.CapRegisterSecretKey,
			}, true
		}
	case BotProtectionSceneLogin:
		if common.CapLoginCheckEnabled {
			return botProtectionConfig{
				provider:       "cap",
				verifyEndpoint: common.CapVerifyEndpoint,
				siteKey:        common.CapLoginSiteKey,
				secretKey:      common.CapLoginSecretKey,
			}, true
		}
	}

	if common.TurnstileCheckEnabled {
		return botProtectionConfig{
			provider:  "turnstile",
			siteKey:   common.TurnstileSiteKey,
			secretKey: common.TurnstileSecretKey,
		}, true
	}
	return botProtectionConfig{}, false
}

func verifyBotProtection(c *gin.Context, config botProtectionConfig, token string) (bool, error) {
	var request *http.Request
	var err error

	switch config.provider {
	case "cap":
		if config.verifyEndpoint == "" || config.siteKey == "" || config.secretKey == "" {
			return false, errors.New("Cap verification is enabled but its server configuration is incomplete")
		}
		payload, marshalErr := common.Marshal(map[string]string{
			"secret":   config.secretKey,
			"response": token,
		})
		if marshalErr != nil {
			return false, marshalErr
		}
		endpoint := strings.TrimRight(config.verifyEndpoint, "/") + "/" + url.PathEscape(config.siteKey) + "/siteverify"
		request, err = http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, bytes.NewReader(payload))
		if err == nil {
			request.Header.Set("Content-Type", "application/json")
		}
	case "turnstile":
		if config.secretKey == "" {
			return false, errors.New("Turnstile verification is enabled but its secret key is empty")
		}
		form := url.Values{
			"secret":   {config.secretKey},
			"response": {token},
			"remoteip": {c.ClientIP()},
		}
		request, err = http.NewRequestWithContext(c.Request.Context(), http.MethodPost, turnstileVerifyEndpoint, strings.NewReader(form.Encode()))
		if err == nil {
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	default:
		return false, fmt.Errorf("unsupported bot protection provider %q", config.provider)
	}
	if err != nil {
		return false, err
	}

	response, err := botProtectionHTTPClient.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return false, fmt.Errorf("verification endpoint returned HTTP %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxVerificationResponseSize+1))
	if err != nil {
		return false, err
	}
	if len(body) > maxVerificationResponseSize {
		return false, errors.New("verification response exceeds size limit")
	}
	var result botProtectionResponse
	if err = common.Unmarshal(body, &result); err != nil {
		return false, err
	}
	return result.Success, nil
}

func abortBotProtection(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"success": false,
		"message": message,
	})
}
