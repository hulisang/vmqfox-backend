package main

import (
	"bufio"
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
	"strings"
	"syscall"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/adapter/mysql"
	"github.com/hulisang/vmqfox-backend/internal/adapter/system"
	"github.com/hulisang/vmqfox-backend/internal/app"
	"github.com/hulisang/vmqfox-backend/internal/auth"
	"github.com/hulisang/vmqfox-backend/internal/config"
	"github.com/hulisang/vmqfox-backend/internal/domain/admin"
	"github.com/hulisang/vmqfox-backend/internal/port"
	"golang.org/x/term"
	"gorm.io/gorm"
)

const maxPasswordInputBytes = 74

func main() {
	hashPassword := flag.Bool("hash-password", false, "从标准输入读取明文密码并输出它的 bcrypt 加密哈希后退出")
	initDB := flag.Bool("init-db", false, "初始化数据库表结构（执行建表与初始配置）")
	migratePublicTokens := flag.Bool("migrate-public-tokens", false, "为订单补齐 public_token 并创建唯一约束")
	initAdmin := flag.Bool("init-admin", false, "初始化或重置管理员账号与密码（自动检测并建表，支持交互式密文输入）")
	resetAdmin := flag.Bool("reset-admin", false, "初始化或重置管理员账号与密码（-init-admin 的别名）")
	adminUser := flag.String("username", "", "管理员用户名（配合 -init-admin 使用）")
	adminPass := flag.String("password", "", "管理员明文密码（配合 -init-admin 使用，留空则进入交互式密文输入）")
	forceAdmin := flag.Bool("force", false, "若已存在管理员则强制覆盖，跳过确认提示（配合 -init-admin 使用）")
	flag.Parse()

	if *initDB {
		if err := runInitDB(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 数据库初始化失败: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *migratePublicTokens {
		if err := runPublicTokenMigration(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ public_token 迁移失败: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *initAdmin || *resetAdmin {
		opts := AdminInitOptions{
			Username: *adminUser,
			Password: *adminPass,
			Force:    *forceAdmin,
		}
		if err := runAdminInit(context.Background(), opts); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 管理员初始化失败: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

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

// AdminInitOptions 封装管理员初始化/重置命令参数
type AdminInitOptions struct {
	Username string
	Password string
	Force    bool
}

// runInitDB 执行独立的数据表结构初始化
func runInitDB() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
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
		return fmt.Errorf("数据库连接失败 (请检查 .env 配置及数据库运行状态): %w", err)
	}
	defer func() { _ = closeDatabase(db) }()

	fmt.Println("📦 正在初始化数据库表结构...")
	if err := mysql.InitSchema(db); err != nil {
		return fmt.Errorf("初始化数据库表结构失败: %w", err)
	}
	fmt.Println("✅ 数据库表结构初始化成功！")
	return nil
}

// runPublicTokenMigration 在切换公开接口前完成令牌列、回填和唯一约束迁移。
func runPublicTokenMigration() error {
	database, err := config.LoadDatabase()
	if err != nil {
		return fmt.Errorf("加载数据库配置失败: %w", err)
	}
	dsn := mysql.DSN(
		database.User,
		database.Password,
		database.Host,
		database.Port,
		database.Name,
		map[string]string{"charset": database.Charset},
	)
	db, err := mysql.Open(dsn, database.MaxOpenConns, database.MaxIdleConns, database.ConnMaxLifetime)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}
	defer func() { _ = closeDatabase(db) }()

	fmt.Println("🔐 正在迁移订单 public_token...")
	if err := mysql.MigratePublicTokens(db, system.PublicTokenGenerator{}); err != nil {
		return err
	}
	fmt.Println("✅ 订单 public_token 迁移完成，所有令牌已校验为唯一且非空。")
	return nil
}

// runAdminInit 执行管理员初始化/重置逻辑
func runAdminInit(ctx context.Context, opts AdminInitOptions) error {
	// 1. 加载数据库与系统配置
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 2. 连接 MySQL 数据库
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
		return fmt.Errorf("数据库连接失败 (请检查 .env 配置及数据库运行状态): %w", err)
	}
	defer func() { _ = closeDatabase(db) }()

	isTTY := term.IsTerminal(int(os.Stdin.Fd()))
	stdinReader := bufio.NewReader(os.Stdin)

	// 3. 检查数据库表是否存在，若尚未初始化则自动建表
	if !mysql.HasSchema(db) {
		if isTTY {
			fmt.Println("📦 检测到数据库尚未初始化，正在自动创建数据表...")
		}
		if err := mysql.InitSchema(db); err != nil {
			return fmt.Errorf("自动初始化数据库表结构失败: %w", err)
		}
		if isTTY {
			fmt.Println("✅ 数据库基础表结构创建成功！")
		}
	}

	repo := mysql.NewAdminCredentialRepository(db)

	// 4. 检查数据库中是否已存在管理员账户
	existing, err := repo.Get(ctx)
	hasExisting := err == nil
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return fmt.Errorf("查询已有管理员记录失败: %w", err)
	}

	if hasExisting && !opts.Force {
		if !isTTY {
			return fmt.Errorf("已存在管理员账户(%s)，在非交互模式下请指定 -force 参数以强制覆盖", existing.Username)
		}
		fmt.Printf("⚠️  检测到已存在管理员账户 (当前用户名: %s)。是否确认覆盖重置？[y/N]: ", existing.Username)
		confirmText, err := stdinReader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("读取确认输入失败: %w", err)
		}
		confirmText = strings.TrimSpace(confirmText)
		if confirmText != "y" && confirmText != "Y" {
			fmt.Println("操作已取消，未对现有管理员数据进行任何修改。")
			return nil
		}
	}

	// 4. 解析或交互输入管理员用户名
	username := strings.TrimSpace(opts.Username)
	if username == "" {
		defaultUser := "admin"
		if hasExisting && existing.Username != "" {
			defaultUser = existing.Username
		}
		if isTTY {
			fmt.Printf("请输入管理员用户名 [默认: %s]: ", defaultUser)
			inputUser, err := stdinReader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("读取用户名输入失败: %w", err)
			}
			inputUser = strings.TrimSpace(inputUser)
			if inputUser != "" {
				username = inputUser
			} else {
				username = defaultUser
			}
		} else {
			username = defaultUser
		}
	}

	username = admin.NormalizeUsername(username)
	if username == "" || len([]byte(username)) > 128 {
		return errors.New("管理员用户名不能为空且长度不能超过 128 个字节")
	}

	// 5. 解析或交互输入管理员密码（密文隐藏与二次确认）
	password := opts.Password
	if password == "" {
		if isTTY {
			for {
				fmt.Print("请输入管理员密码 (8~72位): ")
				pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err != nil {
					return fmt.Errorf("读取密码输入失败: %w", err)
				}
				pw := string(pwBytes)
				if len([]byte(pw)) < 8 || len([]byte(pw)) > 72 {
					fmt.Println("❌ 密码长度必须在 8 到 72 个字节之间，请重新输入。")
					continue
				}

				fmt.Print("请再次输入管理员密码以确认: ")
				confirmBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err != nil {
					return fmt.Errorf("读取密码确认输入失败: %w", err)
				}
				if pw != string(confirmBytes) {
					fmt.Println("❌ 两次输入的密码不一致，请重新输入。")
					continue
				}
				password = pw
				break
			}
		} else {
			// 非 TTY 环境尝试从标准输入管道读取单行密码
			pw, err := readPassword(os.Stdin)
			if err != nil {
				return fmt.Errorf("从标准输入读取密码失败: %w (提示: 可使用 -password 参数传入明文密码)", err)
			}
			password = pw
		}
	}

	if len([]byte(password)) < 8 || len([]byte(password)) > 72 {
		return auth.ErrInvalidPasswordLength
	}

	// 6. 计算 Bcrypt 密码哈希
	hasher := auth.NewBcryptPasswordHasher()
	passwordHash, err := hasher.Hash(password)
	if err != nil {
		return fmt.Errorf("生成密码哈希失败: %w", err)
	}

	// 7. 幂等保存管理员凭据
	credential := admin.Credential{
		ID:           admin.SingletonID,
		Username:     username,
		PasswordHash: passwordHash,
		Enabled:      true,
		UpdatedAt:    time.Now().UTC(),
	}
	if err := repo.Save(ctx, credential); err != nil {
		return fmt.Errorf("保存管理员数据失败: %w", err)
	}

	// 8. 友好输出成功信息
	fmt.Println()
	fmt.Println("==================================================")
	if hasExisting {
		fmt.Println("🎉 管理员账户重置成功！")
	} else {
		fmt.Println("🎉 管理员账户初始化成功！")
	}
	fmt.Printf("👤 管理员用户名: %s\n", username)
	fmt.Println("🔑 管理员密码:   [已通过 Bcrypt 加密保存至数据库]")
	fmt.Println("💡 现在您可以使用此账号登录管理后台。")
	fmt.Println("==================================================")
	fmt.Println()
	return nil
}
