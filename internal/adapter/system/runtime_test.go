package system

import (
	"testing"

	"github.com/hulisang/vmqfox-backend/internal/domain/order"
)

// TestPublicTokenGenerator 确保公开订单令牌使用固定长度的小写十六进制随机值。
func TestPublicTokenGenerator(t *testing.T) {
	generator := PublicTokenGenerator{}
	seen := make(map[string]struct{})

	for range 16 {
		token, err := generator.NewPublicToken()
		if err != nil {
			t.Fatalf("生成公开订单令牌失败: %v", err)
		}
		if !order.IsValidPublicToken(token) {
			t.Fatalf("生成的公开订单令牌格式无效: %q", token)
		}
		if _, exists := seen[token]; exists {
			t.Fatalf("重复生成了公开订单令牌: %q", token)
		}
		seen[token] = struct{}{}
	}
}
