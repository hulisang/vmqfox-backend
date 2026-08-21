package http

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/config"
	"github.com/hulisang/vmqfox-backend/internal/http/handler"
	"github.com/hulisang/vmqfox-backend/internal/http/middleware"
)

type RouterDeps struct {
	Router      *gin.Engine
	Handlers    handler.Handlers
	TokenParser middleware.TokenParser
	Origin      string
	RuntimeMode middleware.RuntimeMode
	RateLimit   config.RateLimitConfig
	Logger      *log.Logger
}

// NewRouter 只登记已确认的路径；未实现业务路径返回明确协议错误，不伪造成功。
func NewRouter(deps RouterDeps) *gin.Engine {
	r := deps.Router
	if r == nil {
		r = gin.New()
	}
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	if deps.RateLimit.Login.Limit < 1 {
		defaults := config.DefaultRateLimitConfig()
		defaults.TrustedCIDRs = deps.RateLimit.TrustedCIDRs
		defaults.TrustedIPs = deps.RateLimit.TrustedIPs
		deps.RateLimit = defaults
	}

	// Gin 默认信任所有代理；这里显式收敛为同一份白名单，避免未来使用 c.ClientIP 时信任伪造转发头。
	trustedProxies := make([]string, 0, len(deps.RateLimit.TrustedCIDRs)+len(deps.RateLimit.TrustedIPs))
	for _, trustedIP := range deps.RateLimit.TrustedIPs {
		if trustedIP != nil {
			trustedProxies = append(trustedProxies, trustedIP.String())
		}
	}
	for _, trustedCIDR := range deps.RateLimit.TrustedCIDRs {
		if trustedCIDR != nil {
			trustedProxies = append(trustedProxies, trustedCIDR.String())
		}
	}
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		deps.Logger.Printf("可信代理配置无效，已禁用转发头解析: %v", err)
		_ = r.SetTrustedProxies(nil)
	}
	clientIPResolver := middleware.NewClientIPResolver(deps.RateLimit.TrustedCIDRs, deps.RateLimit.TrustedIPs)

	r.Use(
		middleware.RequestMetadata(),
		middleware.Recovery(deps.Logger),
		middleware.AccessLog(deps.Logger, clientIPResolver),
		middleware.CORS(deps.Origin),
	)

	h := deps.Handlers
	if h.Health == nil {
		h.Health = handler.New(handler.Dependencies{TokenParser: deps.TokenParser}).Health
	}
	if h.Ready == nil {
		h.Ready = handler.New(handler.Dependencies{TokenParser: deps.TokenParser}).Ready
	}
	if h.Root == nil {
		h.Root = handler.New(handler.Dependencies{TokenParser: deps.TokenParser}).Root
	}
	if h.APIError == nil {
		h.APIError = handler.Unimplemented()
	}
	if h.Login == nil {
		h.Login = h.APIError
	}
	if h.Logout == nil {
		h.Logout = h.APIError
	}
	if h.GetSettings == nil {
		h.GetSettings = h.APIError
	}
	if h.GetMonitor == nil {
		h.GetMonitor = h.APIError
	}
	if h.UpdateSettings == nil {
		h.UpdateSettings = h.APIError
	}
	if h.SystemConfig == nil {
		h.SystemConfig = h.APIError
	}
	if h.Orders.CreateAPI == nil {
		h.Orders.CreateAPI = h.APIError
	}
	if h.Orders.GetAPI == nil {
		h.Orders.GetAPI = h.APIError
	}
	if h.Orders.ListAPI == nil {
		h.Orders.ListAPI = h.APIError
	}
	if h.Orders.StatisticsAPI == nil {
		h.Orders.StatisticsAPI = h.APIError
	}
	if h.Orders.DetailAPI == nil {
		h.Orders.DetailAPI = h.APIError
	}
	if h.Orders.CheckAPI == nil {
		h.Orders.CheckAPI = h.APIError
	}
	if h.Orders.ReturnURLAPI == nil {
		h.Orders.ReturnURLAPI = h.APIError
	}
	if h.Orders.CloseAPI == nil {
		h.Orders.CloseAPI = h.APIError
	}
	if h.Monitor.HeartbeatAPI == nil {
		h.Monitor.HeartbeatAPI = h.APIError
	}
	if h.Monitor.PushAPI == nil {
		h.Monitor.PushAPI = h.APIError
	}
	if h.QRCodes.GenerateAPI == nil {
		h.QRCodes.GenerateAPI = h.APIError
	}
	if h.QRCodes.ParseAPI == nil {
		h.QRCodes.ParseAPI = h.APIError
	}
	if h.QRCodes.ListAPI == nil {
		h.QRCodes.ListAPI = h.APIError
	}
	if h.QRCodes.ListWechatAPI == nil {
		h.QRCodes.ListWechatAPI = h.APIError
	}
	if h.QRCodes.ListAlipayAPI == nil {
		h.QRCodes.ListAlipayAPI = h.APIError
	}
	if h.QRCodes.CreateAPI == nil {
		h.QRCodes.CreateAPI = h.APIError
	}
	if h.QRCodes.CreateWechatAPI == nil {
		h.QRCodes.CreateWechatAPI = h.APIError
	}
	if h.QRCodes.CreateAlipayAPI == nil {
		h.QRCodes.CreateAlipayAPI = h.APIError
	}
	if h.QRCodes.SetStateAPI == nil {
		h.QRCodes.SetStateAPI = h.APIError
	}
	if h.QRCodes.DeleteAPI == nil {
		h.QRCodes.DeleteAPI = h.APIError
	}
	if h.QRCodes.DeleteWechatAPI == nil {
		h.QRCodes.DeleteWechatAPI = h.APIError
	}
	if h.QRCodes.DeleteAlipayAPI == nil {
		h.QRCodes.DeleteAlipayAPI = h.APIError
	}

	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)
	r.GET("/", h.Root)

	registerAPI(r, h, deps)
	registerMonitor(r, h, deps)
	return r
}

func registerAPI(r *gin.Engine, h handler.Handlers, deps RouterDeps) {
	api := r.Group("/api")

	api.POST("/auth/login", middleware.NewIPRateLimit(deps.RateLimit.Login, deps.RateLimit.TrustedCIDRs, deps.RateLimit.TrustedIPs), h.Login)
	api.POST("/auth/logout", middleware.TokenAuth(deps.TokenParser), h.Logout)

	// 管理读写接口统一使用新 Token；只读模式只允许其中的读请求。
	management := api.Group("", middleware.TokenAuth(deps.TokenParser), middleware.WriteGuard(deps.RuntimeMode))
	management.GET("/config/settings", h.GetSettings)
	management.POST("/config/settings", h.UpdateSettings)
	management.GET("/config/monitor", h.GetMonitor)
	management.GET("/config/get", h.SystemConfig)
	management.GET("/config/status", h.Orders.StatisticsAPI)
	management.GET("/order/list", h.Orders.ListAPI)
	management.GET("/order/detail/:id", h.Orders.DetailAPI)
	management.POST("/qrcode/parse", h.QRCodes.ParseAPI)
	management.GET("/qrcode/list", h.QRCodes.ListAPI)
	management.GET("/qrcode/wechat", h.QRCodes.ListWechatAPI)
	management.GET("/qrcode/alipay", h.QRCodes.ListAlipayAPI)
	management.POST("/qrcode/add", h.QRCodes.CreateAPI)
	management.POST("/qrcode/wechat", h.QRCodes.CreateWechatAPI)
	management.POST("/qrcode/alipay", h.QRCodes.CreateAlipayAPI)
	management.POST("/qrcode/bind/:id", h.QRCodes.SetStateAPI)
	management.DELETE("/qrcode/wechat/:id", h.QRCodes.DeleteWechatAPI)
	management.DELETE("/qrcode/alipay/:id", h.QRCodes.DeleteAlipayAPI)
	management.DELETE("/qrcode/:id", h.QRCodes.DeleteAPI)
	management.GET("/user/info", h.UserInfo)
	management.GET("/user/list", h.UserList)
	management.GET("/menu", h.Menu)
	management.DELETE("/order/:id", h.Orders.DeleteAPI)
	management.DELETE("/order/last", h.Orders.DeleteLastAPI)
	management.POST("/order/expired", h.Orders.ExpiredAPI)
	management.DELETE("/order/expired", h.Orders.DeleteExpiredAPI)
	management.POST("/order/reissue/:id", h.Orders.ReissueAPI)
	management.POST("/config/save", h.UpdateSettings)
	management.POST("/config/monitor", h.UpdateMonitor)

	// 公开支付端点只接受高熵 token；内部 order_id 仅保留在已鉴权管理端路由中。
	public := api.Group("")
	public.Any("/order/create",
		middleware.NewIPRateLimit(deps.RateLimit.Create, deps.RateLimit.TrustedCIDRs, deps.RateLimit.TrustedIPs),
		middleware.WriteGuardAlways(deps.RuntimeMode),
		h.Orders.CreateAPI,
	)
	public.GET("/order/get/:id",
		middleware.NewIPRateLimit(deps.RateLimit.PublicRead, deps.RateLimit.TrustedCIDRs, deps.RateLimit.TrustedIPs),
		middleware.NewTokenRateLimit(deps.RateLimit.PublicToken),
		h.Orders.GetAPI,
	)
	public.GET("/order/check/:id",
		middleware.NewIPRateLimit(deps.RateLimit.PublicRead, deps.RateLimit.TrustedCIDRs, deps.RateLimit.TrustedIPs),
		middleware.NewTokenRateLimit(deps.RateLimit.PublicToken),
		h.Orders.CheckAPI,
	)
	public.GET("/order/return-url/:id",
		middleware.NewIPRateLimit(deps.RateLimit.PublicRead, deps.RateLimit.TrustedCIDRs, deps.RateLimit.TrustedIPs),
		middleware.NewTokenRateLimit(deps.RateLimit.PublicToken),
		h.Orders.ReturnURLAPI,
	)
	public.GET("/qrcode/generate",
		middleware.NewIPRateLimit(deps.RateLimit.QRCode, deps.RateLimit.TrustedCIDRs, deps.RateLimit.TrustedIPs),
		h.QRCodes.GenerateAPI,
	)
	management.POST("/order/close/:id", h.Orders.CloseAPI)

}

func registerMonitor(r *gin.Engine, h handler.Handlers, deps RouterDeps) {
	group := r.Group("/api/monitor", middleware.WriteGuardAlways(deps.RuntimeMode))
	group.Any("/heart",
		middleware.NewIPRateLimit(deps.RateLimit.MonitorHeart, deps.RateLimit.TrustedCIDRs, deps.RateLimit.TrustedIPs),
		h.Monitor.HeartbeatAPI,
	)
	group.Any("/push",
		middleware.NewIPRateLimit(deps.RateLimit.MonitorPush, deps.RateLimit.TrustedCIDRs, deps.RateLimit.TrustedIPs),
		h.Monitor.PushAPI,
	)
}
