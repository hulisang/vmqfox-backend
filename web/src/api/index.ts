import { api } from './client'

export interface ApiResponse<T = any> {
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

// 仪表盘统计与系统环境
export interface DashboardStats {
  todayOrderCount: number
  todaySuccessCount: number
  todayMoney: number
  totalOrderCount: number
  totalSuccessCount: number
  totalMoney: number
  wechatMoney: number
  alipayMoney: number
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

// 订单管理模块
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
  pay_time?: string
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
  deleteLast: async (days: number) => {
    const res = await api.delete<ApiResponse<{ count: number }>>('/order/last', { params: { days } })
    return res.data.data
  },
}

// 二维码管理模块
export interface QrcodeItem {
  id: number
  pay_url: string
  price: string
  type: number
  state: number
  state_text?: string
  create_date?: string
}

export interface QrcodeListResponse {
  items: QrcodeItem[]
  total: number
}

export const qrcodeApi = {
  list: async (params: { type: 'wechat' | 'alipay'; page: number; limit: number }) => {
    const url = params.type === 'wechat' ? '/qrcode/wechat' : '/qrcode/alipay'
    const res = await api.get<ApiResponse<QrcodeListResponse>>(url, { params: { page: params.page, limit: params.limit } })
    return res.data.data
  },
  create: async (data: { type: number; payUrl: string; price: string }) => {
    const url = data.type === 1 ? '/qrcode/wechat' : '/qrcode/alipay'
    const res = await api.post<ApiResponse<QrcodeItem>>(url, data)
    return res.data.data
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

// 公开收银台模块
export interface PublicOrderInfo {
  payId: string
  orderId?: string
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
  returnUrl?: string
}

export const publicPaymentApi = {
  getOrder: async (publicToken: string) => {
    const res = await api.get<ApiResponse<PublicOrderInfo>>('/order/get', { params: { publicToken } })
    return res.data.data
  },
  checkOrder: async (publicToken: string) => {
    const res = await api.get<ApiResponse<{ state: number; remainingSeconds: number }>>('/order/check', { params: { publicToken } })
    return res.data.data
  },
  createTestOrder: async (data: { payId: string; type: number; price: string; sign: string; param?: string }) => {
    const res = await api.post<ApiResponse<{ publicToken: string; payUrl: string; reallyPrice: string }>>('/order/create', data)
    return res.data.data
  },
}
