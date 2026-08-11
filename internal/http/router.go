package http

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/http/handler"
	"github.com/hulisang/vmqfox-backend/internal/http/middleware"
)

type RouterDeps struct {
	Router      *gin.Engine
	Handlers    handler.Handlers
	TokenParser middleware.TokenParser
	Origin      string
	RuntimeMode middleware.RuntimeMode
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

	r.Use(
		middleware.RequestMetadata(),
		middleware.Recovery(deps.Logger),
		middleware.AccessLog(deps.Logger),
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
		h.APIError = handler.Unimplemented(handler.ProtocolAPI)
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
	if h.LegacyError == nil {
		h.LegacyError = handler.Unimplemented(handler.ProtocolLegacy)
	}
	if h.LegacyProbe == nil {
		h.LegacyProbe = h.LegacyError
	}
	if h.LayuiError == nil {
		h.LayuiError = handler.Unimplemented(handler.ProtocolLayui)
	}
	if h.RawError == nil {
		h.RawError = handler.Unimplemented(handler.ProtocolRaw)
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
	if h.Orders.CreateLegacy == nil {
		h.Orders.CreateLegacy = h.LegacyError
	}
	if h.Orders.GetLegacy == nil {
		h.Orders.GetLegacy = h.LegacyError
	}
	if h.Orders.CheckLegacy == nil {
		h.Orders.CheckLegacy = h.LegacyError
	}
	if h.Monitor.HeartbeatAPI == nil {
		h.Monitor.HeartbeatAPI = h.APIError
	}
	if h.Monitor.PushAPI == nil {
		h.Monitor.PushAPI = h.APIError
	}
	if h.Monitor.HeartbeatLegacy == nil {
		h.Monitor.HeartbeatLegacy = h.LegacyError
	}
	if h.Monitor.PushLegacy == nil {
		h.Monitor.PushLegacy = h.LegacyError
	}
	if h.QRCodes.GenerateAPI == nil {
		h.QRCodes.GenerateAPI = h.RawError
	}
	if h.QRCodes.GenerateLegacy == nil {
		h.QRCodes.GenerateLegacy = h.RawError
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
	if h.QRCodes.AddLegacy == nil {
		h.QRCodes.AddLegacy = h.LegacyError
	}
	if h.QRCodes.ListLegacy == nil {
		h.QRCodes.ListLegacy = h.LayuiError
	}
	if h.QRCodes.DeleteLegacy == nil {
		h.QRCodes.DeleteLegacy = h.LegacyError
	}
	if h.QRCodes.SetStateLegacy == nil {
		h.QRCodes.SetStateLegacy = h.LegacyError
	}

	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)
	r.GET("/", h.Root)

	registerAPI(r, h, deps)
	registerMonitor(r, h, deps)
	registerLegacy(r, h, deps)
	registerAdmin(r, h, deps)
	return r
}

func registerAPI(r *gin.Engine, h handler.Handlers, deps RouterDeps) {
	api := r.Group("/api")

	api.POST("/auth/login", h.Login)
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
	management.POST("/order/reissue/:id", h.Orders.ReissueAPI)
	management.POST("/config/save", h.UpdateSettings)
	management.POST("/config/monitor", h.UpdateMonitor)

	// 商户订单创建和支付页查询不继承管理 Token；创建即使使用 GET 也始终按写请求保护。
	public := api.Group("")
	public.Any("/order/create", middleware.WriteGuardAlways(deps.RuntimeMode), h.Orders.CreateAPI)
	public.GET("/order/get/:id", h.Orders.GetAPI)
	public.GET("/order/check/:id", h.Orders.CheckAPI)
	public.GET("/order/return-url/:id", h.Orders.ReturnURLAPI)
	public.GET("/qrcode/generate", h.QRCodes.GenerateAPI)
	management.POST("/order/close/:id", h.Orders.CloseAPI)

	api.POST("/admin/index/getSettings", middleware.TokenAuth(deps.TokenParser), h.GetSettings)
	api.POST("/admin/index/saveSetting", middleware.TokenAuth(deps.TokenParser), middleware.WriteGuard(deps.RuntimeMode), h.UpdateSettings)

	// route/app.php 中的历史 /api/* 别名必须与根路径分开登记。
	api.Any("/getMenu", middleware.TokenAuth(deps.TokenParser), h.Menu)
	api.Any("/createOrder", middleware.WriteGuardAlways(deps.RuntimeMode), h.Orders.CreateLegacy)
	api.Any("/getOrder", h.Orders.GetLegacy)
	api.Any("/checkOrder", h.Orders.CheckLegacy)
}

func registerMonitor(r *gin.Engine, h handler.Handlers, deps RouterDeps) {
	for _, prefix := range []string{"/api/monitor", "/api/v2/monitor"} {
		group := r.Group(prefix, middleware.WriteGuardAlways(deps.RuntimeMode))
		group.Any("/heart", h.Monitor.HeartbeatAPI)
		group.Any("/push", h.Monitor.PushAPI)
	}
}

func registerLegacy(r *gin.Engine, h handler.Handlers, deps RouterDeps) {
	// 历史登录和商户订单接口不接受 PHP Session；对应实现会分别校验新 Token/商户签名。
	register(r, routeSpec{method: "ANY", path: "/createOrder"}, middleware.WriteGuardAlways(deps.RuntimeMode), h.Orders.CreateLegacy)
	register(r, routeSpec{method: "ANY", path: "/getOrder"}, h.Orders.GetLegacy)
	register(r, routeSpec{method: "ANY", path: "/checkOrder"}, h.Orders.CheckLegacy)
	register(r, routeSpec{method: "ANY", path: "/appHeart"}, middleware.WriteGuardAlways(deps.RuntimeMode), h.Monitor.HeartbeatLegacy)
	register(r, routeSpec{method: "ANY", path: "/appPush"}, middleware.WriteGuardAlways(deps.RuntimeMode), h.Monitor.PushLegacy)
	register(r, routeSpec{method: "ANY", path: "/index/index/getReturn"}, h.LegacyProbe)
	register(r, routeSpec{method: "ANY", path: "/enQrcode"}, h.QRCodes.GenerateLegacy)
}

func registerAdmin(r *gin.Engine, h handler.Handlers, deps RouterDeps) {
	admin := r.Group("/admin", middleware.TokenAuth(deps.TokenParser), middleware.WriteGuard(deps.RuntimeMode))
	admin.GET("/enQrcode/:url", h.QRCodes.GenerateLegacy)
	admin.GET("/index/enQrcode", h.QRCodes.GenerateLegacy)
	admin.POST("/addPayQrcode", h.QRCodes.AddLegacy)
	admin.GET("/getPayQrcodes", h.QRCodes.ListLegacy)
	admin.POST("/delPayQrcode", h.QRCodes.DeleteLegacy)
	admin.POST("/setBd", h.QRCodes.SetStateLegacy)
	admin.POST("/index/addPayQrcode", h.QRCodes.AddLegacy)
	admin.GET("/index/getPayQrcodes", h.QRCodes.ListLegacy)
	admin.POST("/index/delPayQrcode", h.QRCodes.DeleteLegacy)
	admin.POST("/index/setBd", h.QRCodes.SetStateLegacy)
}

type routeSpec struct {
	method string
	path   string
}

type routeRegistrar interface {
	GET(string, ...gin.HandlerFunc) gin.IRoutes
	POST(string, ...gin.HandlerFunc) gin.IRoutes
	PUT(string, ...gin.HandlerFunc) gin.IRoutes
	PATCH(string, ...gin.HandlerFunc) gin.IRoutes
	DELETE(string, ...gin.HandlerFunc) gin.IRoutes
	Any(string, ...gin.HandlerFunc) gin.IRoutes
}

func register(r routeRegistrar, route routeSpec, handlers ...gin.HandlerFunc) {
	switch route.method {
	case "GET":
		r.GET(route.path, handlers...)
	case "POST":
		r.POST(route.path, handlers...)
	case "PUT":
		r.PUT(route.path, handlers...)
	case "PATCH":
		r.PATCH(route.path, handlers...)
	case "DELETE":
		r.DELETE(route.path, handlers...)
	default:
		r.Any(route.path, handlers...)
	}
}
