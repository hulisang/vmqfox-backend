import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'
import { App } from './app'
import { ApiError } from './api/client'
import './globals.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // 只读查询允许有限重试，但被限流或明确拒绝时立刻停止，避免前端放大 429
      retry: (failureCount, error) => {
        if (error instanceof ApiError) {
          if (error.isRateLimited) return false
          if (error.status !== null && error.status >= 400 && error.status < 500) return false
        }
        return failureCount < 1
      },
      // 退避加抖动，避免多个查询同时重试形成尖峰
      retryDelay: (attemptIndex) =>
        Math.min(1000 * 2 ** attemptIndex, 8000) + Math.random() * 250,
      refetchOnWindowFocus: false,
    },
    mutations: {
      // 写操作一律不自动重放：网络异常时重复提交可能产生重复订单或重复补单
      retry: false,
    },
  },
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
      <Toaster
        position="top-right"
        richColors
        closeButton
        theme="system"
        toastOptions={{
          className: 'rounded-2xl border border-border/80 shadow-lg text-xs font-medium',
        }}
      />
    </QueryClientProvider>
  </React.StrictMode>
)
