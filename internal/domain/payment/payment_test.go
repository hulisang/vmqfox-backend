package payment

import "testing"

// 黄金向量由 OpenSSL 独立计算，PHP 商户插件与安卓挂机端必须复现同一组结果，
// 任何一端改动导致向量不一致即为协议破坏。
const (
	vectorKey         = "testkey123456"
	vectorPayID       = "TEST20260314001"
	vectorType        = "1"
	vectorPrice       = "1.00"
	vectorReallyPrice = "0.99"
	vectorNotifyURL   = "https://shop.example.com/notify"
	vectorReturnURL   = "https://shop.example.com/return"
	vectorTimestamp   = "1773500000000"
)

func TestSignV2GoldenVectors(t *testing.T) {
	cases := []struct {
		name   string
		actual string
		want   string
	}{
		{
			name:   "create",
			actual: CreateSignV2(vectorPayID, "", vectorType, vectorPrice, vectorNotifyURL, vectorReturnURL, vectorKey),
			want:   "729a6c529b4a2ffed215d124a7e4244ed5d4981ba1982d4cd6f9a53b28de9263",
		},
		{
			name:   "callback",
			actual: CallbackSignV2(vectorPayID, "", vectorType, vectorPrice, vectorReallyPrice, vectorKey),
			want:   "0f21b9366a12396a71437d336a21fd7c5fe20b292e9b56d25b6b612a144daa60",
		},
		{
			name:   "heartbeat",
			actual: HeartbeatSignV2(vectorTimestamp, vectorKey),
			want:   "71ec70d2383ec10bdb2fe39f42f60721a1808fa31884deebc9b8c88b5b527bbe",
		},
		{
			name:   "push",
			actual: PushSignV2(vectorType, vectorPrice, vectorTimestamp, vectorKey),
			want:   "61174e2503aec9c2ffe8430ba322d03b8e3c5f46c3d08f29e69ee35d0b34a51e",
		},
	}

	for _, item := range cases {
		if item.actual != item.want {
			t.Errorf("%s 签名向量不一致\n实际: %s\n期望: %s", item.name, item.actual, item.want)
		}
	}
}

// TestCreateSignV2CoversCallbackURLs 锁定本次修复的核心性质：改写回调或回跳地址必然导致签名变化。
func TestCreateSignV2CoversCallbackURLs(t *testing.T) {
	base := CreateSignV2(vectorPayID, "", vectorType, vectorPrice, vectorNotifyURL, vectorReturnURL, vectorKey)

	tampered := map[string]string{
		"篡改 notifyUrl": CreateSignV2(vectorPayID, "", vectorType, vectorPrice, "https://attacker.example.com/notify", vectorReturnURL, vectorKey),
		"篡改 returnUrl": CreateSignV2(vectorPayID, "", vectorType, vectorPrice, vectorNotifyURL, "https://attacker.example.com/return", vectorKey),
		"清空 notifyUrl": CreateSignV2(vectorPayID, "", vectorType, vectorPrice, "", vectorReturnURL, vectorKey),
		"清空 returnUrl": CreateSignV2(vectorPayID, "", vectorType, vectorPrice, vectorNotifyURL, "", vectorKey),
	}

	for name, sign := range tampered {
		if sign == base {
			t.Errorf("%s 后签名未变化，回调地址仍在签名域之外", name)
		}
	}
}

func TestCreateSignV2DiffersFromLegacy(t *testing.T) {
	v2 := CreateSignV2(vectorPayID, "", vectorType, vectorPrice, vectorNotifyURL, vectorReturnURL, vectorKey)
	v1 := LegacyCreateSign(vectorPayID, "", vectorType, vectorPrice, vectorKey)
	if v2 == v1 {
		t.Fatal("v2 与已废弃的 v1 签名相同，版本切换无效")
	}
	if len(v2) != 64 {
		t.Fatalf("v2 签名应为 64 位十六进制，实际长度 %d", len(v2))
	}
	if len(v1) != 32 {
		t.Fatalf("v1 签名应为 32 位十六进制，实际长度 %d", len(v1))
	}
}

func TestSignEqual(t *testing.T) {
	sign := HeartbeatSignV2(vectorTimestamp, vectorKey)
	if !SignEqual(sign, sign) {
		t.Fatal("相同签名应判定相等")
	}
	if SignEqual(sign, sign[:len(sign)-1]) {
		t.Fatal("长度不同的签名不应判定相等")
	}
	if SignEqual(sign, "") {
		t.Fatal("空签名不应判定相等")
	}
}