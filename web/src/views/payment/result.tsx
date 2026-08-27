import React, { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { publicPaymentApi } from '@/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { CheckCircle2, ArrowRight } from 'lucide-react'

interface PaymentResultViewProps {
  publicToken: string
}

export const PaymentResultView: React.FC<PaymentResultViewProps> = ({ publicToken }) => {
  const [countdown, setCountdown] = useState(5)

  const { data: order } = useQuery({
    queryKey: ['payment-result-order', publicToken],
    queryFn: () => publicPaymentApi.getOrder(publicToken),
    enabled: !!publicToken,
  })

  useEffect(() => {
    if (!order?.returnUrl) return
    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          clearInterval(timer)
          window.location.href = order.returnUrl!
          return 0
        }
        return prev - 1
      })
    }, 1000)
    return () => clearInterval(timer)
  }, [order?.returnUrl])

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

          {order?.returnUrl && (
            <div className="pt-2">
              <Button
                onClick={() => (window.location.href = order.returnUrl!)}
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
