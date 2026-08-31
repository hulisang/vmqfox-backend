import React from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { BookOpen, Code2, ShieldAlert, Webhook, ArrowLeftRight } from 'lucide-react'

/**
 * 商户对接文档。
 * 内容必须与 Go 侧实现保持一致：
 *  - 建单签名域见 internal/domain/payment.CreateSignV2
 *  - 回调与回跳签名域见 internal/domain/payment.CallbackSignV2
 *  - 公开收银台路由见 internal/http/router.go
 */
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
              <span className="text-emerald-600 dark:text-emerald-400 font-bold">POST</span> /api/order/create
            </div>
            <div className="text-xs text-muted-foreground leading-relaxed space-y-1.5">
              <div>
                请求参数：<code>payId</code>（商户唯一订单号）、<code>type</code>（1 微信 / 2 支付宝）、
                <code>price</code>（金额，两位小数字符串）、<code>param</code>（自定义透传参数）、
                <code>notifyUrl</code>、<code>returnUrl</code>、<code>sign</code>。
              </div>
              <div>
                <code>notifyUrl</code> 与 <code>returnUrl</code> 留空时使用后台系统设置中的默认值，
                但签名必须按<b>实际提交的值</b>计算：留空即以空字符串参与签名。
              </div>
              <div>
                响应 <code>data</code> 包含 <code>publicToken</code> 与 <code>redirectUrl</code>；
                请引导买家访问 <code>redirectUrl</code>，不要自行拼接内部订单号。
                传入 <code>isHtml=1</code> 时后端直接返回跳转页面。
              </div>
            </div>
          </div>

          {/* 2. 签名算法 */}
          <div className="space-y-3">
            <h3 className="font-bold text-foreground flex items-center gap-1.5">
              <ShieldAlert className="size-4 text-amber-500" /> 2. V2 签名算法 (HMAC-SHA-256)
            </h3>
            <div className="p-4 bg-muted/50 rounded-2xl border border-border/70 space-y-2 font-mono text-xs leading-relaxed">
              <div className="text-muted-foreground">// 建单签名域，字段顺序固定，密钥不拼入明文</div>
              <div className="text-primary font-semibold break-all">
                canonical = "payId=" + payId + "&amp;param=" + param + "&amp;type=" + type
                + "&amp;price=" + price + "&amp;notifyUrl=" + notifyUrl + "&amp;returnUrl=" + returnUrl
              </div>
              <div>sign = HMAC_SHA256(canonical, key) 的小写十六进制</div>
            </div>
            <div className="p-3 rounded-2xl bg-amber-500/10 border border-amber-500/20 text-xs text-amber-700 dark:text-amber-400 leading-relaxed">
              旧版 v1（MD5 拼接 <code>payId + param + type + price + key</code>）已停止受理。
              服务端识别到 v1 签名会返回明确的升级提示，不会放行。
            </div>
          </div>

          {/* 3. 异步回调 */}
          <div className="space-y-3">
            <h3 className="font-bold text-foreground flex items-center gap-1.5">
              <Webhook className="size-4 text-primary" /> 3. 异步通知回调 (Webhook)
            </h3>
            <div className="text-xs text-muted-foreground leading-relaxed space-y-1.5">
              <div>
                买家付款后，系统向 <code>notifyUrl</code> 推送
                <code>payId</code>、<code>param</code>、<code>type</code>、<code>price</code>、
                <code>reallyPrice</code> 与 <code>sign</code>。
                商户处理成功后必须返回纯文本 <code>success</code>，其它响应视为失败并进入重试。
              </div>
              <div>
                回调 <code>payId</code> 是<b>商户订单号</b>，请以它为对账主键。
              </div>
              <div>
                出站通知默认只允许 <code>https</code> 且拒绝内网地址，请确保回调地址为公网可达的 HTTPS。
              </div>
            </div>
            <div className="p-4 bg-muted/50 rounded-2xl border border-border/70 space-y-2 font-mono text-xs leading-relaxed">
              <div className="text-muted-foreground">// 回调与同步回跳共用同一签名域</div>
              <div className="text-primary font-semibold break-all">
                canonical = "payId=" + payId + "&amp;param=" + param + "&amp;type=" + type
                + "&amp;price=" + price + "&amp;reallyPrice=" + reallyPrice
              </div>
              <div>sign = HMAC_SHA256(canonical, key) 的小写十六进制</div>
            </div>
          </div>

          {/* 4. 收银台与回跳 */}
          <div className="space-y-3">
            <h3 className="font-bold text-foreground flex items-center gap-1.5">
              <ArrowLeftRight className="size-4 text-primary" /> 4. 收银台查询与同步回跳
            </h3>
            <div className="p-3 bg-muted/40 rounded-2xl border border-border/70 font-mono text-xs space-y-1">
              <div>GET /api/order/get/&#123;publicToken&#125;</div>
              <div>GET /api/order/check/&#123;publicToken&#125;</div>
              <div>GET /api/order/return-url/&#123;publicToken&#125;</div>
            </div>
            <div className="text-xs text-muted-foreground leading-relaxed space-y-1.5">
              <div>
                <code>publicToken</code> 是高熵随机凭据，只能放在路径中，不要写入日志或第三方统计。
              </div>
              <div>
                回跳地址只能通过 <code>return-url</code> 接口获取，且服务端仅在订单已支付时才签发；
                公开订单响应本身不包含回跳地址、透传参数与通知地址。
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
