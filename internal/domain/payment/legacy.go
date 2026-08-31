package payment

import (
	"crypto/md5"
	"encoding/hex"
)

// 本文件保留已废弃的 v1 算法（MD5 且把密钥拼入明文）。它只用于识别尚未升级的商户 SDK
// 与旧挂机端，以便返回可操作的升级提示；任何校验路径都不得凭 v1 结果放行请求。

func LegacyCreateSign(payID, param, paymentType, price, key string) string {
	return md5Hex(
		"payId=" + payID +
			"&param=" + param +
			"&type=" + paymentType +
			"&price=" + price +
			"&key=" + key,
	)
}

func LegacyHeartbeatSign(timestamp, key string) string {
	return md5Hex(timestamp + key)
}

func LegacyPushSign(paymentType, price, timestamp, key string) string {
	return md5Hex(paymentType + price + timestamp + key)
}

func md5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}
