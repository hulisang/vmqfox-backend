package payment

import (
	"crypto/md5"
	"encoding/hex"
)

type Type int8

const (
	Wechat Type = 1
	Alipay Type = 2
)

func (t Type) Valid() bool {
	return t == Wechat || t == Alipay
}

func HeartbeatSign(timestamp, key string) string {
	return md5Hex(timestamp + key)
}

func PushSign(paymentType, price, timestamp, key string) string {
	return md5Hex(paymentType + price + timestamp + key)
}

func CreateSignNew(payID, param, paymentType, price, key string) string {
	return md5Hex(
		"payId=" + payID +
			"&param=" + param +
			"&type=" + paymentType +
			"&price=" + price +
			"&key=" + key,
	)
}

func CreateSignLegacy(payID, param, paymentType, price, key string) string {
	return md5Hex(payID + param + paymentType + price + key)
}

func CallbackSignNew(payID, param, paymentType, price, reallyPrice, key string) string {
	return md5Hex(
		"payId=" + payID +
			"&param=" + param +
			"&type=" + paymentType +
			"&price=" + price +
			"&reallyPrice=" + reallyPrice +
			"&key=" + key,
	)
}

func CallbackSignLegacy(payID, param, paymentType, price, reallyPrice, key string) string {
	return md5Hex(payID + param + paymentType + price + reallyPrice + key)
}

func md5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}
