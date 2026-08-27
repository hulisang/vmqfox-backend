import React from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { BookOpen, Code2, ShieldAlert } from 'lucide-react'

export const ApiDocView: React.FC = () => {
  return (
    <div className="space-y-6">
      <Card className="p-6">
        <CardHeader className="p-0 pb-6">
          <div className="flex items-center gap-2">
            <div className="p-2 rounded-2xl bg-primary/10 text-primary">
              <BookOpen className="size-4" />
            </div>
            <div>
              <CardTitle className="text-base">VMQ API 协议对接文档</CardTitle>
              <p className="text-xs text-muted-foreground mt-0.5">商户系统发起支付与接收异步回调指南</p>
            </div>
          </div>
        </CardHeader>

        <CardContent className="p-0 space-y-6 text-sm">
          {/* 1. 创建订单 */}
          <div className="space-y-3">
            <h3 className="font-bold text-foreground flex items-center gap-1.5">
              <Code2 className="size-4 text-primary" /> 1. 创建订单接口
            </h3>
            <div className="p-3 bg-muted/40 rounded-2xl border border-border/70 font-mono text-xs">
              <span className="text-emerald-600 dark:text-emerald-400 font-bold">POST / GET</span> /api/order/create
            </div>
            <div className="text-xs text-muted-foreground leading-relaxed">
              请求参数包含：<code>payId</code>（商户唯一订单号）、<code>type</code>（1: 微信, 2: 支付宝）、<code>price</code>（金额）、<code>sign</code>（HMAC-SHA256 签名）、<code>param</code>（自定义透传参数）。
            </div>
          </div>

          {/* 2. 签名算法 */}
          <div className="space-y-3">
            <h3 className="font-bold text-foreground flex items-center gap-1.5">
              <ShieldAlert className="size-4 text-amber-500" /> 2. V2 签名算法 (HMAC-SHA-256)
            </h3>
            <div className="p-4 bg-muted/50 rounded-2xl border border-border/70 space-y-2 font-mono text-xs leading-relaxed">
              <div>// 拼接规则：payId + param + type + price + key</div>
              <div className="text-primary font-semibold">signStr = payId + param + type + price + key</div>
              <div>sign = HMAC_SHA256(signStr, key).toLowerCase()</div>
            </div>
          </div>

          {/* 3. 异步回调 */}
          <div className="space-y-3">
            <h3 className="font-bold text-foreground flex items-center gap-1.5">
              <Code2 className="size-4 text-primary" /> 3. 异步通知回调 (Webhook)
            </h3>
            <div className="text-xs text-muted-foreground leading-relaxed">
              用户完成扫码支付后，系统将向商户指定的 <code>notifyUrl</code> 发起带有签名的 GET/POST 请求。商户处理成功后请直接返回纯文本 <code>success</code>。
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
