package model

import "time"

// Setting 系统设置模型
type Setting struct {
	Vkey    string `json:"vkey" gorm:"primaryKey;size:255;column:vkey"`
	User_id uint   `json:"user_id" gorm:"primaryKey;default:1;column:user_id"`
	Vvalue  string `json:"vvalue" gorm:"type:text;column:vvalue"`
}

// TableName 指定表名
func (Setting) TableName() string {
	return "setting"
}

// ==================== 系统配置相关请求和响应结构 ====================

// SystemConfigRequest 系统配置请求
type SystemConfigRequest struct {
	User      string `json:"user" binding:"required"`
	Pass      string `json:"pass" binding:"required"`
	NotifyUrl string `json:"notifyUrl" binding:"required,url"`
	ReturnUrl string `json:"returnUrl" binding:"required,url"`
	Key       string `json:"key"`
	AppId     string `json:"appId" binding:"required,min=6,max=32"` // 新增AppID字段
	Close     string `json:"close"`
	PayQf     string `json:"payQf"`
	Wxpay     string `json:"wxpay"`
	Zfbpay    string `json:"zfbpay"`
	// 注册配置字段
	RegisterEnabled         string `json:"register_enabled"`          // 是否开放注册 "1"/"0"
	RegisterDefaultRole     string `json:"register_default_role"`     // 默认角色
	RegisterRequireApproval string `json:"register_require_approval"` // 是否需要审核 "1"/"0"
	RegisterRateLimit       string `json:"register_rate_limit"`       // 频率限制
}

// SystemConfigResponse 系统配置响应
type SystemConfigResponse struct {
	User      string `json:"user"`
	Pass      string `json:"pass"`
	NotifyUrl string `json:"notifyUrl"`
	ReturnUrl string `json:"returnUrl"`
	Key       string `json:"key"`
	AppId     string `json:"appId"` // 新增AppID字段
	Lastheart string `json:"lastheart"`
	Lastpay   string `json:"lastpay"`
	Jkstate   string `json:"jkstate"`
	Close     string `json:"close"`
	PayQf     string `json:"payQf"`
	Wxpay     string `json:"wxpay"`
	Zfbpay    string `json:"zfbpay"`
	// 注册配置字段
	RegisterEnabled         string `json:"register_enabled"`          // 是否开放注册 "1"/"0"
	RegisterDefaultRole     string `json:"register_default_role"`     // 默认角色
	RegisterRequireApproval string `json:"register_require_approval"` // 是否需要审核 "1"/"0"
	RegisterRateLimit       string `json:"register_rate_limit"`       // 频率限制
}

// SystemStatusResponse 系统状态响应
type SystemStatusResponse struct {
	TodayOrder        int64   `json:"todayOrder"`
	TodaySuccessOrder int64   `json:"todaySuccessOrder"`
	TodayCloseOrder   int64   `json:"todayCloseOrder"`
	TodayMoney        float64 `json:"todayMoney"`
	CountOrder        int64   `json:"countOrder"`
	CountMoney        float64 `json:"countMoney"`
	Lastheart         string  `json:"lastheart"`
	Lastpay           string  `json:"lastpay"`
	Jkstate           string  `json:"jkstate"`
	MonitorStatus     int     `json:"monitorStatus"` // 监控状态：0-未知 1-正常 2-异常
	LastHeartTime     string  `json:"lastHeartTime"` // 格式化的最后心跳时间
	LastPayTime       string  `json:"lastPayTime"`   // 格式化的最后支付时间
}

// DashboardResponse 仪表板数据响应
type DashboardResponse struct {
	TodayOrder        int64   `json:"todayOrder"`
	TodaySuccessOrder int64   `json:"todaySuccessOrder"`
	TodayCloseOrder   int64   `json:"todayCloseOrder"`
	TodayMoney        float64 `json:"todayMoney"`
	CountOrder        int64   `json:"countOrder"`
	CountMoney        float64 `json:"countMoney"`
	PHPVersion        string  `json:"phpVersion"`
	PHPOS             string  `json:"phpOs"`
	Server            string  `json:"server"`
	MySQL             string  `json:"mysql"`
	Thinkphp          string  `json:"thinkphp"`
	RunTime           string  `json:"runTime"`
	Ver               string  `json:"ver"`
	GD                string  `json:"gd"`
}

// MonitorConfigRequest 监控配置请求
type MonitorConfigRequest struct {
	Jk string `json:"jk" binding:"required,oneof=0 1"`
}

// MonitorConfigResponse 监控配置响应
type MonitorConfigResponse struct {
	Jkstate   string `json:"jkstate"`
	Lastheart string `json:"lastheart"`
	Lastpay   string `json:"lastpay"`
}

// MonitorHeartRequest 监控心跳请求
type MonitorHeartRequest struct {
	T     string `form:"t" binding:"required"`
	Sign  string `form:"sign" binding:"required"`
	AppID string `form:"appid"` // 可选，用于多用户系统
}

// MonitorPushRequest 监控推送请求
type MonitorPushRequest struct {
	T     string `form:"t" binding:"required"`
	Sign  string `form:"sign" binding:"required"`
	Type  string `form:"type" binding:"required"`
	Price string `form:"price" binding:"required"`
	AppID string `form:"appid"` // 可选，用于多用户系统
}

// SystemInfoResponse 系统信息响应
type SystemInfoResponse struct {
	GoVersion    string    `json:"goVersion"`
	GOOS         string    `json:"goOs"`
	Server       string    `json:"server"`
	MySQLVersion string    `json:"mysqlVersion"`
	AppVersion   string    `json:"appVersion"`
	RunTime      string    `json:"runTime"`
	StartTime    time.Time `json:"startTime"`
	MemoryUsage  string    `json:"memoryUsage"`
	GoroutineNum int       `json:"goroutineNum"`
}

// UpdateSystemRequest 更新系统信息请求
type UpdateSystemRequest struct {
	CheckUpdate bool `form:"check_update"`
}

// UpdateSystemResponse 更新系统信息响应
type UpdateSystemResponse struct {
	HasUpdate      bool   `json:"hasUpdate"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	UpdateUrl      string `json:"updateUrl"`
	UpdateLog      string `json:"updateLog"`
}

// IPInfoResponse IP信息响应
type IPInfoResponse struct {
	IP       string `json:"ip"`
	Country  string `json:"country"`
	Region   string `json:"region"`
	City     string `json:"city"`
	ISP      string `json:"isp"`
	Location string `json:"location"`
}
