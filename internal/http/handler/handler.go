package handler

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/auth"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
)

const notImplementedMessage = "接口尚未实现，当前 Go 服务未接管该业务"

type TokenParser interface {
	ParseAuthorization(value string) (auth.Identity, error)
}

type StatusProvider interface {
	Health(context.Context) error
	Ready(context.Context) error
	Environment(context.Context) Environment
}

// Environment 只描述运行时事实，字段命名由各协议的 handler 负责映射。
type Environment struct {
	Service         string
	Version         string
	RuntimeMode     string
	GoVersion       string
	Platform        string
	ServerEngine    string
	WebFramework    string
	QRCodeLibrary   string
	DatabaseVersion string
	Uptime          time.Duration
}

type Dependencies struct {
	Auth         Authenticator
	Settings     SettingsManager
	Orders       OrderManager
	Monitor      MonitorManager
	QRCodes      QRCodeManager
	QRCodeImages QRCodeImageManager
	TokenParser  TokenParser
	Status       StatusProvider
}

type ProtocolStyle uint8

const (
	ProtocolAPI ProtocolStyle = iota
	ProtocolLegacy
	ProtocolLayui
	ProtocolRaw
)

type Handlers struct {
	Health         gin.HandlerFunc
	Ready          gin.HandlerFunc
	Root           gin.HandlerFunc
	Login          gin.HandlerFunc
	Logout         gin.HandlerFunc
	LegacyProbe    gin.HandlerFunc
	GetSettings    gin.HandlerFunc
	GetMonitor     gin.HandlerFunc
	UpdateSettings gin.HandlerFunc
	UpdateMonitor  gin.HandlerFunc
	SystemConfig   gin.HandlerFunc
	UserInfo       gin.HandlerFunc
	UserList       gin.HandlerFunc
	Menu           gin.HandlerFunc
	Orders         OrderHandlers
	Monitor        MonitorHandlers
	QRCodes        QRCodeHandlers
	APIError       gin.HandlerFunc
	LegacyError    gin.HandlerFunc
	LayuiError     gin.HandlerFunc
	RawError       gin.HandlerFunc
}

func New(deps Dependencies) Handlers {
	status := deps.Status
	if status == nil {
		status = staticStatus{}
	}

	return Handlers{
		Health:         healthHandler(status),
		Ready:          readinessHandler(status),
		Root:           rootHandler(status),
		Login:          loginHandler(deps.Auth),
		Logout:         logoutHandler(),
		LegacyProbe:    legacyProbeHandler(),
		GetSettings:    getSettingsHandler(deps.Settings),
		GetMonitor:     getMonitorSettingsHandler(deps.Settings),
		UpdateSettings: updateSettingsHandler(deps.Settings),
		UpdateMonitor:  updateMonitorSettingsHandler(deps.Settings),
		SystemConfig:   systemConfigHandler(status),
		UserInfo:       userInfoHandler(deps.Settings),
		UserList:       userListHandler(deps.Settings),
		Menu:           menuHandler(),
		Orders:         newOrderHandlers(deps.Orders),
		Monitor:        newMonitorHandlers(deps.Monitor),
		QRCodes:        newQRCodeHandlers(deps.QRCodeImages, deps.QRCodes),
		APIError:       Unimplemented(ProtocolAPI),
		LegacyError:    Unimplemented(ProtocolLegacy),
		LayuiError:     Unimplemented(ProtocolLayui),
		RawError:       Unimplemented(ProtocolRaw),
	}
}

func healthHandler(status StatusProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := status.Health(c.Request.Context()); err != nil {
			c.String(http.StatusServiceUnavailable, "unhealthy\n")
			return
		}
		c.String(http.StatusOK, "healthy\n")
	}
}

func readinessHandler(status StatusProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := status.Ready(c.Request.Context()); err != nil {
			c.String(http.StatusServiceUnavailable, "not ready\n")
			return
		}
		c.String(http.StatusOK, "ready\n")
	}
}

func rootHandler(status StatusProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		environment := status.Environment(c.Request.Context())
		c.JSON(http.StatusOK, php.NewEnvelope(1, "成功", gin.H{
			"service":     environment.Service,
			"version":     environment.Version,
			"status":      "running",
			"runtimeMode": environment.RuntimeMode,
			"runtime":     environment.GoVersion,
		}))
	}
}

// systemConfigHandler 保留仪表盘历史字段名，值改为真实的 Go 运行时事实。
func systemConfigHandler(status StatusProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		environment := status.Environment(c.Request.Context())
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{
			"phpOs":           environment.Platform,
			"server":          environment.ServerEngine,
			"phpVersion":      environment.GoVersion,
			"mysqlVersion":    environment.DatabaseVersion,
			"thinkphpVersion": environment.WebFramework,
			"gdInfo":          environment.QRCodeLibrary,
			"appVersion":      environment.Version,
			"runTime":         formatUptime(environment.Uptime),
		}))
	}
}

func formatUptime(value time.Duration) string {
	if value < time.Minute {
		return "不足1分钟"
	}
	minutes := int(value.Minutes())
	days, minutes := minutes/(60*24), minutes%(60*24)
	hours, minutes := minutes/60, minutes%60

	var builder strings.Builder
	if days > 0 {
		fmt.Fprintf(&builder, "%d天", days)
	}
	if hours > 0 {
		fmt.Fprintf(&builder, "%d小时", hours)
	}
	if minutes > 0 {
		fmt.Fprintf(&builder, "%d分钟", minutes)
	}
	return builder.String()
}

func legacyProbeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, php.NewEnvelope(1, "成功", nil))
	}
}

func Unimplemented(style ProtocolStyle) gin.HandlerFunc {
	return func(c *gin.Context) {
		const status = http.StatusNotImplemented
		switch style {
		case ProtocolLegacy:
			c.JSON(status, php.NewEnvelope(-1, notImplementedMessage, nil))
		case ProtocolLayui:
			c.JSON(status, gin.H{
				"code":  -1,
				"msg":   notImplementedMessage,
				"count": 0,
				"data":  nil,
			})
		case ProtocolRaw:
			c.String(status, notImplementedMessage+"\n")
		default:
			c.JSON(status, php.NewEnvelope(501, notImplementedMessage, nil))
		}
	}
}

type staticStatus struct{}

func (staticStatus) Health(context.Context) error { return nil }
func (staticStatus) Ready(context.Context) error  { return nil }

func (staticStatus) Environment(context.Context) Environment {
	return Environment{
		Service:         "vmqfox-backend",
		Version:         "2.0",
		GoVersion:       runtime.Version(),
		Platform:        runtime.GOOS + "/" + runtime.GOARCH,
		ServerEngine:    "Go net/http",
		WebFramework:    "Gin",
		QRCodeLibrary:   "go-qr",
		DatabaseVersion: "未知",
	}
}
