import React, { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { settingsApi, publicPaymentApi } from '@/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Send, ExternalLink, RefreshCw } from 'lucide-react'
import { toast } from 'sonner'

/**
 * 浏览器端 HMAC-SHA-256，输出小写十六进制。
 * 与 Go `internal/domain/payment.hmacHex` 的输出格式保持一致。
 */
async function hmacSha256Hex(canonical: string, secret: string): Promise<string> {
  const encoder = new TextEncoder()
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    encoder.encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  )
  const signature = await crypto.subtle.sign('HMAC', cryptoKey, encoder.encode(canonical))
  return Array.from(new Uint8Array(signature))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}

/**
 * 建单签名的 canonical 串，必须与 Go `payment.CreateSignV2` 逐字节一致：
 * payId=..&param=..&type=..&price=..&notifyUrl=..&returnUrl=..
 * 商户密钥只作为 HMAC 密钥，不再拼接进明文。
 */
function buildCreateCanonical(fields: {
  payId: string
  param: string
  type: string
  price: string
  notifyUrl: string
  returnUrl: string
}): string {
  return (
    `payId=${fields.payId}` +
    `&param=${fields.param}` +
    `&type=${fields.type}` +
    `&price=${fields.price}` +
    `&notifyUrl=${fields.notifyUrl}` +
    `&returnUrl=${fields.returnUrl}`
  )
}

export const TestOrderView: React.FC = () => {
  const { data: settings } = useQuery({
    queryKey: ['system-settings'],
    queryFn: settingsApi.get,
  })

  const [payId, setPayId] = useState(`TEST_${Date.now()}`)
  const [payType, setPayType] = useState<number>(1)
  const [price, setPrice] = useState<string>('0.01')
  const [param, setParam] = useState<string>('test_param')
  const [loading, setLoading] = useState(false)

  const handleCreateOrder = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!settings?.key) {
      toast.error('未获取到系统商户通信密钥，请先在系统设置中配置')
      return
    }

    setLoading(true)
    try {
      // 签名字段与提交字段复用同一份文本，避免金额或地址在两处出现差异
      const typeText = String(payType)
      const notifyUrl = settings.notifyUrl || ''
      const returnUrl = settings.returnUrl || ''
      const canonical = buildCreateCanonical({
        payId,
        param,
        type: typeText,
        price,
        notifyUrl,
        returnUrl,
      })
      const sign = await hmacSha256Hex(canonical, settings.key)

      const res = await publicPaymentApi.createTestOrder({
        payId,
        type: payType,
        price,
        param,
        notifyUrl,
        returnUrl,
        sign,
      })

      if (res?.publicToken) {
        toast.success('测试订单创建成功，正在打开收银台...')
        window.open(`/#/payment/${encodeURIComponent(res.publicToken)}`, '_blank', 'noopener')
        setPayId(`TEST_${Date.now()}`)
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '发起测试订单失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      <Card className="p-6 max-w-2xl">
        <CardHeader className="p-0 pb-6 flex flex-row items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="p-2 rounded-2xl bg-primary/10 text-primary">
              <Send className="size-4" />
            </div>
            <div>
              <CardTitle className="text-base">发起测试订单</CardTitle>
              <p className="text-xs text-muted-foreground mt-0.5">模拟外部商户下单请求，自动计算 v2 签名并唤起收银台</p>
            </div>
          </div>
        </CardHeader>

        <CardContent className="p-0">
          <form onSubmit={handleCreateOrder} className="space-y-4">
            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <label className="text-xs font-medium text-muted-foreground">自定义商户单号 (payId)</label>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => setPayId(`TEST_${Date.now()}`)}
                  className="h-6 px-2 text-xs"
                >
                  <RefreshCw className="size-3" />
                  重新生成
                </Button>
              </div>
              <Input
                value={payId}
                onChange={(e) => setPayId(e.target.value)}
                placeholder="商户订单号"
                required
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">支付渠道</label>
                <select
                  value={payType}
                  onChange={(e) => setPayType(Number(e.target.value))}
                  className="flex h-10 w-full rounded-2xl border border-input bg-background/80 px-3.5 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  <option value={1}>微信支付 (1)</option>
                  <option value={2}>支付宝 (2)</option>
                </select>
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">支付金额 (元)</label>
                <Input
                  value={price}
                  onChange={(e) => setPrice(e.target.value)}
                  placeholder="0.01"
                  required
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">商户自定义透传参数 (param)</label>
              <Input
                value={param}
                onChange={(e) => setParam(e.target.value)}
                placeholder="透传参数，将在异步通知时原样返回"
              />
            </div>

            {/* 明确展示参与签名的通知与回跳地址，避免与后端签名域产生认知偏差 */}
            <div className="p-3.5 rounded-2xl bg-muted/40 border border-border/70 text-xs space-y-1.5 text-muted-foreground">
              <div className="font-medium text-foreground">签名域包含的系统配置</div>
              <div className="flex justify-between gap-3">
                <span>notifyUrl</span>
                <span className="font-mono truncate max-w-[60%] text-right">
                  {settings?.notifyUrl || '(未配置)'}
                </span>
              </div>
              <div className="flex justify-between gap-3">
                <span>returnUrl</span>
                <span className="font-mono truncate max-w-[60%] text-right">
                  {settings?.returnUrl || '(未配置)'}
                </span>
              </div>
            </div>

            <Button
              type="submit"
              disabled={loading}
              className="w-full gap-2 rounded-2xl mt-4"
            >
              <ExternalLink className="size-4" />
              {loading ? '正在生成订单...' : '生成订单并唤起收银台'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
