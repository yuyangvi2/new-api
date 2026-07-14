package controller

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOfficialAlipayPaymentPutsCharsetInGatewayQuery(t *testing.T) {
	originalAppID := setting.OfficialAlipayAppID
	originalPrivateKey := setting.OfficialAlipayPrivateKey
	originalGateway := setting.OfficialAlipayGateway
	defer func() {
		setting.OfficialAlipayAppID = originalAppID
		setting.OfficialAlipayPrivateKey = originalPrivateKey
		setting.OfficialAlipayGateway = originalGateway
	}()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	setting.OfficialAlipayAppID = "app_123"
	setting.OfficialAlipayPrivateKey = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}))
	setting.OfficialAlipayGateway = "https://openapi.alipay.com/gateway.do?foo=bar&charset=gbk"

	gateway, params, err := buildOfficialAlipayPayment(
		officialPaymentOrder{
			tradeNo: "USR1NOabc123",
			name:    "TUC100",
			money:   10,
		},
		"https://cn.tokone.ai",
		"https://cn.tokone.ai/wallet",
	)

	require.NoError(t, err)
	parsedGateway, err := url.Parse(gateway)
	require.NoError(t, err)
	assert.Equal(t, "utf-8", parsedGateway.Query().Get("charset"))
	assert.Equal(t, "bar", parsedGateway.Query().Get("foo"))
	assert.NotContains(t, params, "charset")
	assert.Equal(t, "https://cn.tokone.ai/api/official/alipay/notify", params["notify_url"])
	assert.Equal(t, "https://cn.tokone.ai/wallet", params["return_url"])
	assert.NotEmpty(t, params["sign"])
}

func TestOfficialPaymentAddressUsesForwardedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalCallbackAddress := operation_setting.CustomCallbackAddress
	defer func() {
		operation_setting.CustomCallbackAddress = originalCallbackAddress
	}()
	operation_setting.CustomCallbackAddress = ""

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "http://newapi:3000/api/user/pay", nil)
	c.Request.Header.Set("X-Forwarded-Proto", "https")
	c.Request.Header.Set("X-Forwarded-Host", "cn.tokone.ai")

	assert.Equal(t, "https://cn.tokone.ai", officialPaymentCallbackAddress(c))
	assert.Equal(t, "https://cn.tokone.ai"+common.ThemeAwarePath("/console/topup?show_history=true"), officialPaymentReturnPath(c, "/console/topup?show_history=true"))
}

func TestOfficialPaymentCallbackAddressPrefersExplicitSetting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalCallbackAddress := operation_setting.CustomCallbackAddress
	defer func() {
		operation_setting.CustomCallbackAddress = originalCallbackAddress
	}()
	operation_setting.CustomCallbackAddress = "https://callback.example.com/"

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "http://newapi:3000/api/user/pay", nil)
	c.Request.Header.Set("X-Forwarded-Proto", "https")
	c.Request.Header.Set("X-Forwarded-Host", "cn.tokone.ai")

	assert.Equal(t, "https://callback.example.com", officialPaymentCallbackAddress(c))
}

func TestAlipaySignContentIncludesSignType(t *testing.T) {
	content := alipaySignContent(map[string]string{
		"app_id":    "2021006158665022",
		"method":    "alipay.trade.page.pay",
		"charset":   "utf-8",
		"sign_type": "RSA2",
		"sign":      "ignored",
	}, true)

	assert.Equal(t, "app_id=2021006158665022&charset=utf-8&method=alipay.trade.page.pay&sign_type=RSA2", content)
}

func TestAlipayNotifySignContentExcludesSignType(t *testing.T) {
	content := alipaySignContent(map[string]string{
		"app_id":    "2021006158665022",
		"charset":   "utf-8",
		"sign_type": "RSA2",
		"sign":      "ignored",
	}, false)

	assert.Equal(t, "app_id=2021006158665022&charset=utf-8", content)
}
