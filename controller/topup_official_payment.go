package controller

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const (
	alipayTradeStatusSuccess  = "TRADE_SUCCESS"
	alipayTradeStatusFinished = "TRADE_FINISHED"
	wechatPayGateway          = "https://api.mch.weixin.qq.com"
)

type officialPaymentOrder struct {
	tradeNo string
	name    string
	money   float64
}

type wechatNativePayResponse struct {
	CodeURL string `json:"code_url"`
}

type wechatPayNotifyRequest struct {
	Resource struct {
		Algorithm      string `json:"algorithm"`
		Ciphertext     string `json:"ciphertext"`
		AssociatedData string `json:"associated_data"`
		Nonce          string `json:"nonce"`
	} `json:"resource"`
}

type wechatPayTransaction struct {
	OutTradeNo string `json:"out_trade_no"`
	TradeState string `json:"trade_state"`
	Amount     struct {
		Total int64 `json:"total"`
	} `json:"amount"`
}

func isOfficialPaymentMethod(method string) bool {
	return method == model.PaymentMethodOfficialAlipay || method == model.PaymentMethodOfficialWeChat
}

func containsConfiguredPaymentMethod(method string) bool {
	if operation_setting.ContainsPayMethod(method) {
		return true
	}
	switch method {
	case model.PaymentMethodOfficialAlipay:
		return isOfficialAlipayConfigured()
	case model.PaymentMethodOfficialWeChat:
		return isOfficialWeChatPayConfigured()
	default:
		return false
	}
}

func requestOfficialTopUpPay(c *gin.Context, req EpayRequest, userId int, payMoney float64) {
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}
	tradeNo := fmt.Sprintf("USR%dNO%s%d", userId, common.GetRandomString(6), time.Now().Unix())
	order := officialPaymentOrder{
		tradeNo: tradeNo,
		name:    fmt.Sprintf("TUC%d", req.Amount),
		money:   payMoney,
	}
	topUp := &model.TopUp{
		UserId:          userId,
		Amount:          amount,
		Money:           payMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   req.PaymentMethod,
		PaymentProvider: officialPaymentProvider(req.PaymentMethod),
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("官方支付 创建充值订单失败 user_id=%d trade_no=%s payment_method=%s amount=%d error=%q", userId, tradeNo, req.PaymentMethod, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}
	respondOfficialPayment(c, req.PaymentMethod, order, "/console/log")
}

func requestOfficialSubscriptionPay(c *gin.Context, method string, userId int, plan *model.SubscriptionPlan) {
	tradeNo := fmt.Sprintf("SUBUSR%dNO%s%d", userId, common.GetRandomString(6), time.Now().Unix())
	order := &model.SubscriptionOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           plan.PriceAmount,
		TradeNo:         tradeNo,
		PaymentMethod:   method,
		PaymentProvider: officialPaymentProvider(method),
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}
	respondOfficialPayment(c, method, officialPaymentOrder{
		tradeNo: tradeNo,
		name:    fmt.Sprintf("SUB:%s", plan.Title),
		money:   plan.PriceAmount,
	}, "/console/topup?show_history=true")
}

func respondOfficialPayment(c *gin.Context, method string, order officialPaymentOrder, returnSuffix string) {
	switch method {
	case model.PaymentMethodOfficialAlipay:
		uri, params, err := buildOfficialAlipayPayment(order, officialPaymentCallbackAddress(c), officialPaymentReturnPath(c, returnSuffix))
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方 拉起支付失败 trade_no=%s error=%q", order.tradeNo, err.Error()))
			c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": params, "url": uri})
	case model.PaymentMethodOfficialWeChat:
		payLink, err := buildOfficialWeChatPayment(c, order)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付官方 拉起支付失败 trade_no=%s error=%q", order.tradeNo, err.Error()))
			c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"pay_link": payLink}})
	default:
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付方式不存在"})
	}
}

func officialPaymentProvider(method string) string {
	switch method {
	case model.PaymentMethodOfficialAlipay:
		return model.PaymentProviderOfficialAlipay
	case model.PaymentMethodOfficialWeChat:
		return model.PaymentProviderOfficialWeChat
	default:
		return ""
	}
}

func officialPaymentCallbackAddress(c *gin.Context) string {
	if callbackAddress := strings.TrimSpace(operation_setting.CustomCallbackAddress); callbackAddress != "" {
		return strings.TrimRight(callbackAddress, "/")
	}
	if origin := requestPublicOrigin(c.Request); origin != "" {
		return origin
	}
	return strings.TrimRight(service.GetCallbackAddress(), "/")
}

func officialPaymentReturnPath(c *gin.Context, suffix string) string {
	if origin := requestPublicOrigin(c.Request); origin != "" {
		return origin + common.ThemeAwarePath(suffix)
	}
	return paymentReturnPath(suffix)
}

func requestPublicOrigin(r *http.Request) string {
	if r == nil {
		return ""
	}
	host := firstForwardedHeaderValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if host == "" || strings.ContainsAny(host, "/\\") {
		return ""
	}
	proto := firstForwardedHeaderValue(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	proto = strings.ToLower(proto)
	if proto != "http" && proto != "https" {
		return ""
	}
	return proto + "://" + strings.TrimRight(host, "/")
}

func firstForwardedHeaderValue(value string) string {
	if index := strings.Index(value, ","); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}

func gatewayWithQueryParam(gateway string, key string, value string) (string, error) {
	uri, err := url.Parse(gateway)
	if err != nil {
		return "", err
	}
	query := uri.Query()
	query.Set(key, value)
	uri.RawQuery = query.Encode()
	return uri.String(), nil
}

func buildOfficialAlipayPayment(order officialPaymentOrder, callbackAddress string, returnURL string) (string, map[string]string, error) {
	privateKey, err := parseRSAPrivateKey(setting.OfficialAlipayPrivateKey)
	if err != nil {
		return "", nil, err
	}
	gateway := strings.TrimSpace(setting.OfficialAlipayGateway)
	if gateway == "" {
		gateway = "https://openapi.alipay.com/gateway.do"
	}
	bizContent, err := common.Marshal(map[string]any{
		"out_trade_no": order.tradeNo,
		"product_code": "FAST_INSTANT_TRADE_PAY",
		"subject":      order.name,
		"total_amount": decimal.NewFromFloat(order.money).Round(2).StringFixed(2),
	})
	if err != nil {
		return "", nil, err
	}
	params := map[string]string{
		"app_id":      strings.TrimSpace(setting.OfficialAlipayAppID),
		"method":      "alipay.trade.page.pay",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  callbackAddress + "/api/official/alipay/notify",
		"return_url":  returnURL,
		"biz_content": string(bizContent),
	}
	if params["app_id"] == "" {
		return "", nil, errors.New("OfficialAlipayAppID is not configured")
	}
	signature, err := rsaSign(alipaySignContent(params, true), privateKey)
	if err != nil {
		return "", nil, err
	}
	params["sign"] = signature
	delete(params, "charset")
	uri, err := gatewayWithQueryParam(gateway, "charset", "utf-8")
	if err != nil {
		return "", nil, err
	}
	return uri, params, nil
}

func OfficialAlipayNotify(c *gin.Context) {
	params, err := requestParams(c)
	if err != nil || len(params) == 0 {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	if !verifyAlipayParams(params) {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝官方 webhook 验签失败 path=%q client_ip=%s params=%q", c.Request.RequestURI, c.ClientIP(), common.GetJsonString(params)))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	tradeNo := strings.TrimSpace(params["out_trade_no"])
	status := strings.TrimSpace(params["trade_status"])
	if status != alipayTradeStatusSuccess && status != alipayTradeStatusFinished {
		_, _ = c.Writer.Write([]byte("success"))
		return
	}
	paidMoney, _ := strconv.ParseFloat(params["total_amount"], 64)
	if strings.HasPrefix(tradeNo, "SUBUSR") {
		if err := completeOfficialSubscription(c, tradeNo, model.PaymentProviderOfficialAlipay, model.PaymentMethodOfficialAlipay, paidMoney, params); err != nil {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝官方 订阅订单完成失败 trade_no=%s error=%q", tradeNo, err.Error()))
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
	} else {
		if err := completeOfficialTopUp(c, tradeNo, model.PaymentProviderOfficialAlipay, model.PaymentMethodOfficialAlipay, paidMoney, params); err != nil {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝官方 充值订单完成失败 trade_no=%s error=%q", tradeNo, err.Error()))
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
	}
	_, _ = c.Writer.Write([]byte("success"))
}

func OfficialAlipayReturn(c *gin.Context) {
	params, err := requestParams(c)
	if err == nil && len(params) > 0 && verifyAlipayParams(params) {
		tradeNo := strings.TrimSpace(params["out_trade_no"])
		paidMoney, _ := strconv.ParseFloat(params["total_amount"], 64)
		if strings.HasPrefix(tradeNo, "SUBUSR") {
			_ = completeOfficialSubscription(c, tradeNo, model.PaymentProviderOfficialAlipay, model.PaymentMethodOfficialAlipay, paidMoney, params)
		} else {
			_ = completeOfficialTopUp(c, tradeNo, model.PaymentProviderOfficialAlipay, model.PaymentMethodOfficialAlipay, paidMoney, params)
		}
	}
	c.Redirect(http.StatusFound, paymentReturnPath("/console/topup?show_history=true"))
}

func buildOfficialWeChatPayment(c *gin.Context, order officialPaymentOrder) (string, error) {
	privateKey, err := parseRSAPrivateKey(setting.OfficialWeChatPayPrivateKey)
	if err != nil {
		return "", err
	}
	appID := strings.TrimSpace(setting.OfficialWeChatPayAppID)
	mchID := strings.TrimSpace(setting.OfficialWeChatPayMchID)
	serialNo := strings.TrimSpace(setting.OfficialWeChatPayMchSerialNo)
	if appID == "" || mchID == "" || serialNo == "" {
		return "", errors.New("WeChat Pay app id, mch id, or merchant serial number is not configured")
	}
	callbackAddress := officialPaymentCallbackAddress(c)
	bodyBytes, err := common.Marshal(map[string]any{
		"appid":        appID,
		"mchid":        mchID,
		"description":  order.name,
		"out_trade_no": order.tradeNo,
		"notify_url":   callbackAddress + "/api/official/wechat-pay/notify",
		"amount": map[string]any{
			"total":    amountToCents(order.money),
			"currency": "CNY",
		},
	})
	if err != nil {
		return "", err
	}
	uri := wechatPayGateway + "/v3/pay/transactions/native"
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, uri, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", wechatAuthorizationHeader(http.MethodPost, "/v3/pay/transactions/native", string(bodyBytes), mchID, serialNo, privateKey))
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("wechat pay status %d: %s", resp.StatusCode, string(respBody))
	}
	var data wechatNativePayResponse
	if err := common.Unmarshal(respBody, &data); err != nil {
		return "", err
	}
	if strings.TrimSpace(data.CodeURL) == "" {
		return "", errors.New("wechat pay response code_url is empty")
	}
	values := url.Values{}
	values.Set("code_url", data.CodeURL)
	values.Set("trade_no", order.tradeNo)
	return officialPaymentReturnPath(c, "/console/wechat-pay?"+values.Encode()), nil
}

func OfficialWeChatPayNotify(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		wechatPayNotifyFail(c, "read body failed")
		return
	}
	if !verifyWeChatPaySignature(c, body) {
		wechatPayNotifyFail(c, "signature verification failed")
		return
	}
	var req wechatPayNotifyRequest
	if err := common.Unmarshal(body, &req); err != nil {
		wechatPayNotifyFail(c, "invalid request body")
		return
	}
	plain, err := decryptWeChatResource(req.Resource.Ciphertext, req.Resource.Nonce, req.Resource.AssociatedData)
	if err != nil {
		wechatPayNotifyFail(c, err.Error())
		return
	}
	var tx wechatPayTransaction
	if err := common.Unmarshal([]byte(plain), &tx); err != nil {
		wechatPayNotifyFail(c, "invalid transaction body")
		return
	}
	if tx.TradeState != "SUCCESS" {
		wechatPayNotifySuccess(c)
		return
	}
	paidMoney := decimal.NewFromInt(tx.Amount.Total).Div(decimal.NewFromInt(100)).InexactFloat64()
	if strings.HasPrefix(tx.OutTradeNo, "SUBUSR") {
		if err := completeOfficialSubscription(c, tx.OutTradeNo, model.PaymentProviderOfficialWeChat, model.PaymentMethodOfficialWeChat, paidMoney, plain); err != nil {
			wechatPayNotifyFail(c, err.Error())
			return
		}
	} else {
		if err := completeOfficialTopUp(c, tx.OutTradeNo, model.PaymentProviderOfficialWeChat, model.PaymentMethodOfficialWeChat, paidMoney, plain); err != nil {
			wechatPayNotifyFail(c, err.Error())
			return
		}
	}
	wechatPayNotifySuccess(c)
}

func OfficialPaymentStatus(c *gin.Context) {
	tradeNo := strings.TrimSpace(c.Query("trade_no"))
	userId := c.GetInt("id")
	if tradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if strings.HasPrefix(tradeNo, "SUBUSR") {
		order := model.GetSubscriptionOrderByTradeNo(tradeNo)
		if order == nil || order.UserId != userId {
			common.ApiErrorMsg(c, "订单不存在")
			return
		}
		common.ApiSuccess(c, gin.H{"status": order.Status})
		return
	}
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil || topUp.UserId != userId {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	common.ApiSuccess(c, gin.H{"status": topUp.Status})
}

func completeOfficialTopUp(c *gin.Context, tradeNo string, provider string, method string, paidMoney float64, payload any) error {
	if tradeNo == "" {
		return errors.New("trade_no is empty")
	}
	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		return model.ErrTopUpNotFound
	}
	if topUp.PaymentProvider != provider {
		return model.ErrPaymentMethodMismatch
	}
	if topUp.Status != common.TopUpStatusPending {
		return nil
	}
	if !paymentAmountMatches(topUp.Money, paidMoney) {
		return fmt.Errorf("paid amount %.2f does not match order amount %.2f", paidMoney, topUp.Money)
	}
	topUp.PaymentMethod = method
	topUp.Status = common.TopUpStatusSuccess
	topUp.CompleteTime = common.GetTimestamp()
	if err := topUp.Update(); err != nil {
		return err
	}
	quotaToAdd := int(decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	if err := model.IncreaseUserQuota(topUp.UserId, quotaToAdd, true); err != nil {
		return err
	}
	model.RecordTopupLog(topUp.UserId, fmt.Sprintf("使用官方支付充值成功，充值金额: %v，支付金额：%.2f", logger.LogQuota(quotaToAdd), topUp.Money), c.ClientIP(), topUp.PaymentMethod, provider)
	return nil
}

func completeOfficialSubscription(c *gin.Context, tradeNo string, provider string, method string, paidMoney float64, payload any) error {
	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil {
		return model.ErrSubscriptionOrderNotFound
	}
	if !paymentAmountMatches(order.Money, paidMoney) {
		return fmt.Errorf("paid amount %.2f does not match order amount %.2f", paidMoney, order.Money)
	}
	return model.CompleteSubscriptionOrder(tradeNo, common.GetJsonString(payload), provider, method)
}

func requestParams(c *gin.Context) (map[string]string, error) {
	if c.Request.Method == http.MethodPost {
		if err := c.Request.ParseForm(); err != nil {
			return nil, err
		}
	} else {
		c.Request.Form = c.Request.URL.Query()
	}
	params := make(map[string]string, len(c.Request.Form))
	for key := range c.Request.Form {
		params[key] = c.Request.Form.Get(key)
	}
	return params, nil
}

func verifyAlipayParams(params map[string]string) bool {
	signature := strings.TrimSpace(params["sign"])
	if signature == "" {
		return false
	}
	publicKey, err := parseRSAPublicKey(setting.OfficialAlipayPublicKey)
	if err != nil {
		return false
	}
	signBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	digest := sha256.Sum256([]byte(alipaySignContent(params, false)))
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signBytes) == nil
}

func alipaySignContent(params map[string]string, includeSignType bool) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "sign_type" && !includeSignType {
			continue
		}
		if key == "sign" || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, "&")
}

func wechatAuthorizationHeader(method string, requestPath string, body string, mchID string, serialNo string, privateKey *rsa.PrivateKey) string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := randomString(32)
	message := method + "\n" + requestPath + "\n" + timestamp + "\n" + nonce + "\n" + body + "\n"
	signature, _ := rsaSign(message, privateKey)
	return fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",timestamp="%s",serial_no="%s",signature="%s"`,
		mchID,
		nonce,
		timestamp,
		serialNo,
		signature,
	)
}

func verifyWeChatPaySignature(c *gin.Context, body []byte) bool {
	timestamp := c.GetHeader("Wechatpay-Timestamp")
	nonce := c.GetHeader("Wechatpay-Nonce")
	signature := c.GetHeader("Wechatpay-Signature")
	if timestamp == "" || nonce == "" || signature == "" {
		return false
	}
	publicKey, err := parseRSAPublicKey(setting.OfficialWeChatPayPlatformPublicKey)
	if err != nil {
		return false
	}
	signBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	digest := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signBytes) == nil
}

func decryptWeChatResource(ciphertext string, nonce string, associatedData string) (string, error) {
	apiKey := strings.TrimSpace(setting.OfficialWeChatPayAPIv3Key)
	if len(apiKey) != 32 {
		return "", errors.New("WeChat Pay API v3 key must be 32 bytes")
	}
	cipherBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher([]byte(apiKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plain, err := gcm.Open(nil, []byte(nonce), cipherBytes, []byte(associatedData))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func wechatPayNotifySuccess(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "成功"})
}

func wechatPayNotifyFail(c *gin.Context, message string) {
	logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付官方 webhook 失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), message))
	c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": message})
}

func parseRSAPrivateKey(value string) (*rsa.PrivateKey, error) {
	block, err := pemBlock(value, "PRIVATE KEY")
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return key, nil
}

func parseRSAPublicKey(value string) (*rsa.PublicKey, error) {
	block, err := pemBlock(value, "PUBLIC KEY")
	if err != nil {
		return nil, err
	}
	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if key, ok := pub.(*rsa.PublicKey); ok {
			return key, nil
		}
	}
	if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
		if key, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			return key, nil
		}
	}
	return nil, errors.New("public key is not RSA")
}

func pemBlock(value string, fallbackType string) (*pem.Block, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(value, `\n`, "\n"))
	if normalized == "" {
		return nil, errors.New("key is empty")
	}
	if !strings.Contains(normalized, "-----BEGIN") {
		normalized = fmt.Sprintf("-----BEGIN %s-----\n%s\n-----END %s-----", fallbackType, normalized, fallbackType)
	}
	block, _ := pem.Decode([]byte(normalized))
	if block == nil {
		return nil, errors.New("invalid PEM key")
	}
	return block, nil
}

func rsaSign(content string, privateKey *rsa.PrivateKey) (string, error) {
	digest := sha256.Sum256([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func randomString(length int) string {
	const letters = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if length <= 0 {
		return ""
	}
	max := big.NewInt(int64(len(letters)))
	out := make([]byte, length)
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = letters[time.Now().UnixNano()%int64(len(letters))]
			continue
		}
		out[i] = letters[n.Int64()]
	}
	return string(out)
}

func amountToCents(amount float64) int64 {
	return decimal.NewFromFloat(amount).Mul(decimal.NewFromInt(100)).Round(0).IntPart()
}

func paymentAmountMatches(expected float64, actual float64) bool {
	return amountToCents(actual) >= amountToCents(expected)
}
