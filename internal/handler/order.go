package handler

import (
	"strconv"
	"time"

	"vmqfox-api-go/internal/middleware"
	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/service"
	"vmqfox-api-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// OrderHandler 订单处理器
type OrderHandler struct {
	orderService   service.OrderService
	settingService service.SettingService
}

// NewOrderHandler 创建订单处理器
func NewOrderHandler(orderService service.OrderService, settingService service.SettingService) *OrderHandler {
	return &OrderHandler{
		orderService:   orderService,
		settingService: settingService,
	}
}

// checkOrderAccess 检查用户是否有权限访问指定订单
func (h *OrderHandler) checkOrderAccess(c *gin.Context, orderID uint) (*model.Order, bool) {
	// 获取订单信息
	order, err := h.orderService.GetOrderByID(orderID)
	if err != nil {
		if err == service.ErrOrderNotFound {
			response.Error(c, response.CodeNotFound, "Order not found")
		} else {
			response.InternalError(c, "Failed to get order")
		}
		return nil, false
	}

	// 检查数据访问权限
	if !middleware.ValidateResourceAccess(c, order.User_id) {
		response.Forbidden(c, "Access denied: insufficient permissions")
		return nil, false
	}

	return order, true
}

// GetOrders 获取订单列表
// @Summary 获取订单列表
// @Description 获取订单列表，支持分页和筛选
// @Tags orders
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param status query int false "订单状态"
// @Param type query string false "订单类型"
// @Param order_id query string false "订单号"
// @Param user_id query int false "用户ID"
// @Param start_at query string false "开始时间"
// @Param end_at query string false "结束时间"
// @Success 200 {object} response.PagedResponse
// @Failure 401 {object} response.Response
// @Router /api/v2/orders [get]
func (h *OrderHandler) GetOrders(c *gin.Context) {
	var req model.OrderListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 设置默认值
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	// 应用数据隔离：根据用户角色自动设置用户过滤
	middleware.ApplyUserFilter(c, &req.User_id)

	// 获取订单列表
	orders, total, err := h.orderService.GetOrders(&req)
	if err != nil {
		response.InternalError(c, "Failed to get orders")
		return
	}

	// 转换为响应格式
	orderResponses := make([]*model.OrderResponse, len(orders))
	for i, order := range orders {
		orderResponses[i] = order.ToResponse()
	}

	// 计算总页数
	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	// 返回分页响应
	response.SuccessPaged(c, orderResponses, response.PageMeta{
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// CreateOrder 创建订单
// @Summary 创建订单
// @Description 创建新订单
// @Tags orders
// @Accept json
// @Produce json
// @Param order body model.CreateOrderRequest true "订单信息"
// @Success 200 {object} response.Response{data=model.OrderResponse}
// @Failure 400 {object} response.Response
// @Router /api/v2/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req model.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 获取当前用户ID
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// 获取客户端信息
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// 创建订单
	order, err := h.orderService.CreateOrder(userID, &req, clientIP, userAgent)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			response.Error(c, response.CodeUserNotFound)
			return
		case service.ErrInvalidAmount:
			response.BadRequest(c, "Invalid amount")
			return
		case service.ErrInvalidOrderType:
			response.BadRequest(c, "Invalid order type")
			return
		case service.ErrOrderExists:
			response.Error(c, response.CodeConflict, "Order already exists")
			return
		default:
			response.InternalError(c, "Failed to create order")
			return
		}
	}

	response.SuccessWithMessage(c, "Order created successfully", order.ToResponse())
}

// GetOrder 获取订单详情
// @Summary 获取订单详情
// @Description 根据订单ID获取订单详情
// @Tags orders
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response{data=model.OrderResponse}
// @Failure 404 {object} response.Response
// @Router /api/v2/orders/{id} [get]
func (h *OrderHandler) GetOrder(c *gin.Context) {
	// 获取订单ID
	orderID, err := strconv.ParseUint(c.Param("order_id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	// 检查订单访问权限（包含数据隔离）
	order, hasAccess := h.checkOrderAccess(c, uint(orderID))
	if !hasAccess {
		return
	}

	response.Success(c, order.ToResponse())
}

// GetOrderStatus 获取订单状态
// @Summary 获取订单状态
// @Description 根据订单ID获取订单状态
// @Tags orders
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response{data=model.OrderStatusResponse}
// @Failure 404 {object} response.Response
// @Router /api/v2/orders/{id}/status [get]
func (h *OrderHandler) GetOrderStatus(c *gin.Context) {
	// 获取订单ID
	orderID, err := strconv.ParseUint(c.Param("order_id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	// 先获取订单以获得订单号
	order, err := h.orderService.GetOrderByID(uint(orderID))
	if err != nil {
		if err == service.ErrOrderNotFound {
			response.Error(c, response.CodeNotFound, "Order not found")
			return
		}
		response.InternalError(c, "Failed to get order")
		return
	}

	// 获取订单状态
	status, err := h.orderService.GetOrderStatus(order.Order_id)
	if err != nil {
		response.InternalError(c, "Failed to get order status")
		return
	}

	response.Success(c, status)
}

// UpdateOrder 更新订单
// @Summary 更新订单
// @Description 更新订单信息
// @Tags orders
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Param order body model.UpdateOrderRequest true "订单信息"
// @Success 200 {object} response.Response{data=model.OrderResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v2/orders/{id} [put]
func (h *OrderHandler) UpdateOrder(c *gin.Context) {
	// 获取订单ID
	orderID, err := strconv.ParseUint(c.Param("order_id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	var req model.UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 更新订单
	order, err := h.orderService.UpdateOrder(uint(orderID), &req)
	if err != nil {
		switch err {
		case service.ErrOrderNotFound:
			response.Error(c, response.CodeNotFound, "Order not found")
			return
		case service.ErrOrderClosed:
			response.BadRequest(c, "Order is closed and cannot be updated")
			return
		default:
			response.InternalError(c, "Failed to update order")
			return
		}
	}

	response.SuccessWithMessage(c, "Order updated successfully", order.ToResponse())
}

// DeleteOrder 删除订单
// @Summary 删除订单
// @Description 删除订单
// @Tags orders
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v2/orders/{id} [delete]
func (h *OrderHandler) DeleteOrder(c *gin.Context) {
	// 获取订单ID参数
	orderIDStr := c.Param("order_id")

	if orderIDStr == "" {
		response.ValidationFailed(c, "Order ID is required")
		return
	}

	// 先尝试按订单号查询订单
	order, err := h.orderService.GetOrderByOrderID(orderIDStr)
	if err != nil {
		// 如果按订单号查询失败，尝试按数字ID查询
		if id, parseErr := strconv.ParseUint(orderIDStr, 10, 32); parseErr == nil {
			order, err = h.orderService.GetOrderByID(uint(id))
		}

		if err != nil {
			if err == service.ErrOrderNotFound {
				response.NotFound(c, "Order not found")
				return
			}
			response.InternalError(c, "Failed to get order")
			return
		}
	}

	// 检查数据访问权限
	if !middleware.ValidateResourceAccess(c, order.User_id) {
		response.Forbidden(c, "Access denied")
		return
	}

	// 删除订单
	err = h.orderService.DeleteOrder(order.Id)
	if err != nil {
		switch err {
		case service.ErrOrderNotFound:
			response.Error(c, response.CodeNotFound, "Order not found")
			return
		case service.ErrOrderPaid:
			response.BadRequest(c, "Paid order cannot be deleted")
			return
		default:
			response.InternalError(c, "Failed to delete order")
			return
		}
	}

	response.SuccessWithMessage(c, "Order deleted successfully", nil)
}

// CloseOrder 关闭订单
// @Summary 关闭订单
// @Description 关闭指定订单
// @Tags orders
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v2/orders/{id}/close [put]
func (h *OrderHandler) CloseOrder(c *gin.Context) {
	// 获取订单ID
	orderID, err := strconv.ParseUint(c.Param("order_id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	// 关闭订单
	err = h.orderService.CloseOrder(uint(orderID))
	if err != nil {
		switch err {
		case service.ErrOrderNotFound:
			response.Error(c, response.CodeNotFound, "Order not found")
			return
		case service.ErrOrderPaid:
			response.BadRequest(c, "Paid order cannot be closed")
			return
		case service.ErrOrderClosed:
			response.BadRequest(c, "Order is already closed")
			return
		default:
			response.InternalError(c, "Failed to close order")
			return
		}
	}

	response.SuccessWithMessage(c, "Order closed successfully", nil)
}

// CloseExpiredOrders 关闭过期订单
// @Summary 关闭过期订单
// @Description 批量关闭过期订单
// @Tags orders
// @Accept json
// @Produce json
// @Param request body model.CloseExpiredOrdersRequest false "关闭过期订单请求"
// @Success 200 {object} response.Response{data=object{closed_count=int}}
// @Failure 400 {object} response.Response
// @Router /api/v2/orders/close-expired [post]
func (h *OrderHandler) CloseExpiredOrders(c *gin.Context) {
	var req model.CloseExpiredOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 关闭过期订单
	closedCount, err := h.orderService.CloseExpiredOrders(&req)
	if err != nil {
		response.InternalError(c, "Failed to close expired orders")
		return
	}

	response.SuccessWithMessage(c, "Expired orders closed successfully", map[string]interface{}{
		"closed_count": closedCount,
	})
}

// DeleteExpiredOrders 删除过期订单
// @Summary 删除过期订单
// @Description 批量删除过期订单
// @Tags orders
// @Accept json
// @Produce json
// @Param request body model.DeleteExpiredOrdersRequest false "删除过期订单请求"
// @Success 200 {object} response.Response{data=object{deleted_count=int}}
// @Failure 400 {object} response.Response
// @Router /api/v2/orders/delete-expired [post]
func (h *OrderHandler) DeleteExpiredOrders(c *gin.Context) {
	var req model.DeleteExpiredOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 删除过期订单
	deletedCount, err := h.orderService.DeleteExpiredOrders(&req)
	if err != nil {
		response.InternalError(c, "删除过期订单失败")
		return
	}

	response.SuccessWithMessage(c, "过期订单删除成功", map[string]interface{}{
		"deleted_count": deletedCount,
	})
}

// GenerateReturnURL 生成返回URL
// @Summary 生成返回URL
// @Description 为订单生成支付成功后的返回URL
// @Tags orders
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response{data=object{return_url=string}}
// @Failure 404 {object} response.Response
// @Router /api/v2/orders/{id}/return-url [get]
func (h *OrderHandler) GenerateReturnURL(c *gin.Context) {
	// 获取订单ID
	orderID, err := strconv.ParseUint(c.Param("order_id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	// 先获取订单以获得订单号
	order, err := h.orderService.GetOrderByID(uint(orderID))
	if err != nil {
		if err == service.ErrOrderNotFound {
			response.Error(c, response.CodeNotFound, "Order not found")
			return
		}
		response.InternalError(c, "Failed to get order")
		return
	}

	// 生成返回URL
	returnURL, err := h.orderService.GenerateReturnURL(order.Order_id)
	if err != nil {
		response.InternalError(c, "Failed to generate return URL")
		return
	}

	response.Success(c, map[string]interface{}{
		"return_url": returnURL,
	})
}

// GetOrderUnified 统一的订单查询接口
// @Summary 获取订单详情（支持公开和认证访问）
// @Description 根据访问类型返回订单详情，支付页面无需认证，管理后台需要认证
// @Tags orders
// @Produce json
// @Param order_id path string true "订单ID"
// @Param public query bool false "是否公开访问"
// @Success 200 {object} response.Response{data=model.Order}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v2/orders/{order_id} [get]
func (h *OrderHandler) GetOrderUnified(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		response.ValidationFailed(c, "Order ID is required")
		return
	}

	// 检查访问类型
	accessType, _ := c.Get("access_type")

	if accessType == "public" {
		// 公开访问，使用支付页面逻辑
		h.getOrderForPayment(c, orderID)
	} else {
		// 认证访问，使用管理后台逻辑
		h.getOrderForAdmin(c, orderID)
	}
}

// GetOrderStatusUnified 统一的订单状态查询接口
// @Summary 获取订单状态（支持公开和认证访问）
// @Description 根据访问类型返回订单状态，支付页面无需认证，管理后台需要认证
// @Tags orders
// @Produce json
// @Param order_id path string true "订单ID"
// @Param public query bool false "是否公开访问"
// @Success 200 {object} response.Response{data=model.OrderStatusResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v2/orders/{order_id}/status [get]
func (h *OrderHandler) GetOrderStatusUnified(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		response.ValidationFailed(c, "Order ID is required")
		return
	}

	// 检查访问类型
	accessType, _ := c.Get("access_type")

	if accessType == "public" {
		// 公开访问，使用支付页面逻辑
		h.getOrderStatusForPayment(c, orderID)
	} else {
		// 认证访问，使用管理后台逻辑
		h.getOrderStatusForAdmin(c, orderID)
	}
}

// getOrderForPayment 支付页面的订单查询逻辑
func (h *OrderHandler) getOrderForPayment(c *gin.Context, orderID string) {
	// 使用payment service的逻辑
	order, err := h.orderService.GetOrderByOrderID(orderID)
	if err != nil {
		if err == service.ErrOrderNotFound {
			response.Error(c, response.CodeNotFound, "Order not found")
			return
		}
		response.InternalError(c, "Failed to get payment order")
		return
	}

	// 获取订单超时时间设置
	closeTimeSetting, _ := h.settingService.GetSettingValue("close")
	closeTime := 5 // 默认5分钟
	if closeTimeSetting != "" {
		if ct, err := strconv.Atoi(closeTimeSetting); err == nil && ct > 0 {
			closeTime = ct
		}
	}

	// 计算剩余时间
	timeoutSeconds := closeTime * 60
	elapsedSeconds := int(time.Now().Unix() - order.Create_date)
	remainingSeconds := timeoutSeconds - elapsedSeconds
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}

	// 转换为支付页面格式的响应
	paymentOrder := &model.PaymentOrderResponse{
		Order_id:         order.Order_id,
		Pay_id:           order.Pay_id,
		Type:             order.Type,
		Type_text:        order.GetTypeText(),
		Price:            order.Price,
		Really_price:     order.Really_price,
		Pay_url:          order.Pay_url,
		State:            order.State,
		State_text:       order.GetStatusText(),
		Is_auto:          order.Is_auto,
		Create_date:      order.Create_date,
		Pay_date:         order.Pay_date,
		Close_date:       order.Close_date,
		Is_expired:       remainingSeconds <= 0 && order.State == 0,
		Expired_at:       order.Create_date + int64(timeoutSeconds),
		TimeOut:          closeTime,
		RemainingSeconds: remainingSeconds,
		Return_url:       order.Return_url,
		Param:            order.Param,
	}

	response.Success(c, paymentOrder)
}

// getOrderForAdmin 管理后台的订单查询逻辑
func (h *OrderHandler) getOrderForAdmin(c *gin.Context, orderID string) {
	// 先尝试按订单号查询
	order, err := h.orderService.GetOrderByOrderID(orderID)
	if err != nil {
		// 如果按订单号查询失败，尝试按ID查询
		if id, parseErr := strconv.ParseUint(orderID, 10, 32); parseErr == nil {
			order, err = h.orderService.GetOrderByID(uint(id))
		}

		if err != nil {
			if err == service.ErrOrderNotFound {
				response.NotFound(c, "Order not found")
				return
			}
			response.InternalError(c, "Failed to get order")
			return
		}
	}

	// 检查数据访问权限
	if !middleware.ValidateResourceAccess(c, order.User_id) {
		response.Forbidden(c, "Access denied")
		return
	}

	response.Success(c, order)
}

// getOrderStatusForPayment 支付页面的订单状态查询逻辑
func (h *OrderHandler) getOrderStatusForPayment(c *gin.Context, orderID string) {
	status, err := h.orderService.GetOrderStatus(orderID)
	if err != nil {
		if err == service.ErrOrderNotFound {
			response.Error(c, response.CodeNotFound, "Order not found")
			return
		}
		response.InternalError(c, "Failed to check payment status")
		return
	}

	response.Success(c, status)
}

// getOrderStatusForAdmin 管理后台的订单状态查询逻辑
func (h *OrderHandler) getOrderStatusForAdmin(c *gin.Context, orderID string) {
	// 先获取订单以检查权限
	order, err := h.orderService.GetOrderByOrderID(orderID)
	if err != nil {
		// 如果按订单号查询失败，尝试按ID查询
		if id, parseErr := strconv.ParseUint(orderID, 10, 32); parseErr == nil {
			order, err = h.orderService.GetOrderByID(uint(id))
		}

		if err != nil {
			if err == service.ErrOrderNotFound {
				response.NotFound(c, "Order not found")
				return
			}
			response.InternalError(c, "Failed to get order")
			return
		}
	}

	// 检查数据访问权限
	if !middleware.ValidateResourceAccess(c, order.User_id) {
		response.Forbidden(c, "Access denied")
		return
	}

	// 获取状态信息
	status, err := h.orderService.GetOrderStatus(order.Order_id)
	if err != nil {
		response.InternalError(c, "Failed to get order status")
		return
	}

	response.Success(c, status)
}
