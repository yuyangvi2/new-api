package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type invoiceExternalPaymentRequest struct {
	Orders        []model.InvoiceOrderReference `json:"orders"`
	Invoice       model.InvoiceRequest          `json:"invoice"`
	PaymentMethod string                        `json:"payment_method"`
	TradeType     string                        `json:"trade_type"`
}

func availableInvoicePayMethods() ([]map[string]string, []any) {
	methods := []map[string]string{{
		"name": "余额", "type": model.PaymentMethodBalance,
		"provider": model.PaymentProviderBalance, "color": "rgba(var(--semi-blue-5), 1)",
	}}
	if !operation_setting.IsPaymentComplianceConfirmed() {
		return methods, []any{}
	}
	if isEpayTopUpEnabled() {
		for _, configured := range operation_setting.PayMethods {
			paymentType := strings.TrimSpace(configured["type"])
			if paymentType == "" || paymentType == model.PaymentMethodBalance || paymentType == model.PaymentMethodBepusdt || paymentType == model.PaymentMethodOkpay {
				continue
			}
			name := strings.TrimSpace(configured["name"])
			if name == "" {
				name = paymentType
			}
			color := strings.TrimSpace(configured["color"])
			if color == "" {
				color = "rgba(var(--semi-blue-5), 1)"
			}
			methods = append(methods, map[string]string{
				"name": name, "type": paymentType,
				"provider": model.PaymentProviderEpay, "color": color,
			})
		}
	}
	return methods, []any{}
}

func invoicePaymentResponse(record *model.InvoiceRecord, checkout gin.H, amountText string) gin.H {
	response := gin.H{
		"completed": record != nil && record.PaymentStatus == model.InvoicePaymentStatusSuccess,
		"trade_no":  "",
		"invoice":   record,
	}
	if record != nil {
		response["trade_no"] = record.SourceId
	}
	if checkout != nil {
		response["checkout"] = checkout
	}
	if amountText != "" {
		response["amount_text"] = amountText
	}
	return response
}

func invoicePaymentCNYText(record *model.InvoiceRecord) string {
	if record == nil {
		return ""
	}
	return invoicePaymentCNYAmount(record) + " CNY"
}

func markInvoicePaymentFailed(record *model.InvoiceRecord, reason error) {
	if record == nil {
		return
	}
	payload := "支付拉起失败"
	if reason != nil {
		payload = reason.Error()
	}
	_ = model.FailInvoiceExternalPaymentAndRelease(
		record.SourceId,
		record.PaymentProvider,
		payload,
	)
}

func invoicePaymentCNYAmount(record *model.InvoiceRecord) string {
	if record == nil {
		return ""
	}
	return decimal.NewFromInt(record.PaymentAmountMinor).Div(decimal.NewFromInt(100)).StringFixed(2)
}

func RequestInvoiceExternalPayment(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	var req invoiceExternalPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.Invoice.Required = true
	req.PaymentMethod = strings.TrimSpace(req.PaymentMethod)
	req.TradeType = strings.TrimSpace(req.TradeType)
	if req.PaymentMethod == "" || req.PaymentMethod == model.PaymentMethodBalance {
		common.ApiErrorMsg(c, "余额支付请使用 /api/user/invoice/request")
		return
	}

	options := model.InvoiceExternalPaymentOptions{
		PaymentMethod: req.PaymentMethod,
		RequestIP:     c.ClientIP(),
	}
	switch req.PaymentMethod {
	default:
		if !isEpayTopUpEnabled() || !operation_setting.ContainsPayMethod(req.PaymentMethod) {
			common.ApiErrorMsg(c, "支付方式不存在或不可用")
			return
		}
		options.PaymentProvider = model.PaymentProviderEpay
		options.ProviderMerchantId = operation_setting.EpayId
	}
	options.ProviderPayload = common.GetJsonString(map[string]string{
		"payment_provider": options.PaymentProvider,
		"payment_method":   options.PaymentMethod,
		"trade_type":       req.TradeType,
	})

	callbackBase := strings.TrimRight(service.GetCallbackAddress(), "/")
	var epayClient *epay.Client
	var notifyURL *url.URL
	var returnURL *url.URL
	if options.PaymentProvider == model.PaymentProviderEpay {
		epayClient = GetEpayClient()
		if epayClient == nil {
			common.ApiErrorMsg(c, "当前管理员未配置支付信息")
			return
		}
		var notifyErr error
		var returnErr error
		notifyURL, notifyErr = url.Parse(callbackBase + "/api/invoice/epay/notify")
		returnURL, returnErr = url.Parse(callbackBase + "/api/invoice/epay/return")
		if notifyErr != nil || returnErr != nil {
			common.ApiErrorMsg(c, "回调地址配置错误")
			return
		}
	}

	record, err := model.CreateCombinedInvoiceExternalPayment(c.GetInt("id"), req.Orders, req.Invoice, options)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if record.PaymentStatus == model.InvoicePaymentStatusSuccess {
		common.ApiSuccess(c, invoicePaymentResponse(record, nil, invoicePaymentCNYText(record)))
		return
	}

	switch options.PaymentProvider {
	case model.PaymentProviderEpay:
		uri, params, err := epayClient.Purchase(&epay.PurchaseArgs{
			Type: req.PaymentMethod, ServiceTradeNo: record.SourceId,
			Name:   "Invoice-" + record.SourceId,
			Money:  decimal.NewFromInt(record.PaymentAmountMinor).Div(decimal.NewFromInt(100)).StringFixed(2),
			Device: epay.PC, NotifyUrl: notifyURL, ReturnUrl: returnURL,
		})
		if err != nil {
			markInvoicePaymentFailed(record, err)
			common.ApiErrorMsg(c, fmt.Sprintf("拉起支付失败，支付记录 %s 已关闭，请重新申请", record.SourceId))
			return
		}
		common.ApiSuccess(c, invoicePaymentResponse(record, gin.H{"type": "form", "url": uri, "params": params}, invoicePaymentCNYText(record)))
	}
}

func CancelInvoiceExternalPayment(c *gin.Context) {
	record, err := model.CancelInvoiceExternalPayment(c.GetInt("id"), c.Param("trade_no"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, record)
}

func GetInvoiceExternalPayment(c *gin.Context) {
	record, err := model.GetUserInvoicePaymentByTradeNo(c.GetInt("id"), c.Param("trade_no"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	amountText := invoicePaymentCNYText(record)
	if record.PaymentProvider == model.PaymentProviderOkpay && record.ProviderAmount != "" && record.ProviderCurrency != "" {
		amountText = record.ProviderAmount + " " + record.ProviderCurrency
	}
	common.ApiSuccess(c, invoicePaymentResponse(record, nil, amountText))
}

func parseInvoiceEpayParams(c *gin.Context) (map[string]string, error) {
	if err := c.Request.ParseForm(); err != nil {
		return nil, err
	}
	params := make(map[string]string, len(c.Request.Form))
	for key := range c.Request.Form {
		params[key] = c.Request.Form.Get(key)
	}
	return params, nil
}

func completeInvoiceEpay(params map[string]string) (*model.InvoiceRecord, error) {
	client := GetEpayClient()
	if client == nil {
		return nil, errors.New("易支付未配置")
	}
	verified, err := client.Verify(params)
	if err != nil || !verified.VerifyStatus {
		return nil, errors.New("易支付回调验签失败")
	}
	if verified.TradeStatus != epay.StatusTradeSuccess {
		return nil, nil
	}
	LockOrder(verified.ServiceTradeNo)
	defer UnlockOrder(verified.ServiceTradeNo)
	record, _, err := model.CompleteInvoiceExternalPayment(verified.ServiceTradeNo, model.InvoicePaymentCallback{
		PaymentProvider: model.PaymentProviderEpay, PaymentMethod: verified.Type,
		ProviderOrderId: verified.TradeNo, ProviderMerchantId: params["pid"],
		ProviderAmount: verified.Money, ProviderCurrency: "CNY",
		ProviderPayload: common.GetJsonString(params),
	})
	return record, err
}

func InvoiceEpayNotify(c *gin.Context) {
	if !isEpayWebhookConfigured() {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	params, err := parseInvoiceEpayParams(c)
	if err != nil || len(params) == 0 {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	record, err := completeInvoiceEpay(params)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("发票易支付回调拒绝 error=%q", err.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	_ = record
	_, _ = c.Writer.Write([]byte("success"))
}

func InvoiceEpayReturn(c *gin.Context) {
	params, err := parseInvoiceEpayParams(c)
	if err != nil || len(params) == 0 || !isEpayWebhookConfigured() {
		c.Redirect(http.StatusFound, paymentReturnPath("/console/invoice?pay=fail"))
		return
	}
	record, err := completeInvoiceEpay(params)
	if err != nil {
		c.Redirect(http.StatusFound, paymentReturnPath("/console/invoice?pay=fail"))
		return
	}
	if record == nil {
		c.Redirect(http.StatusFound, paymentReturnPath("/console/invoice?pay=pending"))
		return
	}
	c.Redirect(http.StatusFound, paymentReturnPath("/console/invoice?pay=success"))
}
