package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// SignVersion2 是当前唯一被接受的签名版本：HMAC-SHA-256，密钥作为 HMAC 密钥而不拼入明文。
const SignVersion2 = "2"

type Type int8

const (
	Wechat Type = 1
	Alipay Type = 2
)

func (t Type) Valid() bool {
	return t == Wechat || t == Alipay
}

// CreateSignV2 把 notifyUrl 与 returnUrl 纳入签名域，使回调地址与回跳地址无法在传输途中被改写。
func CreateSignV2(payID, param, paymentType, price, notifyURL, returnURL, key string) string {
	return hmacHex(key,
		"payId="+payID+
			"&param="+param+
			"&type="+paymentType+
			"&price="+price+
			"&notifyUrl="+notifyURL+
			"&returnUrl="+returnURL)
}

func CallbackSignV2(payID, param, paymentType, price, reallyPrice, key string) string {
	return hmacHex(key,
		"payId="+payID+
			"&param="+param+
			"&type="+paymentType+
			"&price="+price+
			"&reallyPrice="+reallyPrice)
}

func HeartbeatSignV2(timestamp, key string) string {
	return hmacHex(key, "t="+timestamp)
}

// QueryByPayIDSignV2 是商户按 payId 只读查询的签名。时间戳字段名与挂机端一致为 t。
func QueryByPayIDSignV2(payID, timestamp, key string) string {
	return hmacHex(key, "payId="+payID+"&t="+timestamp)
}

func PushSignV2(paymentType, price, timestamp, key string) string {
	return hmacHex(key,
		"type="+paymentType+
			"&price="+price+
			"&t="+timestamp)
}

// SignEqual 以常量时间比较签名，避免比较耗时泄露正确前缀。
func SignEqual(actual, expected string) bool {
	return len(actual) == len(expected) &&
		subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func hmacHex(key, canonical string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}
