package handler

import (
	"vmqfox-api-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// MenuHandler 菜单处理器
type MenuHandler struct{}

// NewMenuHandler 创建菜单处理器
func NewMenuHandler() *MenuHandler {
	return &MenuHandler{}
}

// MenuItem 菜单项结构
type MenuItem struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Icon      string     `json:"icon"`
	Component string     `json:"component"`
	Meta      MenuMeta   `json:"meta"`
	Children  []MenuItem `json:"children,omitempty"`
}

// MenuMeta 菜单元数据
type MenuMeta struct {
	Title    string `json:"title"`
	Icon     string `json:"icon"`
	InLayout bool   `json:"inLayout,omitempty"`
}

// GetMenu 获取菜单数据
// @Summary 获取菜单数据
// @Description 根据用户角色获取管理后台的菜单结构
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]MenuItem} "菜单数据"
// @Failure 401 {object} response.Response "未授权"
// @Router /api/v2/menu [get]
func (h *MenuHandler) GetMenu(c *gin.Context) {
	// 获取当前用户角色
	userRole, exists := c.Get("user_role")
	if !exists {
		response.Error(c, 401, "未授权访问")
		return
	}

	role, ok := userRole.(string)
	if !ok {
		response.Error(c, 401, "用户角色信息无效")
		return
	}

	// 构建基础菜单数据
	baseMenu := []MenuItem{
		{
			ID:        1,
			Name:      "首页",
			Path:      "/dashboard",
			Icon:      "&#xe6cc;",
			Component: "/vmq/dashboard/index",
			Meta: MenuMeta{
				Title: "首页",
				Icon:  "&#xe6cc;",
			},
		},
		{
			ID:        2,
			Name:      "订单管理",
			Path:      "/orderlist",
			Icon:      "&#xe70f;",
			Component: "/vmq/orderlist/index",
			Meta: MenuMeta{
				Title: "订单管理",
				Icon:  "&#xe70f;",
			},
		},
		{
			ID:        3,
			Name:      "微信收款码",
			Path:      "/wxqrcode",
			Icon:      "&#xe7f3;",
			Component: "",
			Meta: MenuMeta{
				Title: "微信收款码",
				Icon:  "&#xe7f3;",
			},
			Children: []MenuItem{
				{
					ID:        31,
					Name:      "添加微信收款码",
					Path:      "add",
					Component: "/vmq/wxqrcode/add/index",
					Meta: MenuMeta{
						Title:    "添加收款码",
						Icon:     "&#xe717;",
						InLayout: true,
					},
				},
				{
					ID:        32,
					Name:      "管理微信收款码",
					Path:      "manage",
					Component: "/vmq/wxqrcode/manage/index",
					Meta: MenuMeta{
						Title:    "管理收款码",
						Icon:     "&#xe6d6;",
						InLayout: true,
					},
				},
			},
		},
		{
			ID:        4,
			Name:      "支付宝收款码",
			Path:      "/zfbqrcode",
			Icon:      "&#xe7da;",
			Component: "",
			Meta: MenuMeta{
				Title: "支付宝收款码",
				Icon:  "&#xe7da;",
			},
			Children: []MenuItem{
				{
					ID:        41,
					Name:      "添加支付宝收款码",
					Path:      "add",
					Component: "/vmq/zfbqrcode/add/index",
					Meta: MenuMeta{
						Title:    "添加收款码",
						Icon:     "&#xe717;",
						InLayout: true,
					},
				},
				{
					ID:        42,
					Name:      "管理支付宝收款码",
					Path:      "manage",
					Component: "/vmq/zfbqrcode/manage/index",
					Meta: MenuMeta{
						Title:    "管理收款码",
						Icon:     "&#xe6d6;",
						InLayout: true,
					},
				},
			},
		},
		{
			ID:        5,
			Name:      "系统设置",
			Path:      "/systemSettings",
			Icon:      "&#xe6d0;",
			Component: "/vmq/systemSettings/index",
			Meta: MenuMeta{
				Title: "系统设置",
				Icon:  "&#xe6d0;",
			},
		},
		{
			ID:        6,
			Name:      "监控端设置",
			Path:      "/monitorSettings",
			Icon:      "&#xe81d;",
			Component: "/vmq/monitorSettings/index",
			Meta: MenuMeta{
				Title: "监控端设置",
				Icon:  "&#xe81d;",
			},
		},
		{
			ID:        7,
			Name:      "API文档",
			Path:      "/api-docs",
			Icon:      "&#xe705;",
			Component: "/vmq/api/index",
			Meta: MenuMeta{
				Title: "API文档",
				Icon:  "&#xe705;",
			},
		},
	}

	// 根据用户角色过滤菜单
	var menu []MenuItem
	if role == "super_admin" {
		// 超级管理员可以看到所有菜单，包括用户管理
		userManagementMenu := MenuItem{
			ID:        8,
			Name:      "UserManagement",
			Path:      "/userManagement",
			Icon:      "&#xe608;",
			Component: "/system/user/index",
			Meta: MenuMeta{
				Title:    "用户管理",
				Icon:     "&#xe608;",
				InLayout: true,
			},
		}

		// 在系统设置之后插入用户管理菜单
		menu = make([]MenuItem, 0, len(baseMenu)+1)
		for _, item := range baseMenu {
			menu = append(menu, item)
			if item.ID == 4 { // 系统设置之后
				menu = append(menu, userManagementMenu)
			}
		}
	} else {
		// 普通admin用户只能看到基础菜单，不包括用户管理
		menu = baseMenu
	}

	response.Success(c, menu)
}
