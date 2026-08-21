package mysql

import (
	"errors"
	"fmt"

	drivermysql "github.com/go-sql-driver/mysql"
	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/port"
	"gorm.io/gorm"
)

const (
	publicTokenColumn           = "public_token"
	publicTokenIndex            = "uq_orders_public_token"
	publicTokenBatchSize        = 200
	publicTokenRetryMax         = 8
	publicTokenInvalidCondition = "public_token IS NULL OR CHAR_LENGTH(public_token) <> 64 OR BINARY public_token REGEXP '[^0-9a-f]'"
)

// MigratePublicTokens 为已有订单补齐公开令牌。该过程可重复执行，适合在切换新 API 前离线运行。
func MigratePublicTokens(db *gorm.DB, generator port.PublicTokenGenerator) error {
	if db == nil {
		return errors.New("数据库连接不能为空")
	}
	if generator == nil {
		return errors.New("公开令牌生成器不能为空")
	}
	if !db.Migrator().HasTable("orders") {
		return errors.New("orders 表不存在，请先执行基础数据库初始化")
	}

	if err := ensurePublicTokenColumn(db); err != nil {
		return err
	}
	if err := backfillPublicTokens(db, generator); err != nil {
		return err
	}
	if err := ensurePublicTokenUniqueIndex(db); err != nil {
		return err
	}
	if err := makePublicTokenRequired(db); err != nil {
		return err
	}
	return verifyPublicTokens(db)
}

func ensurePublicTokenColumn(db *gorm.DB) error {
	var count int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = 'orders' AND column_name = ?`, publicTokenColumn).
		Scan(&count).Error; err != nil {
		return fmt.Errorf("检查 public_token 列失败: %w", err)
	}
	if count > 0 {
		return nil
	}
	if err := db.Exec("ALTER TABLE orders ADD COLUMN public_token CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL").Error; err != nil {
		return fmt.Errorf("添加 public_token 列失败: %w", err)
	}
	return nil
}

func backfillPublicTokens(db *gorm.DB, generator port.PublicTokenGenerator) error {
	query := fmt.Sprintf(`
		SELECT id
		FROM orders
		WHERE %s
		ORDER BY id
		LIMIT ?`, publicTokenInvalidCondition)
	for {
		var ids []int64
		err := db.Raw(query, publicTokenBatchSize).Scan(&ids).Error
		if err != nil {
			return fmt.Errorf("查询待回填订单失败: %w", err)
		}
		if len(ids) == 0 {
			return nil
		}

		for _, id := range ids {
			if err := backfillPublicTokenForOrder(db, generator, id); err != nil {
				return err
			}
		}
	}
}

func backfillPublicTokenForOrder(db *gorm.DB, generator port.PublicTokenGenerator, id int64) error {
	for attempt := 0; attempt < publicTokenRetryMax; attempt++ {
		token, err := generator.NewPublicToken()
		if err != nil {
			return fmt.Errorf("为订单 %d 生成 public_token 失败: %w", id, err)
		}
		if !order.IsValidPublicToken(token) {
			return fmt.Errorf("为订单 %d 生成了无效 public_token", id)
		}

		var existing int64
		if err := db.Raw("SELECT COUNT(*) FROM orders WHERE public_token = ? AND id <> ?", token, id).Scan(&existing).Error; err != nil {
			return fmt.Errorf("检查订单 %d 的 public_token 冲突失败: %w", id, err)
		}
		if existing > 0 {
			continue
		}

		result := db.Exec(fmt.Sprintf(`
			UPDATE orders
			SET public_token = ?
			WHERE id = ? AND (%s)`, publicTokenInvalidCondition), token, id)
		if result.Error == nil && result.RowsAffected == 1 {
			return nil
		}
		if result.Error == nil {
			return nil
		}
		if isDuplicateKey(result.Error) {
			continue
		}
		return fmt.Errorf("为订单 %d 写入 public_token 失败: %w", id, result.Error)
	}
	return fmt.Errorf("为订单 %d 生成唯一 public_token 超过重试次数", id)
}

func ensurePublicTokenUniqueIndex(db *gorm.DB) error {
	var total, valid int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = 'orders' AND index_name = ?`, publicTokenIndex).
		Scan(&total).Error; err != nil {
		return fmt.Errorf("检查 public_token 索引失败: %w", err)
	}
	if total > 0 {
		if err := db.Raw(`
			SELECT COUNT(*)
			FROM information_schema.statistics
			WHERE table_schema = DATABASE()
			  AND table_name = 'orders'
			  AND index_name = ?
			  AND non_unique = 0
			  AND column_name = ?
			  AND seq_in_index = 1
			  AND sub_part IS NULL`, publicTokenIndex, publicTokenColumn).
			Scan(&valid).Error; err != nil {
			return fmt.Errorf("检查 public_token 索引唯一性失败: %w", err)
		}
		if total == 1 && valid == 1 {
			return nil
		}
		return errors.New("public_token 索引名称已被不兼容索引占用，请先人工处理")
	}

	var duplicateCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM (
			SELECT public_token
			FROM orders
			GROUP BY public_token
			HAVING COUNT(*) > 1
		) AS duplicates`).Scan(&duplicateCount).Error; err != nil {
		return fmt.Errorf("检查 public_token 重复值失败: %w", err)
	}
	if duplicateCount > 0 {
		return errors.New("public_token 存在重复值，未创建唯一索引")
	}

	if err := db.Exec("ALTER TABLE orders ADD UNIQUE KEY uq_orders_public_token (public_token)").Error; err != nil {
		if isDuplicateKey(err) {
			return errors.New("public_token 存在重复值，未创建唯一索引")
		}
		return fmt.Errorf("创建 public_token 唯一索引失败: %w", err)
	}
	return nil
}

func makePublicTokenRequired(db *gorm.DB) error {
	if err := db.Exec("ALTER TABLE orders MODIFY COLUMN public_token CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL").Error; err != nil {
		return fmt.Errorf("将 public_token 设为非空失败: %w", err)
	}
	return nil
}

func verifyPublicTokens(db *gorm.DB) error {
	var invalid, duplicateCount int64
	if err := db.Raw(fmt.Sprintf(`
		SELECT COUNT(*)
		FROM orders
		WHERE %s`, publicTokenInvalidCondition)).Scan(&invalid).Error; err != nil {
		return fmt.Errorf("校验 public_token 失败: %w", err)
	}
	if invalid != 0 {
		return fmt.Errorf("仍有 %d 条订单的 public_token 无效", invalid)
	}
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM (
			SELECT public_token
			FROM orders
			GROUP BY public_token
			HAVING COUNT(*) > 1
		) AS duplicates`).Scan(&duplicateCount).Error; err != nil {
		return fmt.Errorf("校验 public_token 重复值失败: %w", err)
	}
	if duplicateCount != 0 {
		return fmt.Errorf("仍有 %d 组重复 public_token", duplicateCount)
	}
	return nil
}

func isDuplicateKey(err error) bool {
	var driverError *drivermysql.MySQLError
	return errors.As(err, &driverError) && driverError.Number == 1062
}
