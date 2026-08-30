import React, { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { publicPaymentApi } from '@/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { QRCodeView } from '@/components/common/qr-code-view'
import { Clock, AlertCircle, Smartphone } from 'lucide-react'

interface PaymentViewProps {
  publicToken: string
}

export const PaymentView: React.FC<PaymentViewProps> = ({ publicToken }) => {
  const [remaining, setRemaining] = useState<number | null>(null)

  // 1. 获取公开订单详情
  const { data: order, isLoading, error } = useQuery({
    queryKey: ['payment-order', publicToken],
    queryFn: () => publicPaymentApi.getOrder(publicToken),
    enabled: !!publicToken,
  })

  // 2. 仅在订单确实处于待支付时短轮询；已支付或已关闭立即停止，避免无意义请求触发限流
  const { data: checkData } = useQuery({
    queryKey: ['payment-check', publicToken],
    queryFn: () => publicPaymentApi.checkOrder(publicToken),
    enabled: !!publicToken && order?.state === 0,
    refetchInterval: (query) => (query.state.data?.state === 0 ? 2000 : false),
  })

  // 初始化并维护倒计时
  useEffect(() => {
    if (order?.remainingSeconds !== undefined) {
      setRemaining(order.remainingSeconds)
    }
  }, [order?.remainingSeconds])

  useEffect(() => {
    if (remaining === null || remaining <= 0) return
    const timer = setInterval(() => {
      setRemaining((prev) => (prev && prev > 0 ? prev - 1 : 0))
    }, 1000)
    return () => clearInterval(timer)
  }, [remaining])

  // 当检测到已支付 (state === 1) 时跳转结果页
  const currentState = checkData?.state ?? order?.state

  useEffect(() => {
    if (currentState === 1) {
      window.location.hash = `#/payment/result/${publicToken}`
    }
  }, [currentState, publicToken])

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center p-4">
        <div className="p-8 rounded-3xl bg-card border border-border/80 text-center shadow-lg">
          <div className="size-8 border-3 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-3" />
          <div className="text-sm font-medium">正在拉取收银台数据...</div>
        </div>
      </div>
    )
  }

  if (error || !order) {
    return (
      <div className="min-h-screen flex items-center justify-center p-4">
        <Card className="max-w-md w-full p-6 text-center">
          <div className="p-3 bg-destructive/10 text-destructive rounded-2xl w-fit mx-auto mb-3">
            <AlertCircle className="size-6" />
          </div>
          <CardTitle className="text-base">无法加载订单</CardTitle>
          <p className="text-xs text-muted-foreground mt-2">该支付令牌不存在或已过期失效，请重新发起交易。</p>
        </Card>
      </div>
    )
  }

  const isWechat = order.payType === 1
  const isAlipay = order.payType === 2
  const payTypeName = isWechat ? '微信' : isAlipay ? '支付宝' : '在线'

  const minutes = Math.floor((remaining || 0) / 60)
  const seconds = (remaining || 0) % 60

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <Card className="max-w-md w-full p-6 shadow-xl border-border/80">
        <CardHeader className="p-0 pb-4 text-center">
          <div className="flex items-center justify-center gap-2 mb-2">
            <div className={`p-2 rounded-2xl ${isWechat ? 'bg-emerald-500/10 text-emerald-600' : 'bg-blue-500/10 text-blue-600'}`}>
              <Smartphone className="size-5" />
            </div>
            <CardTitle className="text-lg font-bold">
              {payTypeName} 扫码支付
            </CardTitle>
          </div>
          <p className="text-xs text-muted-foreground">
            请使用 {payTypeName} APP 扫描下方二维码完成支付
          </p>
        </CardHeader>

        <CardContent className="p-0 flex flex-col items-center space-y-4">
          {/* 金额提示 */}
          <div className="text-center py-1">
            <div className="text-3xl font-extrabold tracking-tight text-primary">
              <span className="text-lg mr-0.5">¥</span>
              {order.reallyPrice}
            </div>
            {order.price !== order.reallyPrice && (
              <div className="text-xs font-medium text-amber-600 dark:text-amber-400 mt-1 bg-amber-500/10 px-2.5 py-1 rounded-xl">
                ⚠️ 为保证即时到账，请务必支付实付金额 <b>{order.reallyPrice}</b> 元
              </div>
            )}
          </div>

          {/* 二维码展示区 */}
          <div className="py-2">
            {order.payUrl ? (
              <QRCodeView
                url={order.payUrl}
                size={210}
                payType={order.payType}
              />
            ) : (
              <div className="size-52 bg-muted/60 rounded-2xl flex items-center justify-center text-xs text-muted-foreground">
                未获取到有效收款码
              </div>
            )}
          </div>

          {/* 倒计时与安全提示 */}
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground bg-muted/40 px-3 py-1.5 rounded-full border border-border/60">
            <Clock className="size-3.5 text-primary" />
            <span>支付有效倒计时: </span>
            <span className="font-mono font-bold text-foreground">
              {String(minutes).padStart(2, '0')}:{String(seconds).padStart(2, '0')}
            </span>
          </div>

          {/* 订单详细信息折叠 */}
          <div className="w-full pt-3 border-t border-border/60 text-xs space-y-1.5 text-muted-foreground">
            <div className="flex justify-between">
              <span>商户单号：</span>
              <span className="font-mono text-foreground font-medium">{order.payId}</span>
            </div>
            <div className="flex justify-between">
              <span>订单标价：</span>
              <span>¥ {order.price}</span>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
