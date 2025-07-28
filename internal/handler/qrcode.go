package handler

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"vmqfox-api-go/internal/middleware"
	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/service"
	"vmqfox-api-go/pkg/response"
)

type QrcodeHandler struct {
	qrcodeService service.QrcodeService
}

func NewQrcodeHandler(qrcodeService service.QrcodeService) *QrcodeHandler {
	return &QrcodeHandler{
		qrcodeService: qrcodeService,
	}
}

// GetQrcodes 获取收款码列表
func (h *QrcodeHandler) GetQrcodes(c *gin.Context) {
	var req model.QrcodeListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 设置默认分页参数
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	// 应用数据隔离：根据用户角色自动设置用户过滤
	middleware.ApplyUserFilter(c, &req.User_id)

	// 获取收款码列表
	qrcodes, total, err := h.qrcodeService.GetQrcodes(&req)
	if err != nil {
		response.InternalError(c, "Failed to get qrcodes")
		return
	}

	// 构建响应
	resp := map[string]interface{}{
		"data":  qrcodes,
		"total": total,
		"page":  req.Page,
		"limit": req.Limit,
	}

	response.Success(c, resp)
}

// CreateQrcode 创建收款码
func (h *QrcodeHandler) CreateQrcode(c *gin.Context) {
	var req model.CreateQrcodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 获取当前用户ID
	userID := middleware.GetCurrentUserID(c)

	// 创建收款码
	qrcode, err := h.qrcodeService.CreateQrcode(userID, &req)
	if err != nil {
		if err == service.ErrQrcodeExists {
			response.Error(c, response.CodeBadRequest, "Qrcode already exists")
			return
		}
		response.InternalError(c, "Failed to create qrcode")
		return
	}

	response.Success(c, qrcode)
}

// DeleteQrcode 删除收款码
func (h *QrcodeHandler) DeleteQrcode(c *gin.Context) {
	// 获取收款码ID
	qrcodeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid qrcode ID")
		return
	}

	// 检查收款码访问权限（包含数据隔离）
	qrcode, hasAccess := h.checkQrcodeAccess(c, uint(qrcodeID))
	if !hasAccess {
		return
	}

	// 删除收款码
	err = h.qrcodeService.DeleteQrcode(qrcode.Id)
	if err != nil {
		response.InternalError(c, "Failed to delete qrcode")
		return
	}

	response.Success(c, nil)
}

// ParseQrcode 解析收款码
func (h *QrcodeHandler) ParseQrcode(c *gin.Context) {
	var req model.ParseQrcodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 解析收款码
	result, err := h.qrcodeService.ParseQrcode(&req)
	if err != nil {
		response.Error(c, response.CodeBadRequest, "Failed to parse qrcode: "+err.Error())
		return
	}

	response.Success(c, result)
}

// GenerateQrcode 生成二维码
// @Summary 生成二维码
// @Description 生成二维码图片，直接返回PNG图片数据
// @Tags qrcode
// @Produce image/png
// @Param url query string true "二维码内容URL"
// @Param size query int false "二维码尺寸" default(300)
// @Success 200 {file} binary "PNG图片数据"
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v2/qrcode/generate [get]
func (h *QrcodeHandler) GenerateQrcode(c *gin.Context) {
	var req model.GenerateQrcodeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 生成二维码
	result, err := h.qrcodeService.GenerateQrcode(&req)
	if err != nil {
		response.Error(c, response.CodeBadRequest, "Failed to generate qrcode: "+err.Error())
		return
	}

	// 从base64数据URL中提取PNG数据
	if result.Qrcode_url != "" && len(result.Qrcode_url) > 22 { // "data:image/png;base64," 长度为22
		base64Data := result.Qrcode_url[22:] // 去掉 "data:image/png;base64," 前缀

		// 解码base64数据
		imageData, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			response.Error(c, response.CodeInternalError, "Failed to decode qrcode image")
			return
		}

		// 设置响应头并返回图片数据
		c.Header("Content-Type", "image/png")
		c.Header("Content-Length", fmt.Sprintf("%d", len(imageData)))
		c.Data(200, "image/png", imageData)
		return
	}

	response.Error(c, response.CodeInternalError, "Invalid qrcode data")
}

// UpdateQrcodeStatus 更新收款码状态
// @Summary 更新收款码状态
// @Description 更新收款码的启用/禁用状态
// @Tags qrcode
// @Accept json
// @Produce json
// @Param id path int true "收款码ID"
// @Param request body model.UpdateQrcodeStatusRequest true "状态更新请求"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v2/qrcodes/{id}/status [put]
func (h *QrcodeHandler) UpdateQrcodeStatus(c *gin.Context) {
	// 获取收款码ID
	qrcodeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationFailed(c, "Invalid qrcode ID")
		return
	}

	// 检查权限
	qrcode, hasAccess := h.checkQrcodeAccess(c, uint(qrcodeID))
	if !hasAccess {
		return
	}
	if qrcode == nil {
		response.NotFound(c, "Qrcode not found")
		return
	}

	// 解析请求
	var req model.UpdateQrcodeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, fmt.Sprintf("Failed to bind JSON: %s", err.Error()))
		return
	}

	// 添加调试日志
	fmt.Printf("Received status update request: qrcodeID=%d, state=%d\n", qrcodeID, req.State)

	// 更新状态
	err = h.qrcodeService.UpdateQrcodeStatus(uint(qrcodeID), *req.State)
	if err != nil {
		response.InternalError(c, "Failed to update qrcode status")
		return
	}

	response.Success(c, nil)
}

// checkQrcodeAccess 检查用户是否有权限访问指定收款码
func (h *QrcodeHandler) checkQrcodeAccess(c *gin.Context, qrcodeID uint) (*model.PayQrcode, bool) {
	// 获取收款码信息
	qrcode, err := h.qrcodeService.GetQrcodeByID(qrcodeID)
	if err != nil {
		if err == service.ErrQrcodeNotFound {
			response.Error(c, response.CodeNotFound, "Qrcode not found")
		} else {
			response.InternalError(c, "Failed to get qrcode")
		}
		return nil, false
	}

	// 检查数据访问权限
	if !middleware.ValidateResourceAccess(c, qrcode.User_id) {
		response.Forbidden(c, "Access denied: insufficient permissions")
		return nil, false
	}

	return qrcode, true
}
