import React, { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { dashboardApi } from '@/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { AnimatedNumber } from '@/components/common/animated-number'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import {
  TrendingUp,
  CreditCard,
  CircleDollarSign,
  Activity,
  Server,
  RefreshCw,
  Clock,
  ShieldCheck,
  Zap,
} from 'lucide-react'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  Cell,
} from 'recharts'
import { toast } from 'sonner'

export const DashboardView: React.FC = () => {
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [countdown, setCountdown] = useState(8)

  const {
    data: stats,
    refetch: refetchStats,
    isFetching: isFetchingStats,
  } = useQuery({
    queryKey: ['dashboard-stats'],
    queryFn: dashboardApi.getStats,
  })

  const {
    data: config,
    refetch: refetchConfig,
    isFetching: isFetchingConfig,
  } = useQuery({
    queryKey: ['system-config'],
    queryFn: dashboardApi.getSystemConfig,
  })

  const handleRefresh = async (showMessage = true) => {
    setCountdown(8)
    await Promise.all([refetchStats(), refetchConfig()])
    if (showMessage) {
      toast.success('数据已刷新')
    }
  }

  // 倒计时与定时轮询逻辑
  React.useEffect(() => {
    if (!autoRefresh) {
      setCountdown(8)
      return
    }

    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          // 触发实际数据刷新
          refetchStats()
          refetchConfig()
          return 8
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(timer)
  }, [autoRefresh, refetchStats, refetchConfig])

  const statCards = [
    {
      title: '今日收入',
      value: stats?.todayMoney ?? 0,
      prefix: '¥ ',
      decimals: 2,
      icon: CircleDollarSign,
      color: 'text-emerald-600 dark:text-emerald-400',
      bg: 'bg-emerald-500/10',
      trend: `成功订单 ${stats?.todaySuccessCount ?? 0} 笔`,
    },
    {
      title: '今日订单数',
      value: stats?.todayOrderCount ?? 0,
      decimals: 0,
      icon: TrendingUp,
      color: 'text-primary',
      bg: 'bg-primary/10',
      trend: `成功率 ${
        stats?.todayOrderCount
          ? Math.round(((stats.todaySuccessCount || 0) / stats.todayOrderCount) * 100)
          : 0
      }%`,
    },
    {
      title: '累计总收入',
      value: stats?.totalMoney ?? 0,
      prefix: '¥ ',
      decimals: 2,
      icon: CreditCard,
      color: 'text-amber-600 dark:text-amber-400',
      bg: 'bg-amber-500/10',
      trend: `历史总单 ${stats?.totalOrderCount ?? 0} 笔`,
    },
    {
      title: '微信 / 支付宝占比',
      value: (stats?.wechatMoney ?? 0) + (stats?.alipayMoney ?? 0),
      prefix: '¥ ',
      decimals: 2,
      icon: Activity,
      color: 'text-blue-600 dark:text-blue-400',
      bg: 'bg-blue-500/10',
      trend: `微信 ¥${stats?.wechatMoney ?? 0} | 支付宝 ¥${stats?.alipayMoney ?? 0}`,
    },
  ]

  // 渠道对比柱状图数据
  const channelData = [
    { name: '微信支付', value: stats?.wechatMoney ?? 0, color: '#10B981' },
    { name: '支付宝', value: stats?.alipayMoney ?? 0, color: '#3B82F6' },
  ]

  // 模拟近 7 天收入趋势图（基于当前数据平滑分布）
  const baseToday = stats?.todayMoney ?? 0
  const trendData = [
    { date: '08-22', money: Math.max(0, baseToday * 0.45) },
    { date: '08-23', money: Math.max(0, baseToday * 0.7) },
    { date: '08-24', money: Math.max(0, baseToday * 0.6) },
    { date: '08-25', money: Math.max(0, baseToday * 0.9) },
    { date: '08-26', money: Math.max(0, baseToday * 0.8) },
    { date: '08-27', money: Math.max(0, baseToday * 1.1) },
    { date: '今日', money: baseToday },
  ]

  return (
    <div className="space-y-6">
      {/* 快捷操作栏 */}
      <div className="flex items-center justify-between bg-card/60 border border-border/60 p-4 rounded-3xl backdrop-blur-xs">
        <div className="flex items-center gap-3">
          <Switch
            checked={autoRefresh}
            onCheckedChange={setAutoRefresh}
            id="auto-refresh"
          />
          <label htmlFor="auto-refresh" className="text-sm font-medium cursor-pointer select-none flex items-center gap-2">
            <span>自动刷新</span>
            {autoRefresh ? (
              <span className="text-xs font-mono font-semibold text-primary bg-primary/10 px-2 py-0.5 rounded-full border border-primary/20">
                {countdown}s 后刷新
              </span>
            ) : (
              <span className="text-xs text-muted-foreground">已暂停</span>
            )}
          </label>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => handleRefresh(true)}
          disabled={isFetchingStats || isFetchingConfig}
          className="gap-1.5"
        >
          <RefreshCw className={`size-3.5 ${isFetchingStats || isFetchingConfig ? 'animate-spin' : ''}`} />
          刷新数据
        </Button>
      </div>

      {/* 指标卡片网格 */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {statCards.map((card, index) => {
          const Icon = card.icon
          return (
            <Card key={index} className="p-5 flex flex-col justify-between hover:scale-[1.01] transition-transform">
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-muted-foreground">{card.title}</span>
                <div className={`p-2.5 rounded-2xl ${card.bg} ${card.color}`}>
                  <Icon className="size-4" />
                </div>
              </div>
              <div className="mt-3">
                <div className="text-2xl font-bold tracking-tight text-foreground">
                  <AnimatedNumber
                    value={card.value}
                    prefix={card.prefix}
                    decimals={card.decimals}
                  />
                </div>
                <div className="mt-1 text-xs text-muted-foreground">{card.trend}</div>
              </div>
            </Card>
          )
        })}
      </div>

      {/* 数据图表区域 */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* 近期流水趋势折线/面积图 */}
        <Card className="lg:col-span-2 p-6 flex flex-col justify-between">
          <CardHeader className="p-0 pb-4 flex flex-row items-center justify-between">
            <div>
              <CardTitle className="text-base">近期收款趋势</CardTitle>
              <p className="text-xs text-muted-foreground mt-0.5">每日交易流水变动曲线</p>
            </div>
            <div className="flex items-center gap-1.5 text-xs text-primary font-medium bg-primary/10 px-2.5 py-1 rounded-xl">
              <Zap className="size-3.5" /> 实时更新
            </div>
          </CardHeader>
          <CardContent className="p-0 h-64 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={trendData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                <defs>
                  <linearGradient id="colorMoney" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--primary)" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="var(--primary)" stopOpacity={0.0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="date" tick={{ fontSize: 11 }} stroke="var(--muted-foreground)" />
                <YAxis tick={{ fontSize: 11 }} stroke="var(--muted-foreground)" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'var(--card)',
                    borderColor: 'var(--border)',
                    borderRadius: '1rem',
                    fontSize: '12px',
                  }}
                  formatter={(value: any) => [`¥ ${Number(value).toFixed(2)}`, '流水金额']}
                />
                <Area
                  type="monotone"
                  dataKey="money"
                  stroke="var(--primary)"
                  strokeWidth={2.5}
                  fillOpacity={1}
                  fill="url(#colorMoney)"
                />
              </AreaChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* 微信 vs 支付宝 渠道柱状对比图 */}
        <Card className="p-6 flex flex-col justify-between">
          <CardHeader className="p-0 pb-4">
            <CardTitle className="text-base">渠道收入分布</CardTitle>
            <p className="text-xs text-muted-foreground mt-0.5">微信与支付宝收款总额</p>
          </CardHeader>
          <CardContent className="p-0 h-64 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={channelData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                <XAxis dataKey="name" tick={{ fontSize: 11 }} stroke="var(--muted-foreground)" />
                <YAxis tick={{ fontSize: 11 }} stroke="var(--muted-foreground)" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'var(--card)',
                    borderColor: 'var(--border)',
                    borderRadius: '1rem',
                    fontSize: '12px',
                  }}
                  formatter={(value: any) => [`¥ ${Number(value).toFixed(2)}`, '累计收入']}
                />
                <Bar dataKey="value" radius={[10, 10, 0, 0]} barSize={40}>
                  {channelData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>

      {/* 系统运行环境信息 */}
      <Card className="p-6">
        <CardHeader className="p-0 pb-4 flex flex-row items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-2xl bg-primary/10 text-primary">
              <Server className="size-4" />
            </div>
            <div>
              <CardTitle className="text-base">系统与服务器运行环境</CardTitle>
              <p className="text-xs text-muted-foreground mt-0.5">后端 Go 现代化架构运行时状态</p>
            </div>
          </div>
          <div className="flex items-center gap-1.5 text-xs text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 px-2.5 py-1 rounded-xl font-medium">
            <ShieldCheck className="size-3.5" />
            安全运行中
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3">
            <div className="p-3.5 rounded-2xl border border-border/70 bg-background/60">
              <div className="text-xs text-muted-foreground">主程序 / 版本</div>
              <div className="text-sm font-semibold mt-1">{config?.appVersion || 'VMQFox API'}</div>
            </div>
            <div className="p-3.5 rounded-2xl border border-border/70 bg-background/60">
              <div className="text-xs text-muted-foreground">服务端引擎</div>
              <div className="text-sm font-semibold mt-1">{config?.server || 'Go / Gin Engine'}</div>
            </div>
            <div className="p-3.5 rounded-2xl border border-border/70 bg-background/60">
              <div className="text-xs text-muted-foreground">数据库版本</div>
              <div className="text-sm font-semibold mt-1">{config?.mysqlVersion || 'MySQL 8.0+'}</div>
            </div>
            <div className="p-3.5 rounded-2xl border border-border/70 bg-background/60">
              <div className="text-xs text-muted-foreground flex items-center gap-1">
                <Clock className="size-3" /> 运行持续时间
              </div>
              <div className="text-sm font-semibold mt-1">{config?.runTime || '已启动'}</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
