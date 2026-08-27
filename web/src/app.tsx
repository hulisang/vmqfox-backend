import React, { useState, useEffect } from 'react'
import { useAuthStore } from '@/stores'
import { AppShell, NavItem } from '@/components/app-shell'
import { LoginView } from '@/views/login'
import { DashboardView } from '@/views/dashboard'
import { OrdersView } from '@/views/orders'
import { QrcodesView } from '@/views/qrcodes'
import { MonitorView } from '@/views/monitor'
import { SettingsView } from '@/views/settings'
import { TestOrderView } from '@/views/test-order'
import { ApiDocView } from '@/views/api-doc'
import { PaymentView } from '@/views/payment'
import { PaymentResultView } from '@/views/payment/result'

export const App: React.FC = () => {
  const { isAuthenticated } = useAuthStore()
  const [currentHash, setCurrentHash] = useState(window.location.hash || '#/dashboard')
  const [currentTab, setCurrentTab] = useState<NavItem>('dashboard')

  useEffect(() => {
    const handleHashChange = () => {
      const hash = window.location.hash || '#/dashboard'
      setCurrentHash(hash)

      if (hash.startsWith('#/payment/result/')) {
        // payment result view
      } else if (hash.startsWith('#/payment/')) {
        // payment view
      } else if (hash === '#/orders') {
        setCurrentTab('orders')
      } else if (hash === '#/wxqrcode' || hash === '#/qrcodes/wechat') {
        setCurrentTab('wechat-qrcodes')
      } else if (hash === '#/zfbqrcode' || hash === '#/qrcodes/alipay') {
        setCurrentTab('alipay-qrcodes')
      } else if (hash === '#/monitor' || hash === '#/monitorSettings') {
        setCurrentTab('monitor')
      } else if (hash === '#/settings' || hash === '#/systemSettings') {
        setCurrentTab('settings')
      } else if (hash === '#/testOrder' || hash === '#/test-order') {
        setCurrentTab('test-order')
      } else if (hash === '#/api' || hash === '#/api-doc') {
        setCurrentTab('api-doc')
      } else {
        setCurrentTab('dashboard')
      }
    }

    window.addEventListener('hashchange', handleHashChange)
    handleHashChange()
    return () => window.removeEventListener('hashchange', handleHashChange)
  }, [])

  const handleTabChange = (tab: NavItem) => {
    setCurrentTab(tab)
    if (tab === 'dashboard') window.location.hash = '#/dashboard'
    else if (tab === 'orders') window.location.hash = '#/orders'
    else if (tab === 'wechat-qrcodes') window.location.hash = '#/qrcodes/wechat'
    else if (tab === 'alipay-qrcodes') window.location.hash = '#/qrcodes/alipay'
    else if (tab === 'monitor') window.location.hash = '#/monitor'
    else if (tab === 'settings') window.location.hash = '#/settings'
    else if (tab === 'test-order') window.location.hash = '#/test-order'
    else if (tab === 'api-doc') window.location.hash = '#/api-doc'
  }

  // 1. 公开收银台页面（免登录）
  if (currentHash.startsWith('#/payment/result/')) {
    const token = currentHash.replace('#/payment/result/', '')
    return <PaymentResultView publicToken={token} />
  }

  if (currentHash.startsWith('#/payment/')) {
    const token = currentHash.replace('#/payment/', '')
    return <PaymentView publicToken={token} />
  }

  // 2. 登录校验
  const authed = isAuthenticated()
  if (!authed || currentHash === '#/login') {
    return <LoginView />
  }

  // 3. 管理后台页面
  return (
    <AppShell currentTab={currentTab} onTabChange={handleTabChange}>
      {currentTab === 'dashboard' && <DashboardView />}
      {currentTab === 'orders' && <OrdersView />}
      {currentTab === 'wechat-qrcodes' && <QrcodesView type="wechat" />}
      {currentTab === 'alipay-qrcodes' && <QrcodesView type="alipay" />}
      {currentTab === 'monitor' && <MonitorView />}
      {currentTab === 'settings' && <SettingsView />}
      {currentTab === 'test-order' && <TestOrderView />}
      {currentTab === 'api-doc' && <ApiDocView />}
    </AppShell>
  )
}
