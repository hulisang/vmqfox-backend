import { api } from './client'

export interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

// 认证模块
export interface LoginResponse {
  accessToken: string
  username: string
  expiresAt: number
}

export const authApi = {
  login: async (params: { username: string; password: string }) => {
    const res = await api.post<ApiResponse<LoginResponse>>('/auth/login', params)
    return res.data.data
  },
  logout: async () => {
    const res = await api.post<ApiResponse<void>>('/auth/logout')
    return res.data
  },
}

/**
 * 仪表盘统计。
 * 字段名与 Go `orderStatisticsAPIHandler` 完全一致：
 * todayOrder / todaySuccessOrder / todayCloseOrder / todayMoney / countOrder / countMoney。
 * 后端不提供按渠道拆分的金额，因此这里也不声明渠道字段，避免前端展示恒为 0 的伪数据。
 */
export interface DashboardStats {
  todayOrder: number
  todaySuccessOrder: number
  todayCloseOrder: number
  todayMoney: string
  countOrder: number
  countMoney: string
}

export interface SystemConfig {
  phpOs: string
  server: string
  phpVersion: string
  mysqlVersion: string
  thinkphpVersion: string
  gdInfo: string
  appVersion: string
  runTime: string
}

export const dashboardApi = {
  getStats: async () => {
    const res = await api.get<ApiResponse<DashboardStats>>('/config/status')
    return res.data.data
  },
  getSystemConfig: async () => {
    const res = await api.get<ApiResponse<SystemConfig>>('/config/get')
    return res.data.data
  },
}

// 订单管理模块（管理端列表使用后端 snake_case 字段）
export interface OrderItem {
  id: number
  order_id: string
  pay_id: string
  type: number
  type_text: string
  price: string
  really_price: string
  state: number
  state_text: string
  create_time: string
  create_date: number
  pay_date: number
  close_date: number
  param?: string
}

export interface OrderListResponse {
  items: OrderItem[]
  total: number
}

export const orderApi = {
  list: async (params: { page: number; limit: number; state?: number }) => {
    const res = await api.get<ApiResponse<OrderListResponse>>('/order/list', { params })
    return res.data.data
  },
  reissue: async (id: number) => {
    const res = await api.post<ApiResponse<void>>(`/order/reissue/${id}`)
    return res.data
  },
  delete: async (id: number) => {
    const res = await api.delete<ApiResponse<void>>(`/order/${id}`)
    return res.data
  },
  closeExpired: async () => {
    const res = await api.post<ApiResponse<{ count: number }>>('/order/expired')
    return res.data.data
  },
  deleteExpired: async () => {
    const res = await api.delete<ApiResponse<{ count: number }>>('/order/expired')
    return res.data.data
  },
  deleteLast: async () => {
    // 后端 /order/last 固定清理 24 小时前的订单，不接受 days 参数
    const res = await api.delete<ApiResponse<{ count: number }>>('/order/last')
    return res.data.data
  },
}

// 二维码管理模块
export interface QrcodeItem {
  id: number
  pay_url: string
  price: string
  type: number
  type_text: string
  state: number
  state_text: string
}

export interface QrcodeListResponse {
  items: QrcodeItem[]
  total: number
}

export const qrcodeApi = {
  list: async (params: { type: 'wechat' | 'alipay'; page: number; limit: number }) => {
    const url = params.type === 'wechat' ? '/qrcode/wechat' : '/qrcode/alipay'
    const res = await api.get<ApiResponse<QrcodeListResponse>>(url, {
      params: { page: params.page, limit: params.limit },
    })
    return res.data.data
  },
  create: async (data: { type: number; payUrl: string; price: string }) => {
    const url = data.type === 1 ? '/qrcode/wechat' : '/qrcode/alipay'
    // 后端 createQRCodeHandler 读取 pay_url，此处必须使用同名字段
    const res = await api.post<ApiResponse<void>>(url, {
      pay_url: data.payUrl,
      price: data.price,
    })
    return res.data
  },
  setState: async (id: number, state: number) => {
    const res = await api.post<ApiResponse<void>>(`/qrcode/bind/${id}`, { state })
    return res.data
  },
  delete: async (type: 'wechat' | 'alipay', id: number) => {
    const url = type === 'wechat' ? `/qrcode/wechat/${id}` : `/qrcode/alipay/${id}`
    const res = await api.delete<ApiResponse<void>>(url)
    return res.data
  },
}

// 监控端模块
// jkstate 是监控端在线状态（-1=未绑定、0=已掉线、1=运行正常），由后端心跳与生命周期任务维护，并非总开关。
export interface MonitorSettings {
  jkstate: string
  lastheart: string
  lastpay: string
}

export const monitorApi = {
  get: async () => {
    const res = await api.get<ApiResponse<MonitorSettings>>('/config/monitor')
    return res.data.data
  },
  // 手动覆盖监控状态，仅为兼容旧客户端保留；界面状态展示一律以 get 返回的 jkstate 为准。
  update: async (data: { jkstate: string }) => {
    const res = await api.post<ApiResponse<void>>('/config/monitor', data)
    return res.data
  },
}

// 系统设置模块
export interface SystemSettings {
  user: string
  pass?: string
  notifyUrl: string
  returnUrl: string
  key: string
  close: string
  payQf: string
  wxpay: string
  zfbpay: string
}

export const settingsApi = {
  get: async () => {
    const res = await api.get<ApiResponse<SystemSettings>>('/config/settings')
    return res.data.data
  },
  update: async (data: Partial<SystemSettings>) => {
    const res = await api.post<ApiResponse<void>>('/config/settings', data)
    return res.data
  },
}

/**
 * 公开收银台订单视图。
 * 与 Go `PublicOrderView` 严格一一对应：不含 orderId、param、notifyUrl、returnUrl 与独立签名字段。
 * 回跳地址必须在支付成功后单独调用 getReturnUrl 获取。
 */
export interface PublicOrderInfo {
  payId: string
  payType: number
  price: string
  reallyPrice: string
  payUrl: string
  isAuto: number
  state: number
  stateText: string
  timeOut: number
  date: number
  remainingSeconds: number
}

export interface CreateOrderResult {
  payId: string
  orderId: string
  publicToken: string
  payType: number
  price: string
  reallyPrice: string
  payUrl: string
  isAuto: number
  redirectUrl: string
}

export const publicPaymentApi = {
  /** 公开令牌是 bearer 凭据，必须走路径参数，与 Go 路由 /order/get/:id 对齐 */
  getOrder: async (publicToken: string) => {
    const res = await api.get<ApiResponse<PublicOrderInfo>>(
      `/order/get/${encodeURIComponent(publicToken)}`
    )
    return res.data.data
  },
  checkOrder: async (publicToken: string) => {
    const res = await api.get<ApiResponse<{ state: number; remainingSeconds: number }>>(
      `/order/check/${encodeURIComponent(publicToken)}`
    )
    return res.data.data
  },
  /**
   * 获取服务端生成并签名的商户回跳地址。
   * 后端仅在订单已支付时才返回，未支付会给出明确业务错误。
   */
  getReturnUrl: async (publicToken: string) => {
    const res = await api.get<ApiResponse<{ returnUrl: string; mode: string }>>(
      `/order/return-url/${encodeURIComponent(publicToken)}`
    )
    return res.data.data
  },
  createTestOrder: async (data: {
    payId: string
    type: number
    price: string
    sign: string
    param?: string
    notifyUrl?: string
    returnUrl?: string
  }) => {
    const res = await api.post<ApiResponse<CreateOrderResult>>('/order/create', data)
    return res.data.data
  },
}
