import axios from 'axios'
import { toast } from 'sonner'
import { useAuthStore } from '../stores'

export const api = axios.create({
  baseURL: '/api',
  timeout: 15000,
})

// 请求拦截器注入 Token
api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器统一处理错误
api.interceptors.response.use(
  (response) => {
    const data = response.data
    // 如果返回的标准结构 code != 200 且不是公开查询的特定情况
    if (data && typeof data === 'object' && 'code' in data) {
      if (data.code !== 200 && data.code !== 1) {
        const msg = data.msg || data.message || '请求失败'
        // 如果是未授权
        if (data.code === 401 || data.code === 403) {
          useAuthStore.getState().clearAuth()
        }
        return Promise.reject(new Error(msg))
      }
    }
    return response
  },
  (error) => {
    const status = error.response?.status
    const msg = error.response?.data?.msg || error.response?.data?.message || error.message || '网络连接异常'
    if (status === 401) {
      useAuthStore.getState().clearAuth()
      toast.error('登录已过期，请重新登录')
    }
    return Promise.reject(new Error(msg))
  }
)
