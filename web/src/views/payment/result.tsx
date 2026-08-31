import React, { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { publicPaymentApi } from '@/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { CheckCircle2, ArrowRight } from 'lucide-react'

interface PaymentResultViewProps {
  publicToken: string
}

/**
 * 二次校验最终导航地址。
 * 服务端已限制回跳地址只能是 http(s)，前端在真正跳转前再校验一次，
 * 防止任何环节注入 javascript:、data: 等伪协议。
 */
function safeExternalUrl(raw: string | undefined): string | null {
  if (!raw) return null
  try {
    const parsed = new URL(raw, window.location.origin)
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return null
    return parsed.toString()
  } catch {
    return null
  }
}

export const PaymentResultView: React.FC<PaymentResultViewProps> = ({ publicToken }) => {
  const [countdown, setCountdown] = useState(5)

  const { data: order } = useQuery({
    queryKey: ['payment-result-order', publicToken],
    queryFn: () => publicPaymentApi.getOrder(publicToken),
    enabled: !!publicToken,
  })

  /**
   * 回跳地址来自独立的服务端接口，并由服务端附加签名。
   * 公开订单 DTO 不含 returnUrl，前端也不再从订单数据里猜测跳转目标。
   * 后端仅在订单已支付时才签发，未配置回跳地址时返回业务错误，此处静默忽略。
   */
  const { data: returnUrlData } = useQuery({
    queryKey: ['payment-return-url', publicToken],
    queryFn: () => publicPaymentApi.getReturnUrl(publicToken),
    enabled: !!publicToken && order?.state === 1,
    retry: false,
  })

  const redirectTarget = safeExternalUrl(returnUrlData?.returnUrl)

  useEffect(() => {
    if (!redirectTarget) return
    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          clearInterval(timer)
          window.location.href = redirectTarget
          return 0
        }
        return prev - 1
      })
    }, 1000)
    return () => clearInterval(timer)
  }, [redirectTarget])

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <Card className="max-w-md w-full p-8 text-center shadow-xl border-border/80">
        <CardHeader className="p-0 pb-6 flex flex-col items-center">
          <div className="p-4 bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 rounded-3xl mb-3 shadow-inner">
            <CheckCircle2 className="size-10" />
          </div>
          <CardTitle className="text-xl font-bold text-foreground">支付已成功</CardTitle>
          <p className="text-xs text-muted-foreground mt-1">商户已收到您的付款，订单处理完成</p>
        </CardHeader>

        <CardContent className="p-0 space-y-5">
          <div className="p-4 bg-muted/40 rounded-2xl border border-border/60 text-xs space-y-2 text-left">
            <div className="flex justify-between">
              <span className="text-muted-foreground">实付金额：</span>
              <span className="font-bold text-primary text-sm">¥ {order?.reallyPrice || '-'}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">商户单号：</span>
              <span className="font-mono text-foreground">{order?.payId || '-'}</span>
            </div>
          </div>

          {redirectTarget && (
            <div className="pt-2">
              <Button
                onClick={() => (window.location.href = redirectTarget)}
                className="w-full gap-2 rounded-2xl"
              >
                <span>返回商户 ({countdown}s)</span>
                <ArrowRight className="size-4" />
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
