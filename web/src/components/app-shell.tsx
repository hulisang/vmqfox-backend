import React from 'react'
import {
  LayoutDashboard,
  ReceiptText,
  QrCode,
  Smartphone,
  Activity,
  Settings,
  Send,
  BookOpen,
  LogOut,
  Sun,
  Moon,
} from 'lucide-react'
import { useAuthStore, useThemeStore } from '@/stores'
import { authApi } from '@/api'
import { toast } from 'sonner'

export type NavItem =
  | 'dashboard'
  | 'orders'
  | 'wechat-qrcodes'
  | 'alipay-qrcodes'
  | 'monitor'
  | 'settings'
  | 'test-order'
  | 'api-doc'

interface AppShellProps {
  currentTab: NavItem
  onTabChange: (tab: NavItem) => void
  children: React.ReactNode
}

const navItems: { id: NavItem; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { id: 'dashboard', label: '总览看板', icon: LayoutDashboard },
  { id: 'orders', label: '订单管理', icon: ReceiptText },
  { id: 'wechat-qrcodes', label: '微信码库', icon: QrCode },
  { id: 'alipay-qrcodes', label: '支付宝码库', icon: Smartphone },
  { id: 'monitor', label: '监控端状态', icon: Activity },
  { id: 'settings', label: '系统设置', icon: Settings },
  { id: 'test-order', label: '发起测试', icon: Send },
  { id: 'api-doc', label: '开发文档', icon: BookOpen },
]

export const AppShell: React.FC<AppShellProps> = ({ currentTab, onTabChange, children }) => {
  const { username, clearAuth } = useAuthStore()
  const { theme, toggleTheme } = useThemeStore()

  const handleLogout = async () => {
    try {
      await authApi.logout()
    } catch {
      // 服务端登出失败不阻塞本地会话清理，本地 Token 必须无条件失效
    }
    clearAuth()
    toast.success('已退出登录')
  }

  const currentNav = navItems.find((item) => item.id === currentTab)

  return (
    <div className="min-h-screen max-w-6xl mx-auto flex flex-col md:flex-row p-4 md:p-8 gap-6 relative">
      {/* 桌面端垂直 Dock 侧栏 */}
      <aside className="hidden md:flex flex-col items-center justify-between w-18 bg-card/85 backdrop-blur-md border border-border/80 rounded-3xl p-3 shadow-lg sticky top-8 h-[calc(100vh-4rem)] z-40">
        <div className="flex flex-col items-center gap-4 w-full">
          {/* Logo 标识 */}
          <div className="size-11 rounded-2xl bg-primary/15 text-primary flex items-center justify-center font-bold text-lg shadow-2xs border border-primary/20">
            V
          </div>

          <div className="w-8 h-px bg-border/80 my-1" />

          {/* 导航按钮组 */}
          <nav className="flex flex-col gap-2 w-full items-center">
            {navItems.map((item) => {
              const Icon = item.icon
              const isActive = currentTab === item.id
              return (
                <button
                  key={item.id}
                  onClick={() => onTabChange(item.id)}
                  title={item.label}
                  className={`relative size-11 rounded-2xl flex items-center justify-center transition-all duration-200 group ${
                    isActive
                      ? 'bg-primary text-primary-foreground shadow-md scale-105'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground hover:scale-105'
                  }`}
                >
                  <Icon className="size-5 shrink-0 transition-transform group-hover:scale-110" />
                </button>
              )
            })}
          </nav>
        </div>

        {/* 底部主题切换与退出登录 */}
        <div className="flex flex-col gap-2 w-full items-center">
          <button
            onClick={toggleTheme}
            title={theme === 'dark' ? '切换浅色模式' : '切换深色模式'}
            className="size-10 rounded-2xl flex items-center justify-center text-muted-foreground hover:bg-muted hover:text-foreground hover:scale-105 transition-all"
          >
            {theme === 'dark' ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </button>
          <button
            onClick={handleLogout}
            title="退出登录"
            className="size-10 rounded-2xl flex items-center justify-center text-destructive/80 hover:bg-destructive/10 hover:text-destructive hover:scale-105 transition-all"
          >
            <LogOut className="size-4" />
          </button>
        </div>
      </aside>

      {/* 主工作区 */}
      <main className="flex-1 flex flex-col min-w-0 pb-20 md:pb-0">
        {/* 顶部标题栏 */}
        <header className="flex items-center justify-between mb-6 px-1">
          <div>
            <h1 className="text-2xl md:text-3xl font-bold tracking-tight text-foreground">
              {currentNav?.label || '控制台'}
            </h1>
            <p className="text-xs md:text-sm text-muted-foreground mt-0.5">
              欢迎回来，<span className="text-primary font-medium">{username || '管理员'}</span>
            </p>
          </div>
          <div className="flex items-center gap-3">
            <span className="hidden sm:inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-primary/10 text-primary border border-primary/20">
              单商户安全模式
            </span>
          </div>
        </header>

        {/* 内容展示区 */}
        <div className="flex-1 transition-all duration-300 ease-out">{children}</div>
      </main>

      {/* 移动端底部悬浮 Dock 栏 */}
      <nav className="md:hidden fixed bottom-4 left-4 right-4 h-16 bg-card/90 backdrop-blur-lg border border-border/80 rounded-3xl flex items-center justify-around px-2 shadow-2xl z-50">
        {navItems.map((item) => {
          const Icon = item.icon
          const isActive = currentTab === item.id
          return (
            <button
              key={item.id}
              onClick={() => onTabChange(item.id)}
              className={`size-10 rounded-xl flex items-center justify-center transition-all ${
                isActive ? 'bg-primary text-primary-foreground shadow-sm scale-110' : 'text-muted-foreground'
              }`}
            >
              <Icon className="size-4" />
            </button>
          )
        })}
      </nav>
    </div>
  )
}
