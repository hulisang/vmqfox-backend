import React from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { monitorApi } from '@/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import { Activity, Heart, Radio, ShieldCheck } from 'lucide-react'
import { toast } from 'sonner'

export const MonitorView: React.FC = () => {
  const queryClient = useQueryClient()

  const { data } = useQuery({
    queryKey: ['monitor-status'],
    queryFn: monitorApi.get,
    refetchInterval: 5000, // 5秒自动拉取最新心跳
  })

  const updateMutation = useMutation({
    mutationFn: (jkstate: string) => monitorApi.update({ jkstate }),
    onSuccess: () => {
      toast.success('监控端接收状态已更新')
      queryClient.invalidateQueries({ queryKey: ['monitor-status'] })
    },
    onError: (err: any) => {
      toast.error(err.message || '更新失败')
    },
  })

  const isPushEnabled = data?.jkstate === '1'

  return (
    <div className="space-y-6">
      {/* 状态卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* 运行状态 */}
        <Card className="p-5 flex items-center justify-between">
          <div>
            <div className="text-xs text-muted-foreground font-medium">心跳接收开关</div>
            <div className="text-lg font-bold mt-1">
              {isPushEnabled ? '正常接收中' : '已暂停接收'}
            </div>
          </div>
          <Switch
            checked={isPushEnabled}
            onCheckedChange={(checked) => updateMutation.mutate(checked ? '1' : '0')}
            disabled={updateMutation.isPending}
          />
        </Card>

        {/* 最近一次心跳 */}
        <Card className="p-5 flex items-center gap-4">
          <div className="p-3 rounded-2xl bg-emerald-500/10 text-emerald-600">
            <Heart className="size-5 animate-pulse" />
          </div>
          <div>
            <div className="text-xs text-muted-foreground font-medium">最近一次心跳记录</div>
            <div className="text-sm font-semibold mt-1 font-mono">
              {data?.lastheart || '暂无心跳'}
            </div>
          </div>
        </Card>

        {/* 最近一次收款推送 */}
        <Card className="p-5 flex items-center gap-4">
          <div className="p-3 rounded-2xl bg-primary/10 text-primary">
            <Radio className="size-5" />
          </div>
          <div>
            <div className="text-xs text-muted-foreground font-medium">最近一次推送通知</div>
            <div className="text-sm font-semibold mt-1 font-mono">
              {data?.lastpay || '暂无收款推送'}
            </div>
          </div>
        </Card>
      </div>

      {/* 监控端配置与 App 连接指引 */}
      <Card className="p-6">
        <CardHeader className="p-0 pb-4">
          <div className="flex items-center gap-2">
            <div className="p-2 rounded-2xl bg-primary/10 text-primary">
              <Activity className="size-4" />
            </div>
            <div>
              <CardTitle className="text-base">Android 监控端 App 对接说明</CardTitle>
              <p className="text-xs text-muted-foreground mt-0.5">配合 vmqApk 安装在专用安卓手机上进行收款自动化监听</p>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0 space-y-4">
          <div className="p-4 rounded-2xl bg-muted/40 border border-border/70 space-y-2 text-xs leading-relaxed text-muted-foreground">
            <div className="flex items-center gap-1.5 font-medium text-foreground">
              <ShieldCheck className="size-4 text-primary" />
              <span>配置步骤：</span>
            </div>
            <ol className="list-decimal list-inside space-y-1.5 pl-1">
              <li>下载并安装系统配套的 Android 监控端 App (vmqApk)。</li>
              <li>在 App 中开启「通知监听权限」与「电池优化白名单/忽略电池优化」。</li>
              <li>在 App 配置页面填写本系统域名 URL (包含协议头，如 <code>http://your-domain:8080</code>) 以及系统设置中的「商户通讯密钥 Key」。</li>
              <li>点击「保存并启动监控」，右上角状态指示灯变绿且上方出现最新心跳时间即代表配置成功。</li>
            </ol>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
