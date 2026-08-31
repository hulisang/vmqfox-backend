package app

import (
	"context"
	"errors"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/config"
	"github.com/hulisang/vmqfox-backend/internal/http/handler"
	"gorm.io/gorm"
)

const (
	applicationVersion     = "2.0"
	databaseVersionUnknown = "未知"
	databaseVersionTimeout = 2 * time.Second
)

type statusProvider struct {
	db        *gorm.DB
	mode      config.RuntimeMode
	startedAt time.Time
	modules   map[string]string
}

func newStatusProvider(db *gorm.DB, cfg config.Config) *statusProvider {
	return &statusProvider{
		db:        db,
		mode:      cfg.Runtime.Mode,
		startedAt: time.Now(),
		modules:   moduleVersions(),
	}
}

func (s *statusProvider) Health(context.Context) error { return nil }

func (s *statusProvider) Ready(ctx context.Context) error {
	if s.db == nil {
		return errors.New("数据库连接未初始化")
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (s *statusProvider) Environment(ctx context.Context) handler.Environment {
	return handler.Environment{
		Service:         "vmqfox-backend",
		Version:         applicationVersion,
		RuntimeMode:     string(s.mode),
		GoVersion:       runtime.Version(),
		Platform:        runtime.GOOS + "/" + runtime.GOARCH,
		ServerEngine:    "Go net/http",
		WebFramework:    moduleLabel(s.modules, "github.com/gin-gonic/gin", "Gin"),
		QRCodeLibrary:   moduleLabel(s.modules, "github.com/piglig/go-qr", "go-qr"),
		DatabaseVersion: s.databaseVersion(ctx),
		Uptime:          time.Since(s.startedAt),
	}
}

// databaseVersion 只做只读探测；探测失败不影响仪表盘其余运行时信息。
func (s *statusProvider) databaseVersion(ctx context.Context) string {
	if s.db == nil {
		return databaseVersionUnknown
	}
	queryContext, cancel := context.WithTimeout(ctx, databaseVersionTimeout)
	defer cancel()

	var version string
	if err := s.db.WithContext(queryContext).Raw("SELECT VERSION()").Scan(&version).Error; err != nil {
		return databaseVersionUnknown
	}
	if strings.TrimSpace(version) == "" {
		return databaseVersionUnknown
	}
	return version
}

// moduleVersions 读取构建信息，避免把第三方版本号写死在源码里造成事实漂移。
func moduleVersions() map[string]string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}
	versions := make(map[string]string, len(info.Deps))
	for _, dependency := range info.Deps {
		if dependency == nil {
			continue
		}
		versions[dependency.Path] = dependency.Version
	}
	return versions
}

func moduleLabel(versions map[string]string, path, name string) string {
	if version := versions[path]; version != "" {
		return name + " " + version
	}
	return name
}
