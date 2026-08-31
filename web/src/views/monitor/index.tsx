import React from 'react'
import { useQuery } from '@tanstack/react-query'
import { monitorApi, settingsApi } from '@/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { QRCodeView } from '@/components/common/qr-code-view'
import { CopyButton } from '@/components/common/copy-button'
import { Activity, Heart, Radio, ShieldCheck, CheckCircle2, AlertTriangle, XCircle } from 'lucide-react'
import dayjs from 'dayjs'

// 将时间戳格式化为 YYYY-MM-DD HH:mm:ss
const formatTimestamp = (val: string | number | undefined) => {
  if (!val || val === '0' || val === 0 || val === '') return '暂无记录'
  const num = typeof val === 'string' ? parseInt(val, 10) : val
  if (isNaN(num) || num <= 0) return '暂无记录'
  // 若为 10 位秒级时间戳，转为毫秒
  const timestamp = num < 10000000000 ? num * 1000 : num
  return dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss')
}

export const MonitorView: React.FC = () => {
  const { data: monitorData } = useQuery({
    queryKey: ['monitor-status'],
    queryFn: monitorApi.get,
    refetchInterval: 5000, // 5秒自动拉取最新心跳
  })

  const { data: settingsData } = useQuery({
    queryKey: ['system-settings'],
    queryFn: settingsApi.get,
  })

  // 监控端状态以后端返回的 jkstate 为唯一事实来源：1=在线，0=掉线，-1/其他=未绑定或暂无心跳。
  // 该值由后端心跳上报与生命周期任务（VMQ_MONITOR_HEARTBEAT_TIMEOUT，默认 180 秒）统一维护，
  // 前端不再自行按时间计算，避免与后端下单门禁的判定阈值出现漂移。
  const monitorState = monitorData?.jkstate

  let statusText = '未绑定或暂无心跳'
  let statusColor = 'text-amber-600 dark:text-amber-400'
  let statusBg = 'bg-amber-500/10 border-amber-500/20'
  let StatusIcon = AlertTriangle

  if (monitorState === '1') {
    statusText = '监控端运行正常 (已连接)'
    statusColor = 'text-emerald-600 dark:text-emerald-400'
    statusBg = 'bg-emerald-500/10 border-emerald-500/20'
    StatusIcon = CheckCircle2
  } else if (monitorState === '0') {
    statusText = '监控端已掉线，请检查手机 App 是否存活'
    statusColor = 'text-destructive'
    statusBg = 'bg-destructive/10 border-destructive/20'
    StatusIcon = XCircle
  }

  /**
   * 生成 Android 监控端可解析的配置码。
   *
   * 当前 vmqApk 的 MainActivity 按 `/` 切分配置码，取第一段作为主机、第二段作为密钥，
   * 并自行拼接协议前缀。因此这里必须输出 `host[:port]/key`，不能带 http(s):// 前缀，
   * 否则 App 会因切分出 3 段而直接报「二维码错误」。
   */
  const host = window.location.host
  const key = settingsData?.key || ''
  const configString = key ? `${host}/${key}` : ''

  return (
    <div className="space-y-6">
      {/* 顶部三栏指标卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* 监控端当前状态：仅展示在线/离线，不提供开关操作 */}
        <Card className="p-5 flex items-center gap-4">
          <div className="flex items-center gap-4">
            <div className={`p-3 rounded-2xl ${statusBg} ${statusColor} shrink-0`}>
              <StatusIcon className="size-5" />
            </div>
            <div>
              <div className="text-xs text-muted-foreground font-medium">监控端运行状态</div>
              <div className={`text-sm font-semibold mt-1 ${statusColor}`}>
                {statusText}
              </div>
            </div>
          </div>
        </Card>

        {/* 最近一次心跳 */}
        <Card className="p-5 flex items-center gap-4">
          <div className="p-3 rounded-2xl bg-emerald-500/10 text-emerald-600 shrink-0">
            <Heart className="size-5 animate-pulse" />
          </div>
          <div>
            <div className="text-xs text-muted-foreground font-medium">最近一次心跳记录</div>
            <div className="text-sm font-semibold mt-1 font-mono">
              {formatTimestamp(monitorData?.lastheart)}
            </div>
          </div>
        </Card>

        {/* 最近一次收款推送 */}
        <Card className="p-5 flex items-center gap-4">
          <div className="p-3 rounded-2xl bg-primary/10 text-primary shrink-0">
            <Radio className="size-5" />
          </div>
          <div>
            <div className="text-xs text-muted-foreground font-medium">最近一次收款推送</div>
            <div className="text-sm font-semibold mt-1 font-mono">
              {formatTimestamp(monitorData?.lastpay)}
            </div>
          </div>
        </Card>
      </div>

      {/* 扫码绑定与配置说明 */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* 二维码快捷扫码绑定 */}
        <Card className="p-6 flex flex-col items-center justify-between text-center">
          <CardHeader className="p-0 pb-3">
            <CardTitle className="text-base">App 扫码快捷配置</CardTitle>
            <p className="text-xs text-muted-foreground mt-0.5">在监控端 App 扫一扫即可完成服务器与 Key 绑定</p>
          </CardHeader>
          <CardContent className="p-0 flex flex-col items-center space-y-3">
            <div className="py-2">
              {configString ? (
                <QRCodeView url={configString} size={180} />
              ) : (
                <div className="size-44 bg-muted/60 rounded-2xl flex items-center justify-center text-xs text-muted-foreground">
                  请先在系统设置中配置密钥
                </div>
              )}
            </div>
            {configString && (
              <div className="w-full flex items-center justify-between bg-muted/40 p-2.5 rounded-2xl border border-border/70 text-xs">
                <span className="font-mono truncate max-w-[180px] text-muted-foreground">{configString}</span>
                <CopyButton text={configString} label="配置参数" />
              </div>
            )}
          </CardContent>
        </Card>

        {/* Android 监控端配置与保活指南 */}
        <Card className="lg:col-span-2 p-6 flex flex-col justify-between">
          <CardHeader className="p-0 pb-4">
            <div className="flex items-center gap-2">
              <div className="p-2 rounded-2xl bg-primary/10 text-primary">
                <Activity className="size-4" />
              </div>
              <div>
                <CardTitle className="text-base">Android 监控端 App 对接与保活说明</CardTitle>
                <p className="text-xs text-muted-foreground mt-0.5">配合 vmqApk 安装在专用安卓手机上进行收款自动化监听</p>
              </div>
            </div>
          </CardHeader>
          <CardContent className="p-0 space-y-4 text-xs leading-relaxed text-muted-foreground">
            <div className="p-4 rounded-2xl bg-muted/40 border border-border/70 space-y-2">
              <div className="flex items-center gap-1.5 font-medium text-foreground">
                <ShieldCheck className="size-4 text-primary" />
                <span>配置与保活步骤：</span>
              </div>
              <ol className="list-decimal list-inside space-y-1.5 pl-1">
                <li>安装系统配套的 Android 监控端 App (vmqApk)。</li>
                <li>在手机设置中为 App 授予「<b>通知监听权限</b>」并加入「<b>电池白名单 / 无限制运行</b>」。</li>
                <li>打开 App 使用扫一扫扫描左侧二维码，或手动填入左下方的配置数据。</li>
                <li>点击「保存并启动监控」，上方状态指示灯变绿且显示实时心跳时间即代表连接成功。</li>
              </ol>
              <div className="pt-1 text-[11px]">
                配置数据格式为 <span className="font-mono text-foreground">主机[:端口]/通讯密钥</span>，
                App 会按该格式解析服务器地址与密钥。本机调试请使用局域网 IP，不要使用 localhost。
              </div>
            </div>

            <div className="flex items-center justify-between p-3.5 rounded-2xl border border-border/60 bg-background/50">
              <div>
                <div className="font-semibold text-foreground">配套客户端下载</div>
                <div className="text-[11px] text-muted-foreground mt-0.5">支持微信/支付宝官方通知与免密支付自动监听</div>
              </div>
              <a
                href="https://github.com/szvone/vmqApk/releases"
                target="_blank"
                rel="noopener noreferrer"
                className="px-3.5 py-1.5 rounded-xl bg-primary text-primary-foreground text-xs font-medium hover:bg-primary/90 transition-all"
              >
                下载最新版 APK
              </a>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
