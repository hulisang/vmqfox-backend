package mysql

import (
	"fmt"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DSN 使用驱动官方配置编码，避免凭据或连接参数中的特殊字符破坏连接串。
func DSN(user, password, host string, port int, database string, params map[string]string) string {
	if params == nil {
		params = make(map[string]string)
	}
	if _, exists := params["charset"]; !exists {
		params["charset"] = "utf8mb4"
	}

	return (&drivermysql.Config{
		User:                 user,
		Passwd:               password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", host, port),
		DBName:               database,
		Params:               params,
		ParseTime:            true,
		Loc:                  time.Local,
		AllowNativePasswords: true,
	}).FormatDSN()
}

// Open 打开数据库连接并设置连接池；不会执行 AutoMigrate 或任何建表操作。
func Open(dsn string, maxOpen, maxIdle int, maxLifetime time.Duration) (*gorm.DB, error) {
	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(maxLifetime)
	return db, nil
}
