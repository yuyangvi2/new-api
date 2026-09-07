package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStatusExposesOnlyPublicBotProtectionConfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalPublicEndpoint := common.CapPublicEndpoint
	originalRegisterEnabled := common.CapRegisterCheckEnabled
	originalRegisterSiteKey := common.CapRegisterSiteKey
	originalRegisterSecretKey := common.CapRegisterSecretKey
	originalLoginEnabled := common.CapLoginCheckEnabled
	originalLoginSiteKey := common.CapLoginSiteKey
	originalLoginSecretKey := common.CapLoginSecretKey
	t.Cleanup(func() {
		common.CapPublicEndpoint = originalPublicEndpoint
		common.CapRegisterCheckEnabled = originalRegisterEnabled
		common.CapRegisterSiteKey = originalRegisterSiteKey
		common.CapRegisterSecretKey = originalRegisterSecretKey
		common.CapLoginCheckEnabled = originalLoginEnabled
		common.CapLoginSiteKey = originalLoginSiteKey
		common.CapLoginSecretKey = originalLoginSecretKey
	})
	common.CapPublicEndpoint = "https://captcha.example.com/cap"
	common.CapRegisterCheckEnabled = true
	common.CapRegisterSiteKey = "register-site"
	common.CapRegisterSecretKey = "register-secret-must-not-leak"
	common.CapLoginCheckEnabled = true
	common.CapLoginSiteKey = "login-site"
	common.CapLoginSecretKey = "login-secret-must-not-leak"

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	GetStatus(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), common.CapRegisterSecretKey)
	assert.NotContains(t, recorder.Body.String(), common.CapLoginSecretKey)

	var response struct {
		Data struct {
			BotProtection struct {
				Register struct {
					Enabled        bool   `json:"enabled"`
					Provider       string `json:"provider"`
					PublicEndpoint string `json:"public_endpoint"`
					SiteKey        string `json:"site_key"`
				} `json:"register"`
				Login struct {
					Enabled  bool   `json:"enabled"`
					Provider string `json:"provider"`
					SiteKey  string `json:"site_key"`
				} `json:"login"`
			} `json:"bot_protection"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Data.BotProtection.Register.Enabled)
	assert.Equal(t, "cap", response.Data.BotProtection.Register.Provider)
	assert.Equal(t, "https://captcha.example.com/cap", response.Data.BotProtection.Register.PublicEndpoint)
	assert.Equal(t, "register-site", response.Data.BotProtection.Register.SiteKey)
	assert.True(t, response.Data.BotProtection.Login.Enabled)
	assert.Equal(t, "cap", response.Data.BotProtection.Login.Provider)
	assert.Equal(t, "login-site", response.Data.BotProtection.Login.SiteKey)
}
