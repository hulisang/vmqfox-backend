import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'
import { App } from './app'
import './globals.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
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
