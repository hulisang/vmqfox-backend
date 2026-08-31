import axios, { AxiosError, AxiosResponse } from 'axios'
import { toast } from 'sonner'
import { useAuthStore } from '../stores'

/**
 * ApiError 是前端统一的错误载体。
 * 同时保留 HTTP 状态码、后端业务码、requestId 与 Retry-After，
 * 避免把这些排障与限流信息在拦截器里丢弃，只剩一句文案。
 */
export class ApiError extends Error {
  /** HTTP 状态码；网络层失败时为 null */
  readonly status: number | null
  /** 后端 envelope 中的业务码 */
  readonly code: number | null
  /** 服务端返回的 X-Request-ID，用于对齐后端访问日志 */
  readonly requestId: string | null
  /** 被限流时的建议等待秒数 */
  readonly retryAfterSeconds: number | null

  constructor(
    message: string,
    options: {
      status?: number | null
      code?: number | null
      requestId?: string | null
      retryAfterSeconds?: number | null
    } = {}
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = options.status ?? null
    this.code = options.code ?? null
    this.requestId = options.requestId ?? null
    this.retryAfterSeconds = options.retryAfterSeconds ?? null
  }

  /** 是否为限流错误，供 React Query 决定是否退避重试 */
  get isRateLimited(): boolean {
    return this.status === 429 || this.code === 429
  }
}

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

/** 从响应头读取 Retry-After；只接受秒数形式，非法值返回 null 而不是猜测 */
function readRetryAfter(headers: unknown): number | null {
  if (!headers || typeof headers !== 'object') return null
  const raw = (headers as Record<string, unknown>)['retry-after']
  if (typeof raw !== 'string' && typeof raw !== 'number') return null
  const seconds = Number(raw)
  return Number.isFinite(seconds) && seconds >= 0 ? seconds : null
}

function readRequestId(headers: unknown): string | null {
  if (!headers || typeof headers !== 'object') return null
  const raw = (headers as Record<string, unknown>)['x-request-id']
  return typeof raw === 'string' && raw !== '' ? raw : null
}

/** 后端 envelope 结构；code 为 200 或 1 时视为业务成功 */
interface BackendEnvelope {
  code?: number
  msg?: string
  message?: string
}

// 响应拦截器：把 HTTP 层与业务层错误统一收敛为 ApiError
api.interceptors.response.use(
  (response: AxiosResponse) => {
    const data = response.data as BackendEnvelope | undefined
    if (data && typeof data === 'object' && typeof data.code === 'number') {
      if (data.code !== 200 && data.code !== 1) {
        // 401/403 一律安全退出，避免用过期 Token 反复触发限流
        if (data.code === 401 || data.code === 403) {
          useAuthStore.getState().clearAuth()
        }
        return Promise.reject(
          new ApiError(data.msg || data.message || '请求失败', {
            status: response.status,
            code: data.code,
            requestId: readRequestId(response.headers),
            retryAfterSeconds: readRetryAfter(response.headers),
          })
        )
      }
    }
    return response
  },
  (error: AxiosError<BackendEnvelope>) => {
    const status = error.response?.status ?? null
    const body = error.response?.data
    const message =
      body?.msg || body?.message || error.message || '网络连接异常'

    if (status === 401) {
      useAuthStore.getState().clearAuth()
      toast.error('登录已过期，请重新登录')
    }

    return Promise.reject(
      new ApiError(message, {
        status,
        code: typeof body?.code === 'number' ? body.code : null,
        requestId: readRequestId(error.response?.headers),
        retryAfterSeconds: readRetryAfter(error.response?.headers),
      })
    )
  }
)
