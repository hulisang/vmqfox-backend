package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
	"github.com/hulisang/vmqfox-backend/internal/http/middleware"
	"github.com/hulisang/vmqfox-backend/internal/usecase"
)

type OrderManager interface {
	Create(context.Context, usecase.CreateOrderInput) (usecase.CreateOrderResult, error)
	Get(context.Context, string) (usecase.OrderView, error)
	List(context.Context, usecase.ListOrdersInput) (usecase.OrderPageView, error)
	Statistics(context.Context) (usecase.OrderStatisticsView, error)
	Detail(context.Context, int64) (order.Order, error)
	Check(context.Context, string) (usecase.CheckOrderResult, error)
	ReturnURL(context.Context, string) (usecase.ReturnURLResult, error)
	CloseByID(context.Context, int64) error
	DeleteByID(context.Context, int64) error
	DeleteLast(context.Context, time.Duration) (int64, error)
	ExpireOrders(context.Context) (int, error)
	DeleteExpired(context.Context) (int64, error)
	ReissueByID(context.Context, int64) error
}

type OrderHandlers struct {
	CreateAPI        gin.HandlerFunc
	GetAPI           gin.HandlerFunc
	ListAPI          gin.HandlerFunc
	StatisticsAPI    gin.HandlerFunc
	DetailAPI        gin.HandlerFunc
	CheckAPI         gin.HandlerFunc
	ReturnURLAPI     gin.HandlerFunc
	CloseAPI         gin.HandlerFunc
	DeleteAPI        gin.HandlerFunc
	DeleteLastAPI    gin.HandlerFunc
	ExpiredAPI       gin.HandlerFunc
	DeleteExpiredAPI gin.HandlerFunc
	ReissueAPI       gin.HandlerFunc
}

// newOrderHandlers 统一装配订单路由处理器，避免公开令牌改造遗漏原有管理端处理器。
func newOrderHandlers(service OrderManager) OrderHandlers {
	return OrderHandlers{
		CreateAPI:        createOrderHandler(service),
		GetAPI:           getOrderAPIHandler(service),
		ListAPI:          listOrdersAPIHandler(service),
		StatisticsAPI:    orderStatisticsAPIHandler(service),
		DetailAPI:        orderDetailAPIHandler(service),
		CheckAPI:         checkOrderAPIHandler(service),
		ReturnURLAPI:     returnURLAPIHandler(service),
		CloseAPI:         closeOrderAPIHandler(service),
		DeleteAPI:        deleteOrderAPIHandler(service),
		DeleteLastAPI:    deleteLastOrderAPIHandler(service),
		ExpiredAPI:       expireOrdersAPIHandler(service),
		DeleteExpiredAPI: deleteExpiredOrdersAPIHandler(service),
		ReissueAPI:       reissueOrderAPIHandler(service),
	}
}

type PublicOrderView struct {
	PayID            string `json:"payId"`
	PayType          int    `json:"payType"`
	Price            string `json:"price"`
	ReallyPrice      string `json:"reallyPrice"`
	PayURL           string `json:"payUrl"`
	IsAuto           int    `json:"isAuto"`
	State            int    `json:"state"`
	StateText        string `json:"stateText"`
	TimeoutMinutes   int    `json:"timeOut"`
	CreatedAt        int64  `json:"date"`
	RemainingSeconds int64  `json:"remainingSeconds"`
}

type CreateOrderResponse struct {
	PayID       string `json:"payId"`
	OrderID     string `json:"orderId"`
	PublicToken string `json:"publicToken"`
	PayType     int    `json:"payType"`
	Price       string `json:"price"`
	ReallyPrice string `json:"reallyPrice"`
	PayURL      string `json:"payUrl"`
	IsAuto      int    `json:"isAuto"`
	RedirectURL string `json:"redirectUrl"`
}

type PublicOrderCheck struct {
	State            int   `json:"state"`
	RemainingSeconds int64 `json:"remainingSeconds"`
}

func createOrderHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		params, err := scalarRequestParams(c)
		if err != nil {
			writeOrderBindingError(c)
			return
		}
		result, err := service.Create(c.Request.Context(), usecase.CreateOrderInput{
			PayID:     params["payId"],
			Param:     params["param"],
			Type:      params["type"],
			Price:     params["price"],
			Sign:      params["sign"],
			NotifyURL: params["notifyUrl"],
			ReturnURL: params["returnUrl"],
		})
		if err != nil {
			writeOrderError(c, err)
			return
		}
		if params["isHtml"] == "1" {
			writePaymentRedirectHTML(c, result.RedirectURL, true)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", apiCreateOrderData(result)))
	}
}

func getOrderAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		result, err := service.Get(c.Request.Context(), publicOrderToken(c))
		if err != nil {
			writeOrderError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", apiOrderViewData(result)))
	}
}

func publicOrderToken(c *gin.Context) string {
	if token := c.Param("token"); token != "" {
		return token
	}
	return c.Param("id")
}

func listOrdersAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		page, err := positiveQueryInt(c.Query("page"), 1)
		if err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "分页参数无效", nil))
			return
		}
		limit, err := positiveQueryInt(c.Query("limit"), 10)
		if err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "分页参数无效", nil))
			return
		}

		var state *order.Status
		if rawState := strings.TrimSpace(c.Query("state")); rawState != "" {
			value, parseErr := strconv.Atoi(rawState)
			parsed := order.Status(value)
			if parseErr != nil || !parsed.Valid() {
				c.JSON(http.StatusOK, php.NewEnvelope(400, "订单状态无效", nil))
				return
			}
			state = &parsed
		}

		result, err := service.List(c.Request.Context(), usecase.ListOrdersInput{
			State: state,
			Page:  page,
			Limit: limit,
		})
		if err != nil {
			writeOrderError(c, err)
			return
		}
		items := make([]gin.H, 0, len(result.Items))
		for _, value := range result.Items {
			items = append(items, apiOrderRowData(value))
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{
			"total": result.Total,
			"items": items,
		}))
	}
}

func orderStatisticsAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		statistics, err := service.Statistics(c.Request.Context())
		if err != nil {
			writeOrderError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{
			"todayOrder":        statistics.TodayOrders,
			"todaySuccessOrder": statistics.TodayPaidOrders,
			"todayCloseOrder":   statistics.TodayClosedOrders,
			"todayMoney":        order.FormatCents(statistics.TodayPaidCents),
			"countOrder":        statistics.TotalOrders,
			"countMoney":        order.FormatCents(statistics.TotalPaidCents),
		}))
	}
}

func orderDetailAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "订单ID无效", nil))
			return
		}
		value, err := service.Detail(c.Request.Context(), id)
		if err != nil {
			writeOrderError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", apiOrderRowData(value)))
	}
}

func checkOrderAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		result, err := service.Check(c.Request.Context(), publicOrderToken(c))
		if err != nil {
			writeOrderError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, result.Message, apiCheckOrderData(result)))
	}
}

func returnURLAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		result, err := service.ReturnURL(c.Request.Context(), publicOrderToken(c))
		if err != nil {
			writeOrderError(c, err)
			return
		}
		// 回跳 URL 自身已携带服务端签名参数，避免额外下发可复用的独立签名字段。
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{
			"returnUrl": result.ReturnURL,
			"mode":      "new-first",
		}))
	}
}

func closeOrderAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "订单ID无效", nil))
			return
		}
		if err := service.CloseByID(c.Request.Context(), id); err != nil {
			writeOrderError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "关闭订单成功", nil))
	}
}

func deleteOrderAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "订单ID无效", nil))
			return
		}
		if err := service.DeleteByID(c.Request.Context(), id); err != nil {
			writeOrderError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "删除订单成功", nil))
	}
}

func deleteLastOrderAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		olderThan := 24 * time.Hour
		count, err := service.DeleteLast(c.Request.Context(), olderThan)
		if err != nil {
			writeOrderError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "清理旧订单成功", gin.H{
			"count":        count,
			"deletedCount": count,
		}))
	}
}

func expireOrdersAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		count, err := service.ExpireOrders(c.Request.Context())
		if err != nil {
			writeOrderError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "处理过期订单成功", gin.H{
			"count":        count,
			"expiredCount": count,
		}))
	}
}

func reissueOrderAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "订单ID无效", nil))
			return
		}
		if err := service.ReissueByID(c.Request.Context(), id); err != nil {
			writeOrderError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "人工补单成功", nil))
	}
}

func deleteExpiredOrdersAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c)
			return
		}
		count, err := service.DeleteExpired(c.Request.Context())
		if err != nil {
			writeOrderError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "删除过期订单成功", gin.H{
			"count":        count,
			"deletedCount": count,
		}))
	}
}

func apiCreateOrderData(result usecase.CreateOrderResult) CreateOrderResponse {
	value := result.Order
	return CreateOrderResponse{
		PayID:       value.PayID,
		OrderID:     value.OrderID,
		PublicToken: value.PublicToken,
		PayType:     int(value.Type),
		Price:       result.Price,
		ReallyPrice: result.ReallyPrice,
		PayURL:      value.PayURL,
		IsAuto:      boolInt(value.IsAuto),
		RedirectURL: result.RedirectURL,
	}
}

// apiOrderViewData 是匿名收银台专用 DTO，刻意不包含通知地址、回跳原文、透传参数和内部订单 ID。
func apiOrderViewData(result usecase.OrderView) PublicOrderView {
	value := result.Order
	publicState := value.State
	stateText := result.StateText
	if publicState == order.StatusNotifyFailed {
		// 通知失败不改变付款事实；公开支付页只消费支付状态，管理端仍保留原始状态 2。
		publicState = order.StatusPaid
		stateText = "已支付"
	}
	return PublicOrderView{
		PayID:            value.PayID,
		PayType:          int(value.Type),
		Price:            amountText(value.PriceText, value.PriceCents),
		ReallyPrice:      amountText(value.ReallyPriceText, value.ReallyPriceCents),
		PayURL:           value.PayURL,
		IsAuto:           boolInt(value.IsAuto),
		State:            int(publicState),
		StateText:        stateText,
		TimeoutMinutes:   result.TimeoutMinutes,
		CreatedAt:        value.CreatedAt.Unix(),
		RemainingSeconds: result.RemainingSeconds,
	}
}

func apiCheckOrderData(result usecase.CheckOrderResult) PublicOrderCheck {
	state := result.State
	if state == order.StatusNotifyFailed {
		state = order.StatusPaid
	}
	return PublicOrderCheck{
		State:            int(state),
		RemainingSeconds: result.RemainingSeconds,
	}
}

func apiOrderRowData(value order.Order) gin.H {
	return gin.H{
		"id":           value.ID,
		"order_id":     value.OrderID,
		"pay_id":       value.PayID,
		"type":         int(value.Type),
		"type_text":    paymentTypeText(value.Type),
		"price":        amountText(value.PriceText, value.PriceCents),
		"really_price": amountText(value.ReallyPriceText, value.ReallyPriceCents),
		"state":        int(value.State),
		"state_text":   apiOrderStateText(value.State),
		"param":        value.Param,
		"pay_url":      value.PayURL,
		"is_auto":      boolInt(value.IsAuto),
		"notify_url":   value.NotifyURL,
		"return_url":   value.ReturnURL,
		"create_date":  value.CreatedAt.Unix(),
		"pay_date":     unixOrZero(value.PaidAt),
		"close_date":   unixOrZero(value.ClosedAt),
		"create_time":  value.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func positiveQueryInt(raw string, fallback int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, errors.New("参数必须为正整数")
	}
	return value, nil
}

func unixOrZero(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.Unix()
}

func paymentTypeText(value payment.Type) string {
	if value == payment.Wechat {
		return "微信"
	}
	return "支付宝"
}

func apiOrderStateText(value order.Status) string {
	switch value {
	case order.StatusClosed:
		return "已关闭"
	case order.StatusPending:
		return "未支付"
	case order.StatusPaid:
		return "已支付"
	case order.StatusNotifyFailed:
		return "通知失败"
	default:
		return "未知状态"
	}
}

func amountText(text string, cents int64) string {
	raw := strings.TrimSpace(text)
	// 只有当商户提交的原始金额文本与内部分值完全等价时才回显原文，避免格式漂移或伪造精度。
	if parsed, err := order.ParseAmountCents(raw); err == nil && parsed == cents {
		return raw
	}
	return order.FormatCents(cents)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// writePaymentRedirectHTML 输出跳转页。内联脚本携带本次请求的 CSP nonce，
// 因此无需为该端点放开 script-src 'unsafe-inline'。
func writePaymentRedirectHTML(c *gin.Context, redirectURL string, fullPage bool) {
	encodedURL, _ := json.Marshal(redirectURL)
	nonceAttribute := ""
	if nonce := middleware.CSPNonce(c); nonce != "" {
		encodedNonce, _ := json.Marshal(nonce)
		nonceAttribute = " nonce=" + string(encodedNonce)
	}
	c.Header("Content-Type", php.ContentTypeHTML)
	c.Header("Cache-Control", "no-store")
	if !fullPage {
		c.String(http.StatusOK, "<script%s>window.location.href = %s</script>", nonceAttribute, encodedURL)
		return
	}
	c.String(http.StatusOK, `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
  <title>正在跳转到支付页面...</title>
  <style>
    body { background:#f5f5f5; font-family:Arial,sans-serif; text-align:center; padding-top:100px; }
    .loading { display:inline-block; width:50px; height:50px; border:3px solid rgba(0,0,0,.3); border-radius:50%%; border-top-color:#333; animation:spin 1s ease-in-out infinite; }
    @keyframes spin { to { transform:rotate(360deg); } }
    .text { margin-top:20px; color:#333; font-size:16px; }
  </style>
</head>
<body>
  <div class="loading"></div>
  <div class="text">正在跳转到支付页面，请稍候...</div>
  <script%s>window.location.href = %s;</script>
</body>
</html>`, nonceAttribute, encodedURL)
}

func writeOrderUnavailable(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "订单服务不可用", nil))
}

func writeOrderBindingError(c *gin.Context) {
	c.JSON(http.StatusOK, php.NewEnvelope(400, "请求参数格式错误", nil))
}

func writeOrderError(c *gin.Context, err error) {
	var appError *usecase.Error
	if !errors.As(err, &appError) {
		c.JSON(http.StatusOK, php.NewEnvelope(500, "服务器处理请求时发生错误", nil))
		return
	}
	code := 400
	if appError.Code == usecase.CodeDependency {
		code = 500
	}
	c.JSON(http.StatusOK, php.NewEnvelope(code, appError.Message, nil))
}
