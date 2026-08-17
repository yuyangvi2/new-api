package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func invoiceConfigResponseForTest(t *testing.T) struct {
	Success bool `json:"success"`
	Data    struct {
		PayMethods []map[string]string `json:"pay_methods"`
	} `json:"data"`
} {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/invoice/config", nil)
	GetInvoiceConfig(ctx)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			PayMethods []map[string]string `json:"pay_methods"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func invoicePaymentMethodTypes(methods []map[string]string) []string {
	types := make([]string, 0, len(methods))
	for _, method := range methods {
		types = append(types, method["type"])
	}
	return types
}

func TestGetInvoiceConfigFiltersExternalPaymentMethodsByComplianceAndAvailability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	paymentSetting := operation_setting.GetPaymentSetting()
	originalConfirmed := paymentSetting.ComplianceConfirmed
	originalTerms := paymentSetting.ComplianceTermsVersion
	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalMethods := operation_setting.PayMethods
	t.Cleanup(func() {
		paymentSetting.ComplianceConfirmed = originalConfirmed
		paymentSetting.ComplianceTermsVersion = originalTerms
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalMethods
	})

	operation_setting.PayAddress = "https://epay.example.com"
	operation_setting.EpayId = "epay-id"
	operation_setting.EpayKey = "epay-key"
	operation_setting.PayMethods = []map[string]string{
		{"name": "支付宝", "type": "alipay", "color": "blue"},
		{"name": "微信", "type": "wxpay", "color": "green"},
	}

	paymentSetting.ComplianceConfirmed = false
	paymentSetting.ComplianceTermsVersion = ""
	response := invoiceConfigResponseForTest(t)
	require.True(t, response.Success)
	assert.Equal(t, []string{model.PaymentMethodBalance}, invoicePaymentMethodTypes(response.Data.PayMethods))

	paymentSetting.ComplianceConfirmed = true
	paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	response = invoiceConfigResponseForTest(t)
	require.True(t, response.Success)
	assert.ElementsMatch(t, []string{
		model.PaymentMethodBalance, "alipay", "wxpay",
	}, invoicePaymentMethodTypes(response.Data.PayMethods))
	for _, method := range response.Data.PayMethods {
		assert.NotEmpty(t, method["name"])
		assert.NotEmpty(t, method["type"])
		assert.NotEmpty(t, method["provider"])
		assert.NotEmpty(t, method["color"])
	}

	operation_setting.EpayKey = ""
	response = invoiceConfigResponseForTest(t)
	assert.Equal(t, []string{model.PaymentMethodBalance}, invoicePaymentMethodTypes(response.Data.PayMethods))
}
