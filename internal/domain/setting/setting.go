package setting

const (
	MerchantKey     = "key"
	NotifyURLKey    = "notifyUrl"
	ReturnURLKey    = "returnUrl"
	LastHeartKey    = "lastheart"
	LastPayKey      = "lastpay"
	MonitorStateKey = "jkstate"
	CloseMinutesKey = "close"
	PriceAdjustKey  = "payQf"
	WechatPayURLKey = "wxpay"
	AlipayPayURLKey = "zfbpay"
)

var allKeys = []string{
	MerchantKey,
	NotifyURLKey,
	ReturnURLKey,
	LastHeartKey,
	LastPayKey,
	MonitorStateKey,
	CloseMinutesKey,
	PriceAdjustKey,
	WechatPayURLKey,
	AlipayPayURLKey,
}

func AllKeys() []string {
	return append([]string(nil), allKeys...)
}

func IsSensitiveKey(key string) bool {
	switch key {
	case MerchantKey, WechatPayURLKey, AlipayPayURLKey:
		return true
	default:
		return false
	}
}
