package mysql

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/port"
	"gorm.io/gorm"
)

// legacyOrdersDDL 复刻切换前的订单表结构：没有 public_token 列，也没有对应唯一索引。
const legacyOrdersDDL = `
CREATE TABLE orders (
  id BIGINT NOT NULL AUTO_INCREMENT,
  order_id VARBINARY(128) NOT NULL,
  pay_id VARBINARY(255) NOT NULL,
  payment_type TINYINT NOT NULL,
  price_cent BIGINT NOT NULL,
  really_price_cent BIGINT NOT NULL,
  price_text VARCHAR(64) NOT NULL,
  really_price_text VARCHAR(64) NOT NULL,
  state TINYINT NOT NULL DEFAULT 0,
  param VARCHAR(255) NOT NULL DEFAULT '',
  pay_url TEXT NOT NULL,
  is_auto TINYINT(1) NOT NULL DEFAULT 0,
  notify_url TEXT NOT NULL,
  return_url TEXT NOT NULL,
  created_at DATETIME(6) NOT NULL,
  paid_at DATETIME(6) NULL,
  closed_at DATETIME(6) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_orders_order_id (order_id),
  UNIQUE KEY uq_orders_pay_id (pay_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

// stubTokenGenerator 让迁移测试可以精确控制令牌序列，从而覆盖冲突重试与生成失败分支。
type stubTokenGenerator struct {
	scripted []string
	err      error
	calls    int
	counter  int
}

func (g *stubTokenGenerator) NewPublicToken() (string, error) {
	g.calls++
	if g.err != nil {
		return "", g.err
	}
	if len(g.scripted) > 0 {
		token := g.scripted[0]
		g.scripted = g.scripted[1:]
		return token, nil
	}
	g.counter++
	return hexToken(g.counter), nil
}

var _ port.PublicTokenGenerator = (*stubTokenGenerator)(nil)

// hexToken 生成固定长度的小写十六进制令牌，便于断言而不依赖随机性。
func hexToken(seed int) string {
	return fmt.Sprintf("%064x", seed)
}

// openTestDatabase 只在显式提供测试库 DSN 时运行迁移集成测试，避免误连生产库。
func openTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("VMQ_TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("未设置 VMQ_TEST_MYSQL_DSN，跳过 public_token 迁移集成测试")
	}
	db, err := Open(dsn, 4, 2, time.Minute)
	if err != nil {
		t.Fatalf("连接测试数据库失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func resetLegacyOrders(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("DROP TABLE IF EXISTS orders").Error; err != nil {
		t.Fatalf("清理测试订单表失败: %v", err)
	}
	if err := db.Exec(legacyOrdersDDL).Error; err != nil {
		t.Fatalf("创建旧版订单表失败: %v", err)
	}
}

func insertLegacyOrders(t *testing.T, db *gorm.DB, count int) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	for index := 1; index <= count; index++ {
		identifier := fmt.Sprintf("legacy-%06d", index)
		err := db.Exec(`
			INSERT INTO orders
				(order_id, pay_id, payment_type, price_cent, really_price_cent,
				 price_text, really_price_text, state, param, pay_url, is_auto,
				 notify_url, return_url, created_at)
			VALUES (?, ?, 2, 100, 100, '1.00', '1.00', 0, '', '', 0, '', '', ?)`,
			identifier, identifier, now).Error
		if err != nil {
			t.Fatalf("写入旧版订单 %s 失败: %v", identifier, err)
		}
	}
}

func addNullablePublicTokenColumn(t *testing.T, db *gorm.DB) {
	t.Helper()
	err := db.Exec("ALTER TABLE orders ADD COLUMN public_token CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL").Error
	if err != nil {
		t.Fatalf("手动添加 public_token 列失败: %v", err)
	}
}

func publicTokenNullable(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var nullable string
	err := db.Raw(`
		SELECT IS_NULLABLE
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = 'orders' AND column_name = 'public_token'`).
		Scan(&nullable).Error
	if err != nil {
		t.Fatalf("读取 public_token 列约束失败: %v", err)
	}
	return nullable
}

func uniqueIndexCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		  AND table_name = 'orders'
		  AND index_name = ?
		  AND non_unique = 0
		  AND column_name = 'public_token'`, publicTokenIndex).
		Scan(&count).Error
	if err != nil {
		t.Fatalf("读取 public_token 索引信息失败: %v", err)
	}
	return count
}

func storedTokens(t *testing.T, db *gorm.DB) map[int64]string {
	t.Helper()
	var rows []struct {
		ID          int64   `gorm:"column:id"`
		PublicToken *string `gorm:"column:public_token"`
	}
	if err := db.Raw("SELECT id, public_token FROM orders ORDER BY id").Scan(&rows).Error; err != nil {
		t.Fatalf("读取订单令牌失败: %v", err)
	}
	result := make(map[int64]string, len(rows))
	for _, row := range rows {
		if row.PublicToken == nil {
			result[row.ID] = ""
			continue
		}
		result[row.ID] = *row.PublicToken
	}
	return result
}

func assertTokensValidAndUnique(t *testing.T, tokens map[int64]string, expectedCount int) {
	t.Helper()
	if len(tokens) != expectedCount {
		t.Fatalf("订单数量应为 %d，实际 %d", expectedCount, len(tokens))
	}
	seen := make(map[string]int64, len(tokens))
	for id, token := range tokens {
		if !order.IsValidPublicToken(token) {
			t.Fatalf("订单 %d 的 public_token 无效: %q", id, token)
		}
		if previous, exists := seen[token]; exists {
			t.Fatalf("订单 %d 与 %d 的 public_token 重复", id, previous)
		}
		seen[token] = id
	}
}

// TestMigratePublicTokensRejectsInvalidInput 锁定迁移入口的前置校验，避免在缺表或缺依赖时产生半迁移状态。
func TestMigratePublicTokensRejectsInvalidInput(t *testing.T) {
	db := openTestDatabase(t)

	if err := MigratePublicTokens(nil, &stubTokenGenerator{}); err == nil {
		t.Fatal("数据库为空时应直接失败")
	}
	if err := MigratePublicTokens(db, nil); err == nil {
		t.Fatal("令牌生成器为空时应直接失败")
	}

	if err := db.Exec("DROP TABLE IF EXISTS orders").Error; err != nil {
		t.Fatalf("清理测试订单表失败: %v", err)
	}
	err := MigratePublicTokens(db, &stubTokenGenerator{})
	if err == nil || !strings.Contains(err.Error(), "orders 表不存在") {
		t.Fatalf("缺少 orders 表时应提示先初始化数据库，实际 err=%v", err)
	}
}

// TestMigratePublicTokensOnEmptyTable 覆盖空表场景：加列、建唯一索引并收紧为非空。
func TestMigratePublicTokensOnEmptyTable(t *testing.T) {
	db := openTestDatabase(t)
	resetLegacyOrders(t, db)

	generator := &stubTokenGenerator{}
	if err := MigratePublicTokens(db, generator); err != nil {
		t.Fatalf("空表迁移应成功，实际 err=%v", err)
	}
	if generator.calls != 0 {
		t.Fatalf("空表不应生成令牌，实际调用 %d 次", generator.calls)
	}
	if nullable := publicTokenNullable(t, db); nullable != "NO" {
		t.Fatalf("public_token 应为 NOT NULL，实际 IS_NULLABLE=%s", nullable)
	}
	if count := uniqueIndexCount(t, db); count != 1 {
		t.Fatalf("应创建唯一索引 %s，实际匹配 %d 条", publicTokenIndex, count)
	}
}

// TestMigratePublicTokensBackfillsAndIsRepeatable 覆盖既有订单分批回填与重复执行幂等。
func TestMigratePublicTokensBackfillsAndIsRepeatable(t *testing.T) {
	db := openTestDatabase(t)
	resetLegacyOrders(t, db)
	// 超过单批 200 条，确保回填循环真正跨批次执行。
	const orderCount = publicTokenBatchSize + 5
	insertLegacyOrders(t, db, orderCount)

	generator := &stubTokenGenerator{}
	if err := MigratePublicTokens(db, generator); err != nil {
		t.Fatalf("回填迁移应成功，实际 err=%v", err)
	}
	if generator.calls != orderCount {
		t.Fatalf("应为每条订单生成一次令牌，期望 %d 次实际 %d 次", orderCount, generator.calls)
	}

	first := storedTokens(t, db)
	assertTokensValidAndUnique(t, first, orderCount)
	if nullable := publicTokenNullable(t, db); nullable != "NO" {
		t.Fatalf("回填后 public_token 应为 NOT NULL，实际 IS_NULLABLE=%s", nullable)
	}
	if count := uniqueIndexCount(t, db); count != 1 {
		t.Fatalf("回填后应存在唯一索引，实际匹配 %d 条", count)
	}

	repeat := &stubTokenGenerator{}
	if err := MigratePublicTokens(db, repeat); err != nil {
		t.Fatalf("重复执行迁移应成功，实际 err=%v", err)
	}
	if repeat.calls != 0 {
		t.Fatalf("重复执行不应再生成令牌，实际调用 %d 次", repeat.calls)
	}
	if second := storedTokens(t, db); fmt.Sprint(second) != fmt.Sprint(first) {
		t.Fatal("重复执行迁移不应改写既有令牌")
	}
	if count := uniqueIndexCount(t, db); count != 1 {
		t.Fatalf("重复执行后唯一索引应保持一份，实际匹配 %d 条", count)
	}
}

// TestMigratePublicTokensRetriesGeneratorCollision 覆盖生成器偶发重复时的回填重试。
func TestMigratePublicTokensRetriesGeneratorCollision(t *testing.T) {
	db := openTestDatabase(t)
	resetLegacyOrders(t, db)
	insertLegacyOrders(t, db, 2)

	collision := hexToken(0xC0FFEE)
	generator := &stubTokenGenerator{scripted: []string{collision, collision}}
	if err := MigratePublicTokens(db, generator); err != nil {
		t.Fatalf("生成器重复时应重试并成功，实际 err=%v", err)
	}
	if generator.calls != 3 {
		t.Fatalf("重复令牌应触发一次重试，期望 3 次调用实际 %d 次", generator.calls)
	}
	assertTokensValidAndUnique(t, storedTokens(t, db), 2)
}

// TestMigratePublicTokensStopsOnDuplicateTokens 覆盖脏数据场景：存在重复令牌时不得建唯一索引也不得收紧为非空。
func TestMigratePublicTokensStopsOnDuplicateTokens(t *testing.T) {
	db := openTestDatabase(t)
	resetLegacyOrders(t, db)
	insertLegacyOrders(t, db, 2)
	addNullablePublicTokenColumn(t, db)

	duplicate := hexToken(0xDEAD)
	if err := db.Exec("UPDATE orders SET public_token = ?", duplicate).Error; err != nil {
		t.Fatalf("构造重复令牌失败: %v", err)
	}

	err := MigratePublicTokens(db, &stubTokenGenerator{})
	if err == nil || !strings.Contains(err.Error(), "存在重复值") {
		t.Fatalf("存在重复令牌时应报错终止，实际 err=%v", err)
	}
	if nullable := publicTokenNullable(t, db); nullable != "YES" {
		t.Fatalf("迁移失败后不应把列改为非空，实际 IS_NULLABLE=%s", nullable)
	}
	if count := uniqueIndexCount(t, db); count != 0 {
		t.Fatalf("迁移失败时不应创建唯一索引，实际匹配 %d 条", count)
	}
}

// TestMigratePublicTokensRejectsIncompatibleIndexName 覆盖索引名被非唯一索引占用时的人工介入提示。
func TestMigratePublicTokensRejectsIncompatibleIndexName(t *testing.T) {
	db := openTestDatabase(t)
	resetLegacyOrders(t, db)
	addNullablePublicTokenColumn(t, db)
	if err := db.Exec("ALTER TABLE orders ADD KEY uq_orders_public_token (public_token)").Error; err != nil {
		t.Fatalf("构造非唯一同名索引失败: %v", err)
	}

	err := MigratePublicTokens(db, &stubTokenGenerator{})
	if err == nil || !strings.Contains(err.Error(), "不兼容索引") {
		t.Fatalf("同名非唯一索引应要求人工处理，实际 err=%v", err)
	}
	if nullable := publicTokenNullable(t, db); nullable != "YES" {
		t.Fatalf("迁移中止后列约束不应被修改，实际 IS_NULLABLE=%s", nullable)
	}
}

// TestMigratePublicTokensPropagatesGeneratorFailure 覆盖生成器故障：迁移中止且不留下非空约束或部分令牌。
func TestMigratePublicTokensPropagatesGeneratorFailure(t *testing.T) {
	db := openTestDatabase(t)
	resetLegacyOrders(t, db)
	insertLegacyOrders(t, db, 3)

	generatorErr := errors.New("随机源不可用")
	err := MigratePublicTokens(db, &stubTokenGenerator{err: generatorErr})
	if !errors.Is(err, generatorErr) {
		t.Fatalf("生成器故障应向上传递，实际 err=%v", err)
	}
	if nullable := publicTokenNullable(t, db); nullable != "YES" {
		t.Fatalf("生成失败后列必须保持可空，实际 IS_NULLABLE=%s", nullable)
	}
	if count := uniqueIndexCount(t, db); count != 0 {
		t.Fatalf("生成失败后不应创建唯一索引，实际匹配 %d 条", count)
	}
	for id, token := range storedTokens(t, db) {
		if token != "" {
			t.Fatalf("生成失败后订单 %d 不应写入令牌，实际 %q", id, token)
		}
	}
}

// TestMigratePublicTokensRejectsInvalidGeneratedToken 覆盖生成器返回格式非法令牌时的拒绝写入。
func TestMigratePublicTokensRejectsInvalidGeneratedToken(t *testing.T) {
	db := openTestDatabase(t)
	resetLegacyOrders(t, db)
	insertLegacyOrders(t, db, 1)

	err := MigratePublicTokens(db, &stubTokenGenerator{scripted: []string{"NOT-A-VALID-TOKEN"}})
	if err == nil || !strings.Contains(err.Error(), "无效 public_token") {
		t.Fatalf("非法令牌应被拒绝，实际 err=%v", err)
	}
	for id, token := range storedTokens(t, db) {
		if token != "" {
			t.Fatalf("订单 %d 不应写入非法令牌，实际 %q", id, token)
		}
	}
}
