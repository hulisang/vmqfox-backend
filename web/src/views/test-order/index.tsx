import React, { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { settingsApi, publicPaymentApi } from '@/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Send, ExternalLink, RefreshCw } from 'lucide-react'
import { toast } from 'sonner'

// 客户端简易 HMAC-SHA256 / MD5 签名辅助（用于测试下单调试）
async function calculateHMACSHA256(message: string, secret: string): Promise<string> {
  const encoder = new TextEncoder()
  const keyData = encoder.encode(secret)
  const msgData = encoder.encode(message)
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    keyData,
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  )
  const signature = await crypto.subtle.sign('HMAC', cryptoKey, msgData)
  return Array.from(new Uint8Array(signature))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
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
      toast.error('未获取到系统商户通信密钥')
      return
    }

    setLoading(true)
    try {
      // 构造 V2 HMAC-SHA256: payId + param + type + price + key
      const signStr = `${payId}${param}${payType}${price}${settings.key}`
      const sign = await calculateHMACSHA256(signStr, settings.key)

      const res = await publicPaymentApi.createTestOrder({
        payId,
        type: payType,
        price,
        param,
        sign,
      })

      if (res?.publicToken) {
        toast.success('测试订单创建成功，正在打开收银台...')
        window.open(`/#/payment/${res.publicToken}`, '_blank')
        setPayId(`TEST_${Date.now()}`)
      }
    } catch (err: any) {
      toast.error(err.message || '发起测试订单失败')
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
              <p className="text-xs text-muted-foreground mt-0.5">模拟外部商户下单请求，自动计算签名并唤起收银台</p>
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
