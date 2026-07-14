package setting

const (
	PaymentMethodOfficialAlipay = "alipay_official"
	PaymentMethodOfficialWeChat = "wechat_pay"

	PaymentProviderOfficialAlipay = "official_alipay"
	PaymentProviderOfficialWeChat = "official_wechat_pay"
)

var (
	OfficialAlipayAppID      = ""
	OfficialAlipayPrivateKey = ""
	OfficialAlipayPublicKey  = ""
	OfficialAlipayGateway    = "https://openapi.alipay.com/gateway.do"

	OfficialWeChatPayAppID             = ""
	OfficialWeChatPayMchID             = ""
	OfficialWeChatPayMchSerialNo       = ""
	OfficialWeChatPayAPIv3Key          = ""
	OfficialWeChatPayPrivateKey        = ""
	OfficialWeChatPayPlatformPublicKey = ""
)
