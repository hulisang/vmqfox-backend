package handler

import (
	"log"
	"vmqfox-api-go/internal/middleware"
	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/service"
	"vmqfox-api-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// SettingHandler 系统设置处理器
type SettingHandler struct {
	settingService service.SettingService
}

// NewSettingHandler 创建系统设置处理器
func NewSettingHandler(settingService service.SettingService) *SettingHandler {
	return &SettingHandler{
		settingService: settingService,
	}
}

// GetSystemConfig 获取系统配置
// @Summary 获取系统配置
// @Description 获取系统配置信息
// @Tags settings
// @Produce json
// @Success 200 {object} response.Response{data=model.SystemConfigResponse}
// @Failure 500 {object} response.Response
// @Router /api/v2/settings [get]
func (h *SettingHandler) GetSystemConfig(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)

	config, err := h.settingService.GetSystemConfig(userID)
	if err != nil {
		response.InternalError(c, "Failed to get system config")
		return
	}

	response.Success(c, config)
}

// UpdateSystemConfig 更新系统配置
// @Summary 更新系统配置
// @Description 更新系统配置信息
// @Tags settings
// @Accept json
// @Produce json
// @Param config body model.SystemConfigRequest true "系统配置"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v2/settings [post]
func (h *SettingHandler) UpdateSystemConfig(c *gin.Context) {
	var req model.SystemConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	userID := middleware.GetCurrentUserID(c)

	err := h.settingService.UpdateSystemConfig(userID, &req)
	if err != nil {
		response.InternalError(c, "Failed to update system config")
		return
	}

	response.Success(c, nil)
}

// GetSystemStatus 获取系统状态
// @Summary 获取系统状态
// @Description 获取系统运行状态和统计信息
// @Tags settings
// @Produce json
// @Success 200 {object} response.Response{data=model.SystemStatusResponse}
// @Failure 500 {object} response.Response
// @Router /api/v2/system/status [get]
func (h *SettingHandler) GetSystemStatus(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)

	status, err := h.settingService.GetSystemStatus(userID)
	if err != nil {
		response.InternalError(c, "Failed to get system status")
		return
	}

	response.Success(c, status)
}

// GetGlobalSystemStatus 获取全局系统状态
// @Summary 获取全局系统状态
// @Description 获取所有用户的汇总统计数据，只有超级管理员可以访问
// @Tags 系统管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=model.SystemStatusResponse}
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v2/system/global-status [get]
func (h *SettingHandler) GetGlobalSystemStatus(c *gin.Context) {
	status, err := h.settingService.GetGlobalSystemStatus()
	if err != nil {
		response.InternalError(c, "Failed to get global system status")
		return
	}

	response.Success(c, status)
}

// GetDashboard 获取仪表板数据
// @Summary 获取仪表板数据
// @Description 获取管理后台仪表板数据
// @Tags dashboard
// @Produce json
// @Success 200 {object} response.Response{data=model.DashboardResponse}
// @Failure 500 {object} response.Response
// @Router /api/v2/dashboard [get]
func (h *SettingHandler) GetDashboard(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)

	dashboard, err := h.settingService.GetDashboard(userID)
	if err != nil {
		response.InternalError(c, "Failed to get dashboard data")
		return
	}

	response.Success(c, dashboard)
}

// GetMonitorConfig 获取监控配置
// @Summary 获取监控配置
// @Description 获取监控配置信息
// @Tags settings
// @Produce json
// @Success 200 {object} response.Response{data=model.MonitorConfigResponse}
// @Failure 500 {object} response.Response
// @Router /api/v2/settings/monitor [get]
func (h *SettingHandler) GetMonitorConfig(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)

	config, err := h.settingService.GetMonitorConfig(userID)
	if err != nil {
		response.InternalError(c, "Failed to get monitor config")
		return
	}

	response.Success(c, config)
}

// UpdateMonitorConfig 更新监控配置
// @Summary 更新监控配置
// @Description 更新监控配置信息
// @Tags settings
// @Accept json
// @Produce json
// @Param config body model.MonitorConfigRequest true "监控配置"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v2/settings/monitor [put]
func (h *SettingHandler) UpdateMonitorConfig(c *gin.Context) {
	var req model.MonitorConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	userID := middleware.GetCurrentUserID(c)

	err := h.settingService.UpdateMonitorConfig(userID, &req)
	if err != nil {
		response.InternalError(c, "Failed to update monitor config")
		return
	}

	response.Success(c, nil)
}

// MonitorHeart 监控心跳
// @Summary 监控心跳
// @Description 处理监控端心跳请求，支持GET和POST请求
// @Tags monitor
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param t query string true "时间戳"
// @Param sign query string true "签名"
// @Param appid query string false "AppID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v2/monitor/heart [post]
// @Router /api/v2/monitor/heart [get]
func (h *SettingHandler) MonitorHeart(c *gin.Context) {
	var req model.MonitorHeartRequest

	// 兼容GET和POST请求，优先从查询参数获取
	req.T = c.Query("t")
	req.Sign = c.Query("sign")
	req.AppID = c.Query("appid")

	// 如果查询参数为空，尝试从表单数据获取
	if req.T == "" || req.Sign == "" {
		if err := c.ShouldBind(&req); err != nil {
			response.ValidationFailed(c, err.Error())
			return
		}
	}

	// 验证必需参数
	if req.T == "" || req.Sign == "" {
		log.Printf("心跳参数验证失败: t=%s, sign=%s, appid=%s", req.T, req.Sign, req.AppID)
		response.ValidationFailed(c, "Missing required parameters: t and sign")
		return
	}

	log.Printf("收到心跳请求: t=%s, sign=%s, appid=%s", req.T, req.Sign, req.AppID)

	err := h.settingService.ProcessMonitorHeart(&req)
	if err != nil {
		if err == service.ErrInvalidSign {
			response.Error(c, response.CodeUnauthorized, "Invalid signature")
			return
		}
		response.InternalError(c, "Failed to process monitor heart")
		return
	}

	response.Success(c, nil)
}

// MonitorPush 监控推送
// @Summary 监控推送
// @Description 处理监控端推送请求
// @Tags monitor
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param push formData model.MonitorPushRequest true "推送参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v2/monitor/push [post]
func (h *SettingHandler) MonitorPush(c *gin.Context) {
	var req model.MonitorPushRequest
	if err := c.ShouldBind(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	err := h.settingService.ProcessMonitorPush(&req)
	if err != nil {
		if err == service.ErrInvalidSign {
			response.Error(c, response.CodeUnauthorized, "Invalid signature")
			return
		}
		response.InternalError(c, "Failed to process monitor push")
		return
	}

	response.Success(c, nil)
}

// GetSystemInfo 获取系统信息
// @Summary 获取系统信息
// @Description 获取系统版本和运行信息
// @Tags system
// @Produce json
// @Success 200 {object} response.Response{data=model.SystemInfoResponse}
// @Failure 500 {object} response.Response
// @Router /api/v2/system/info [get]
func (h *SettingHandler) GetSystemInfo(c *gin.Context) {
	info, err := h.settingService.GetSystemInfo()
	if err != nil {
		response.InternalError(c, "Failed to get system info")
		return
	}

	response.Success(c, info)
}

// CheckUpdate 检查更新
// @Summary 检查更新
// @Description 检查系统更新
// @Tags system
// @Produce json
// @Param check query model.UpdateSystemRequest false "检查参数"
// @Success 200 {object} response.Response{data=model.UpdateSystemResponse}
// @Failure 500 {object} response.Response
// @Router /api/v2/system/update [get]
func (h *SettingHandler) CheckUpdate(c *gin.Context) {
	var req model.UpdateSystemRequest
	c.ShouldBindQuery(&req)

	update, err := h.settingService.CheckUpdate(&req)
	if err != nil {
		response.InternalError(c, "Failed to check update")
		return
	}

	response.Success(c, update)
}

// GetIPInfo 获取IP信息
// @Summary 获取IP信息
// @Description 获取服务器IP信息
// @Tags system
// @Produce json
// @Success 200 {object} response.Response{data=model.IPInfoResponse}
// @Failure 500 {object} response.Response
// @Router /api/v2/system/ip [get]
func (h *SettingHandler) GetIPInfo(c *gin.Context) {
	info, err := h.settingService.GetIPInfo()
	if err != nil {
		response.InternalError(c, "Failed to get IP info")
		return
	}

	response.Success(c, info)
}
