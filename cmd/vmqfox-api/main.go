package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hulisang/vmqfox-backend/internal/adapter/mysql"
	"github.com/hulisang/vmqfox-backend/internal/app"
	"github.com/hulisang/vmqfox-backend/internal/auth"
	"github.com/hulisang/vmqfox-backend/internal/config"
	"gorm.io/gorm"
)

const maxPasswordInputBytes = 74

func main() {
	hashPassword := flag.Bool("hash-password", false, "从标准输入读取明文密码并输出它的 bcrypt 加密哈希后退出")
	flag.Parse()

	if *hashPassword {
		password, err := readPassword(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "生成管理员密码哈希失败: %v\n", err)
			os.Exit(1)
		}
		passwordHash, err := auth.NewBcryptPasswordHasher().Hash(password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "生成管理员密码哈希失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, passwordHash)
		os.Exit(0)
	}

	logger := log.New(os.Stderr, "vmqfox-api ", log.LstdFlags|log.LUTC)
	cfg, err := config.Load()
	if err != nil {
		logger.Printf("配置加载失败: %v", err)
		os.Exit(1)
	}

	tokenService, err := auth.NewTokenService(cfg.Token)
	if err != nil {
		logger.Printf("Token 配置失败: %v", err)
		os.Exit(1)
	}

	dsn := mysql.DSN(
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		map[string]string{"charset": cfg.Database.Charset},
	)
	db, err := mysql.Open(dsn, cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns, cfg.Database.ConnMaxLifetime)
	if err != nil {
		logger.Printf("数据库连接初始化失败: %v", err)
		os.Exit(1)
	}

	application, err := app.New(cfg, db, tokenService, logger)
	if err != nil {
		logger.Printf("应用组装失败: %v", err)
		_ = closeDatabase(db)
		os.Exit(1)
	}

	shutdownContext, stopSignal := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignal()
	application.Start(shutdownContext)

	serverErrors := make(chan error, 1)
	go func() {
		logger.Printf("HTTP 服务监听 %s，运行模式=%s", cfg.Server.Address(), cfg.Runtime.Mode)
		serverErrors <- application.Server().ListenAndServe()
	}()

	select {
	case err = <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Printf("HTTP 服务停止: %v", err)
			ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
			if closeErr := application.Close(ctx); closeErr != nil {
				logger.Printf("应用关闭失败: %v", closeErr)
			}
			cancel()
			os.Exit(1)
		}
	case <-shutdownContext.Done():
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		if err = application.Close(ctx); err != nil {
			logger.Printf("应用优雅停机失败: %v", err)
			os.Exit(1)
		}
	}
}

func closeDatabase(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func readPassword(input *os.File) (string, error) {
	info, err := input.Stat()
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return "", errors.New("拒绝从交互终端读取，请通过标准输入管道传入密码")
	}

	raw, err := io.ReadAll(io.LimitReader(input, maxPasswordInputBytes+1))
	if err != nil {
		return "", err
	}
	if len(raw) > maxPasswordInputBytes {
		return "", auth.ErrInvalidPasswordLength
	}
	raw = bytes.TrimSuffix(raw, []byte{'\n'})
	raw = bytes.TrimSuffix(raw, []byte{'\r'})
	if bytes.ContainsAny(raw, "\r\n") {
		return "", errors.New("标准输入只能包含一行密码")
	}
	return string(raw), nil
}
