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
  XCircle,
  Server,
  RefreshCw,
  Clock,
} from 'lucide-react'
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

  const todayOrder = stats?.todayOrder ?? 0
  const todaySuccessOrder = stats?.todaySuccessOrder ?? 0
  const successRate = todayOrder > 0 ? Math.round((todaySuccessOrder / todayOrder) * 100) : 0

  /**
   * 指标卡片只展示后端 /api/config/status 真实返回的字段。
   * 后端未按渠道拆分收款金额，因此不展示微信/支付宝占比，
   * 也不再用今日金额推算「近 7 天趋势」这类不存在的数据。
   */
  const statCards = [
    {
      title: '今日收入',
      value: Number(stats?.todayMoney ?? 0),
      prefix: '¥ ',
      decimals: 2,
      icon: CircleDollarSign,
      color: 'text-emerald-600 dark:text-emerald-400',
      bg: 'bg-emerald-500/10',
      trend: `成功订单 ${todaySuccessOrder} 笔`,
    },
    {
      title: '今日订单数',
      value: todayOrder,
      decimals: 0,
      icon: TrendingUp,
      color: 'text-primary',
      bg: 'bg-primary/10',
      trend: `成功率 ${successRate}%`,
    },
    {
      title: '今日关闭订单',
      value: stats?.todayCloseOrder ?? 0,
      decimals: 0,
      icon: XCircle,
      color: 'text-amber-600 dark:text-amber-400',
      bg: 'bg-amber-500/10',
      trend: '超时未支付或人工关闭',
    },
    {
      title: '累计总收入',
      value: Number(stats?.countMoney ?? 0),
      prefix: '¥ ',
      decimals: 2,
      icon: CreditCard,
      color: 'text-blue-600 dark:text-blue-400',
      bg: 'bg-blue-500/10',
      trend: `历史总单 ${stats?.countOrder ?? 0} 笔`,
    },
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

      {/* 系统运行环境信息 */}
      <Card className="p-6">
        <CardHeader className="p-0 pb-4 flex flex-row items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-2xl bg-primary/10 text-primary">
              <Server className="size-4" />
            </div>
            <div>
              <CardTitle className="text-base">系统与服务器运行环境</CardTitle>
              <p className="text-xs text-muted-foreground mt-0.5">后端 Go 服务运行时状态</p>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3">
            <div className="p-3.5 rounded-2xl border border-border/70 bg-background/60">
              <div className="text-xs text-muted-foreground">主程序 / 版本</div>
              <div className="text-sm font-semibold mt-1">{config?.appVersion || '-'}</div>
            </div>
            <div className="p-3.5 rounded-2xl border border-border/70 bg-background/60">
              <div className="text-xs text-muted-foreground">服务端引擎</div>
              <div className="text-sm font-semibold mt-1">{config?.server || '-'}</div>
            </div>
            <div className="p-3.5 rounded-2xl border border-border/70 bg-background/60">
              <div className="text-xs text-muted-foreground">数据库版本</div>
              <div className="text-sm font-semibold mt-1">{config?.mysqlVersion || '-'}</div>
            </div>
            <div className="p-3.5 rounded-2xl border border-border/70 bg-background/60">
              <div className="text-xs text-muted-foreground flex items-center gap-1">
                <Clock className="size-3" /> 运行持续时间
              </div>
              <div className="text-sm font-semibold mt-1">{config?.runTime || '-'}</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
