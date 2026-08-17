package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTopUpInvoiceControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.RedisEnabled = false
	model.InitColForTest()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Log{},
		&model.TopUp{},
		&model.SubscriptionOrder{},
		&model.UserSubscription{},
		&model.InvoiceRecord{},
		&model.InvoiceOrderLink{},
	))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func newTopUpPaymentContext(t *testing.T, body any, userID int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	payload, err := common.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/user/amount", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", userID)
	return ctx, recorder
}

func setupTopUpInvoiceControllerTest(t *testing.T) {
	t.Helper()

	db := setupTopUpInvoiceControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       901,
		Username: common.GetRandomString(8),
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  common.GetRandomString(8),
	}).Error)

	originalInvoiceEnabled := model.InvoiceEnabled
	originalInvoiceTypes := model.InvoiceTypes
	originalInvoiceFeeRules := model.InvoiceFeeRules
	originalPrice := operation_setting.Price
	originalMinTopUp := operation_setting.MinTopUp
	originalPayMethods := operation_setting.PayMethods
	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey

	model.InvoiceEnabled = true
	model.InvoiceTypes = `["personal","company"]`
	model.InvoiceFeeRules = `[{"min":0,"max":500,"type":"fixed","value":50}]`
	operation_setting.Price = 1
	operation_setting.MinTopUp = 1
	operation_setting.PayMethods = []map[string]string{{"type": "alipay", "name": "支付宝"}}
	operation_setting.PayAddress = "https://pay.example.com"
	operation_setting.EpayId = "epay_id"
	operation_setting.EpayKey = "epay_key"

	t.Cleanup(func() {
		model.InvoiceEnabled = originalInvoiceEnabled
		model.InvoiceTypes = originalInvoiceTypes
		model.InvoiceFeeRules = originalInvoiceFeeRules
		operation_setting.Price = originalPrice
		operation_setting.MinTopUp = originalMinTopUp
		operation_setting.PayMethods = originalPayMethods
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
	})
}

func TestTopUpRequestAmount_PreviewsInvoiceFeeWithoutTitle(t *testing.T) {
	setupTopUpInvoiceControllerTest(t)

	ctx, recorder := newTopUpPaymentContext(t, AmountRequest{
		Amount: 100,
		Invoice: model.InvoiceRequest{
			Required: true,
			Type:     model.InvoiceTypePersonal,
		},
	}, 901)
	RequestAmount(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Message          string  `json:"message"`
		Data             string  `json:"data"`
		InvoiceRequired  bool    `json:"invoice_required"`
		InvoiceBase      float64 `json:"invoice_base_amount"`
		InvoiceFee       float64 `json:"invoice_fee"`
		InvoiceTotal     float64 `json:"invoice_total_amount"`
		InvoicePayment   float64 `json:"invoice_payment_amount"`
		InvoiceFeePay    float64 `json:"invoice_fee_payment_amount"`
		InvoiceCurrency  string  `json:"invoice_currency"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "success", response.Message)
	assert.Equal(t, "150.00", response.Data)
	assert.True(t, response.InvoiceRequired)
	assert.Equal(t, 100.0, response.InvoiceBase)
	assert.Equal(t, 50.0, response.InvoiceFee)
	assert.Equal(t, 150.0, response.InvoiceTotal)
	assert.Equal(t, 150.0, response.InvoicePayment)
	assert.Equal(t, 50.0, response.InvoiceFeePay)
	assert.Equal(t, "CNY", response.InvoiceCurrency)
}

func TestTopUpRequestEpay_RequiresTitleWhenInvoiceRequired(t *testing.T) {
	setupTopUpInvoiceControllerTest(t)

	ctx, recorder := newTopUpPaymentContext(t, EpayRequest{
		Amount:        100,
		PaymentMethod: "alipay",
		Invoice: model.InvoiceRequest{
			Required: true,
			Type:     model.InvoiceTypeCompany,
		},
	}, 901)
	RequestEpay(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Message string `json:"message"`
		Data    string `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "error", response.Message)
	assert.Equal(t, "请填写发票抬头", response.Data)
}
