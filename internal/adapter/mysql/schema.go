package mysql

import (
	_ "embed"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

//go:embed schema.sql
var schemaSQL string

// HasSchema 检查核心表 admin_credentials 是否存在
func HasSchema(db *gorm.DB) bool {
	return db.Migrator().HasTable("admin_credentials")
}

// InitSchema 执行完整的数据库初始化建表与初始配置
func InitSchema(db *gorm.DB) error {
	statements := splitSQLStatements(schemaSQL)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("执行建表 SQL 失败 [%s]: %w", stmt, err)
		}
	}
	return nil
}

// splitSQLStatements 拆分多条 SQL 语句并去除注释行
func splitSQLStatements(sqlContent string) []string {
	var statements []string
	var current strings.Builder
	lines := strings.Split(sqlContent, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			statements = append(statements, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 && strings.TrimSpace(current.String()) != "" {
		statements = append(statements, current.String())
	}
	return statements
}
