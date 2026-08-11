package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
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
	ReissueByID(context.Context, int64) error
}

type OrderHandlers struct {
	CreateAPI     gin.HandlerFunc
	GetAPI        gin.HandlerFunc
	ListAPI       gin.HandlerFunc
	StatisticsAPI gin.HandlerFunc
	DetailAPI     gin.HandlerFunc
	CheckAPI      gin.HandlerFunc
	ReturnURLAPI  gin.HandlerFunc
	CloseAPI      gin.HandlerFunc
	DeleteAPI     gin.HandlerFunc
	DeleteLastAPI gin.HandlerFunc
	ExpiredAPI    gin.HandlerFunc
	ReissueAPI    gin.HandlerFunc
	CreateLegacy  gin.HandlerFunc
	GetLegacy     gin.HandlerFunc
	CheckLegacy   gin.HandlerFunc
}

func newOrderHandlers(service OrderManager) OrderHandlers {
	return OrderHandlers{
		CreateAPI:     createOrderHandler(service, usecase.CreateSignatureNew, ProtocolAPI),
		GetAPI:        getOrderAPIHandler(service),
		ListAPI:       listOrdersAPIHandler(service),
		StatisticsAPI: orderStatisticsAPIHandler(service),
		DetailAPI:     orderDetailAPIHandler(service),
		CheckAPI:      checkOrderAPIHandler(service),
		ReturnURLAPI:  returnURLAPIHandler(service),
		CloseAPI:      closeOrderAPIHandler(service),
		DeleteAPI:     deleteOrderAPIHandler(service),
		DeleteLastAPI: deleteLastOrderAPIHandler(service),
		ExpiredAPI:    expireOrdersAPIHandler(service),
		ReissueAPI:    reissueOrderAPIHandler(service),
		CreateLegacy:  createOrderHandler(service, usecase.CreateSignatureLegacy, ProtocolLegacy),
		GetLegacy:     getOrderLegacyHandler(service),
		CheckLegacy:   checkOrderLegacyHandler(service),
	}
}

func createOrderHandler(service OrderManager, signatureMode usecase.CreateSignatureMode, style ProtocolStyle) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, style)
			return
		}
		params, err := scalarRequestParams(c)
		if err != nil {
			writeOrderBindingError(c, style)
			return
		}
		if style == ProtocolLegacy {
			if message := validateLegacyCreateParams(params); message != "" {
				c.JSON(http.StatusOK, php.NewEnvelope(-1, message, nil))
				return
			}
		}

		result, err := service.Create(c.Request.Context(), usecase.CreateOrderInput{
			PayID:         params["payId"],
			Param:         params["param"],
			Type:          params["type"],
			Price:         params["price"],
			Sign:          params["sign"],
			NotifyURL:     params["notifyUrl"],
			ReturnURL:     params["returnUrl"],
			SignatureMode: signatureMode,
		})
		if err != nil {
			writeOrderError(c, err, style)
			return
		}
		if params["isHtml"] == "1" {
			writePaymentRedirectHTML(c, result.RedirectURL, style == ProtocolAPI)
			return
		}

		if style == ProtocolLegacy {
			c.JSON(http.StatusOK, php.NewEnvelope(1, "成功", legacyCreateOrderData(result)))
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", apiCreateOrderData(result)))
	}
}

func getOrderAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, ProtocolAPI)
			return
		}
		result, err := service.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeOrderError(c, err, ProtocolAPI)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", apiOrderViewData(result)))
	}
}

func listOrdersAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, ProtocolAPI)
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
			writeOrderError(c, err, ProtocolAPI)
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
			writeOrderUnavailable(c, ProtocolAPI)
			return
		}
		statistics, err := service.Statistics(c.Request.Context())
		if err != nil {
			writeOrderError(c, err, ProtocolAPI)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{
			"todayOrder":        statistics.TodayOrders,
			"todaySuccessOrder": statistics.TodayPaidOrders,
			"todayCloseOrder":   statistics.TodayClosedOrders,
			"todayMoney":        amountNumber("", statistics.TodayPaidCents),
			"countOrder":        statistics.TotalOrders,
			"countMoney":        amountNumber("", statistics.TotalPaidCents),
		}))
	}
}

func orderDetailAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, ProtocolAPI)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "订单ID无效", nil))
			return
		}
		value, err := service.Detail(c.Request.Context(), id)
		if err != nil {
			writeOrderError(c, err, ProtocolAPI)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", apiOrderRowData(value)))
	}
}

func checkOrderAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, ProtocolAPI)
			return
		}
		result, err := service.Check(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeOrderError(c, err, ProtocolAPI)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, result.Message, apiCheckOrderData(result)))
	}
}

func returnURLAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, ProtocolAPI)
			return
		}
		result, err := service.ReturnURL(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeOrderError(c, err, ProtocolAPI)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{
			"returnUrl":       result.ReturnURL,
			"returnUrlNew":    result.ReturnURLNew,
			"returnUrlLegacy": result.ReturnURLLegacy,
			"mode":            "new-first",
			"sign":            result.Sign,
			"signLegacy":      result.SignLegacy,
		}))
	}
}

func closeOrderAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, ProtocolAPI)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "订单ID无效", nil))
			return
		}
		if err := service.CloseByID(c.Request.Context(), id); err != nil {
			writeOrderError(c, err, ProtocolAPI)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "关闭订单成功", nil))
	}
}

func deleteOrderAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, ProtocolAPI)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "订单ID无效", nil))
			return
		}
		if err := service.DeleteByID(c.Request.Context(), id); err != nil {
			writeOrderError(c, err, ProtocolAPI)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "删除订单成功", nil))
	}
}

func deleteLastOrderAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, ProtocolAPI)
			return
		}
		olderThan := 24 * time.Hour
		count, err := service.DeleteLast(c.Request.Context(), olderThan)
		if err != nil {
			writeOrderError(c, err, ProtocolAPI)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "清理旧订单成功", gin.H{
			"deletedCount": count,
		}))
	}
}

func expireOrdersAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, ProtocolAPI)
			return
		}
		count, err := service.ExpireOrders(c.Request.Context())
		if err != nil {
			writeOrderError(c, err, ProtocolAPI)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "处理过期订单成功", gin.H{
			"expiredCount": count,
		}))
	}
}

func reissueOrderAPIHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, ProtocolAPI)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "订单ID无效", nil))
			return
		}
		if err := service.ReissueByID(c.Request.Context(), id); err != nil {
			writeOrderError(c, err, ProtocolAPI)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "人工补单成功", nil))
	}
}

func getOrderLegacyHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, ProtocolLegacy)
			return
		}
		params, err := scalarRequestParams(c)
		if err != nil {
			writeOrderBindingError(c, ProtocolLegacy)
			return
		}
		if params["orderId"] == "" {
			c.JSON(http.StatusOK, php.NewEnvelope(-1, "云端订单编号不存在", nil))
			return
		}
		result, err := service.Get(c.Request.Context(), params["orderId"])
		if err != nil {
			if code, ok := usecase.ErrorCodeOf(err); ok && code == usecase.CodeNotFound {
				c.JSON(http.StatusOK, php.NewEnvelope(-1, "云端订单编号不存在", nil))
				return
			}
			writeOrderError(c, err, ProtocolLegacy)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(1, "成功", legacyOrderViewData(result)))
	}
}

func checkOrderLegacyHandler(service OrderManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeOrderUnavailable(c, ProtocolLegacy)
			return
		}
		params, err := scalarRequestParams(c)
		if err != nil {
			writeOrderBindingError(c, ProtocolLegacy)
			return
		}
		orderID := params["orderId"]
		if orderID == "" {
			c.JSON(http.StatusOK, php.NewEnvelope(-1, "订单ID不能为空", nil))
			return
		}
		result, err := service.Check(c.Request.Context(), orderID)
		if err != nil {
			if code, ok := usecase.ErrorCodeOf(err); ok && code == usecase.CodeNotFound {
				c.JSON(http.StatusOK, php.NewEnvelope(-1, "云端订单编号不存在", nil))
				return
			}
			writeOrderError(c, err, ProtocolLegacy)
			return
		}

		switch result.State {
		case order.StatusPending:
			c.JSON(http.StatusOK, php.NewEnvelope(-1, "订单未支付", nil))
		case order.StatusClosed:
			c.JSON(http.StatusOK, php.NewEnvelope(-1, "订单已过期", nil))
		default:
			if result.ReturnURL == "" {
				c.JSON(http.StatusOK, php.NewEnvelope(1, "订单支付成功，但未设置回调URL", nil))
				return
			}
			returnURL, returnErr := service.ReturnURL(c.Request.Context(), orderID)
			if returnErr != nil {
				writeOrderError(c, returnErr, ProtocolLegacy)
				return
			}
			c.JSON(http.StatusOK, php.NewEnvelope(1, "成功", returnURL.ReturnURLLegacy))
		}
	}
}

func validateLegacyCreateParams(params map[string]string) string {
	if params["payId"] == "" {
		return "请传入商户订单号"
	}
	if params["type"] == "" {
		return "请传入支付方式=>1|微信 2|支付宝"
	}
	if params["type"] != "1" && params["type"] != "2" {
		return "支付方式错误=>1|微信 2|支付宝"
	}
	if params["price"] == "" {
		return "请传入订单金额"
	}
	priceCents, err := order.ParseAmountCents(params["price"])
	if err != nil || priceCents <= 0 {
		return "订单金额必须大于0"
	}
	if params["sign"] == "" {
		return "请传入签名"
	}
	return ""
}

func apiCreateOrderData(result usecase.CreateOrderResult) gin.H {
	value := result.Order
	return gin.H{
		"payId":       value.PayID,
		"orderId":     value.OrderID,
		"payType":     int(value.Type),
		"price":       result.Price,
		"reallyPrice": result.ReallyPrice,
		"payUrl":      value.PayURL,
		"isAuto":      boolInt(value.IsAuto),
		"redirectUrl": result.RedirectURL,
	}
}

func legacyCreateOrderData(result usecase.CreateOrderResult) gin.H {
	value := result.Order
	return gin.H{
		"payId":       value.PayID,
		"orderId":     value.OrderID,
		"payType":     int(value.Type),
		"price":       result.Price,
		"reallyPrice": result.ReallyPrice,
		"payUrl":      value.PayURL,
		"isAuto":      boolInt(value.IsAuto),
		"state":       int(value.State),
		"timeOut":     result.TimeoutMinutes,
		"date":        value.CreatedAt.Unix(),
	}
}

func apiOrderViewData(result usecase.OrderView) gin.H {
	value := result.Order
	publicState := value.State
	stateText := result.StateText
	if publicState == order.StatusNotifyFailed {
		// 通知失败不改变付款事实；公开支付页只消费支付状态，管理端仍保留原始状态 2。
		publicState = order.StatusPaid
		stateText = "已支付"
	}
	return gin.H{
		"payId":            value.PayID,
		"orderId":          value.OrderID,
		"payType":          int(value.Type),
		"price":            amountNumber(value.PriceText, value.PriceCents),
		"reallyPrice":      amountNumber(value.ReallyPriceText, value.ReallyPriceCents),
		"payUrl":           value.PayURL,
		"isAuto":           boolInt(value.IsAuto),
		"state":            int(publicState),
		"stateText":        stateText,
		"timeOut":          result.TimeoutMinutes,
		"date":             value.CreatedAt.Unix(),
		"remainingSeconds": result.RemainingSeconds,
		"return_url":       value.ReturnURL,
		"param":            value.Param,
	}
}

func legacyOrderViewData(result usecase.OrderView) gin.H {
	value := result.Order
	return gin.H{
		"payId":       value.PayID,
		"orderId":     value.OrderID,
		"payType":     int(value.Type),
		"price":       amountText(value.PriceText, value.PriceCents),
		"reallyPrice": amountText(value.ReallyPriceText, value.ReallyPriceCents),
		"payUrl":      value.PayURL,
		"isAuto":      boolInt(value.IsAuto),
		"state":       int(value.State),
		"timeOut":     result.TimeoutMinutes,
		"date":        value.CreatedAt.Unix(),
	}
}

func apiCheckOrderData(result usecase.CheckOrderResult) gin.H {
	data := gin.H{
		"remainingSeconds": result.RemainingSeconds,
		"return_url":       result.ReturnURL,
		"param":            result.Param,
	}
	switch result.State {
	case order.StatusPaid, order.StatusNotifyFailed:
		data["redirectUrl"] = result.RedirectURL
	case order.StatusClosed:
		data["state"] = int(order.StatusClosed)
	default:
		data["state"] = int(order.StatusPending)
	}
	return data
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

func amountNumber(text string, cents int64) json.Number {
	raw := strings.TrimSpace(text)
	if parsed, err := order.ParseAmountCents(raw); err == nil && parsed == cents && validJSONNumber(raw) {
		return json.Number(raw)
	}
	return json.Number(order.FormatCents(cents))
}

func validJSONNumber(value string) bool {
	var number json.Number
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()
	if err := decoder.Decode(&number); err != nil || number.String() != value {
		return false
	}
	var trailing any
	return errors.Is(decoder.Decode(&trailing), io.EOF)
}

func amountText(text string, cents int64) string {
	if text != "" {
		return text
	}
	return order.FormatCents(cents)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func writePaymentRedirectHTML(c *gin.Context, redirectURL string, fullPage bool) {
	encodedURL, _ := json.Marshal(redirectURL)
	c.Header("Content-Type", php.ContentTypeHTML)
	if !fullPage {
		c.String(http.StatusOK, "<script>window.location.href = %s</script>", encodedURL)
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
  <script>window.location.href = %s;</script>
</body>
</html>`, encodedURL)
}

func writeOrderUnavailable(c *gin.Context, style ProtocolStyle) {
	if style == ProtocolLegacy {
		c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(-1, "订单服务不可用", nil))
		return
	}
	c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "订单服务不可用", nil))
}

func writeOrderBindingError(c *gin.Context, style ProtocolStyle) {
	if style == ProtocolLegacy {
		c.JSON(http.StatusOK, php.NewEnvelope(-1, "请求参数格式错误", nil))
		return
	}
	c.JSON(http.StatusOK, php.NewEnvelope(400, "请求参数格式错误", nil))
}

func writeOrderError(c *gin.Context, err error, style ProtocolStyle) {
	var appError *usecase.Error
	if !errors.As(err, &appError) {
		if style == ProtocolLegacy {
			c.JSON(http.StatusOK, php.NewEnvelope(-1, "服务器处理请求时发生错误", nil))
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(500, "服务器处理请求时发生错误", nil))
		return
	}

	if style == ProtocolLegacy {
		message := appError.Message
		if appError.Code == usecase.CodeDuplicateOrder {
			message = "订单号已存在"
		}
		c.JSON(http.StatusOK, php.NewEnvelope(-1, message, nil))
		return
	}

	code := 400
	if appError.Code == usecase.CodeDependency {
		code = 500
	}
	c.JSON(http.StatusOK, php.NewEnvelope(code, appError.Message, nil))
}
