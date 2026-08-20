package usecase

import (
	"testing"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
)

const testMerchantKey = "testkey123456"

func signedCreateInput() CreateOrderInput {
	input := CreateOrderInput{
		PayID:     "TEST20260314001",
		Type:      "1",
		Price:     "1.00",
		NotifyURL: "https://shop.example.com/notify",
		ReturnURL: "https://shop.example.com/return",
	}
	input.Sign = payment.CreateSignV2(
		input.PayID, input.Param, input.Type, input.Price,
		input.NotifyURL, input.ReturnURL, testMerchantKey,
	)
	return input
}

func TestValidCreateSignAcceptsUntamperedRequest(t *testing.T) {
	if !validCreateSign(signedCreateInput(), testMerchantKey) {
		t.Fatal("未被篡改的 v2 请求应通过验签")
	}
}

// TestValidCreateSignRejectsTamperedCallbackURLs 覆盖 P0 根因：签名域必须锁住回调与回跳地址。
func TestValidCreateSignRejectsTamperedCallbackURLs(t *testing.T) {
	cases := map[string]func(CreateOrderInput) CreateOrderInput{
		"改写 notifyUrl 指向攻击者": func(in CreateOrderInput) CreateOrderInput {
			in.NotifyURL = "https://attacker.example.com/notify"
			return in
		},
		"改写 returnUrl 指向钓鱼站": func(in CreateOrderInput) CreateOrderInput {
			in.ReturnURL = "https://attacker.example.com/return"
			return in
		},
		"抹掉 notifyUrl": func(in CreateOrderInput) CreateOrderInput {
			in.NotifyURL = ""
			return in
		},
		"改写金额": func(in CreateOrderInput) CreateOrderInput {
			in.Price = "0.01"
			return in
		},
	}

	for name, tamper := range cases {
		if validCreateSign(tamper(signedCreateInput()), testMerchantKey) {
			t.Errorf("%s 后仍通过验签", name)
		}
	}
}

func TestUsesLegacyCreateSignOnlyMatchesOldAlgorithm(t *testing.T) {
	legacy := signedCreateInput()
	legacy.Sign = payment.LegacyCreateSign(legacy.PayID, legacy.Param, legacy.Type, legacy.Price, testMerchantKey)

	if validCreateSign(legacy, testMerchantKey) {
		t.Fatal("v1 签名必须被拒绝")
	}
	if !usesLegacyCreateSign(legacy, testMerchantKey) {
		t.Fatal("v1 签名应被识别出来以便返回升级提示")
	}
	if usesLegacyCreateSign(signedCreateInput(), testMerchantKey) {
		t.Fatal("v2 签名不应被误判为 v1")
	}
}

func TestValidCallbackURL(t *testing.T) {
	allowed := []string{"", "http://shop.example.com/notify", "https://shop.example.com/return?a=1"}
	for _, item := range allowed {
		if !validCallbackURL(item) {
			t.Errorf("%q 应被允许", item)
		}
	}

	rejected := []string{
		"javascript:alert(document.cookie)",
		"data:text/html;base64,PHNjcmlwdD4=",
		"file:///etc/passwd",
		"https://",
		"shop.example.com/notify",
	}
	for _, item := range rejected {
		if validCallbackURL(item) {
			t.Errorf("%q 应被拒绝", item)
		}
	}
}

func TestValidMonitorSigns(t *testing.T) {
	timestamp := "1773500000000"

	heartbeat := HeartbeatInput{Timestamp: timestamp, Sign: payment.HeartbeatSignV2(timestamp, testMerchantKey)}
	if !validHeartbeatSign(heartbeat, testMerchantKey) {
		t.Fatal("v2 心跳签名应通过")
	}
	heartbeat.Sign = payment.LegacyHeartbeatSign(timestamp, testMerchantKey)
	if validHeartbeatSign(heartbeat, testMerchantKey) {
		t.Fatal("v1 心跳签名必须被拒绝")
	}
	if !usesLegacyHeartbeatSign(heartbeat, testMerchantKey) {
		t.Fatal("v1 心跳签名应被识别出来")
	}

	push := PaymentPushInput{Timestamp: timestamp, Type: "1", Price: "1.00"}
	push.Sign = payment.PushSignV2(push.Type, push.Price, push.Timestamp, testMerchantKey)
	if !validPushSign(push, testMerchantKey) {
		t.Fatal("v2 推送签名应通过")
	}
	push.Price = "9.99"
	if validPushSign(push, testMerchantKey) {
		t.Fatal("改写金额后的推送签名必须被拒绝")
	}
}

func TestParseMonitorTimestamp(t *testing.T) {
	moment, ok := parseMonitorTimestamp(" 1773500000000 ")
	if !ok {
		t.Fatal("毫秒时间戳应解析成功")
	}
	if moment.UnixMilli() != 1773500000000 {
		t.Fatalf("解析结果错误: %d", moment.UnixMilli())
	}

	for _, raw := range []string{"", "abc", "0", "-1"} {
		if _, ok := parseMonitorTimestamp(raw); ok {
			t.Errorf("%q 不应解析成功", raw)
		}
	}
}

func TestCheckTimestampFreshness(t *testing.T) {
	service := &MonitorService{signTTL: time.Minute}
	now := time.UnixMilli(1773500000000)

	if err := service.checkTimestampFreshness("1773500000000", now); err != nil {
		t.Fatalf("窗口内时间戳应通过: %v", err)
	}
	if err := service.checkTimestampFreshness("1773499970000", now); err != nil {
		t.Fatalf("窗口内的 30 秒偏移应通过: %v", err)
	}

	stale := []string{"1773499880000", "1773500120000"}
	for _, raw := range stale {
		err := service.checkTimestampFreshness(raw, now)
		if err == nil {
			t.Errorf("%s 超出窗口应被拒绝", raw)
			continue
		}
		if code, ok := ErrorCodeOf(err); !ok || code != CodeStaleTimestamp {
			t.Errorf("%s 应返回 %s，实际 %v", raw, CodeStaleTimestamp, err)
		}
	}
}