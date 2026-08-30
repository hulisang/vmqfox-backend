import React, { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { settingsApi, SystemSettings } from '@/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { CopyButton } from '@/components/common/copy-button'
import { Settings, Key, Shield, RefreshCw, Save } from 'lucide-react'
import { toast } from 'sonner'

export const SettingsView: React.FC = () => {
  const queryClient = useQueryClient()

  const { data } = useQuery({
    queryKey: ['system-settings'],
    queryFn: settingsApi.get,
  })

  const [formData, setFormData] = useState<Partial<SystemSettings>>({
    user: '',
    pass: '',
    notifyUrl: '',
    returnUrl: '',
    key: '',
    close: '5',
    payQf: '1',
    wxpay: '',
    zfbpay: '',
  })

  useEffect(() => {
    if (data) {
      setFormData({
        user: data.user || '',
        pass: '',
        notifyUrl: data.notifyUrl || '',
        returnUrl: data.returnUrl || '',
        key: data.key || '',
        close: data.close || '5',
        payQf: data.payQf || '1',
        wxpay: data.wxpay || '',
        zfbpay: data.zfbpay || '',
      })
    }
  }, [data])

  const updateMutation = useMutation({
    mutationFn: (values: Partial<SystemSettings>) => settingsApi.update(values),
    onSuccess: () => {
      toast.success('系统配置已成功保存')
      queryClient.invalidateQueries({ queryKey: ['system-settings'] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : '保存配置失败')
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    updateMutation.mutate(formData)
  }

  // 随机生成 32 位安全 Key：使用 crypto.getRandomValues，避免 Math.random 产生可预测密钥
  const generateRandomKey = () => {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
    const randomBytes = new Uint32Array(32)
    crypto.getRandomValues(randomBytes)
    let res = ''
    for (let i = 0; i < randomBytes.length; i++) {
      res += chars.charAt(randomBytes[i] % chars.length)
    }
    setFormData((prev) => ({ ...prev, key: res }))
  }

  return (
    <div className="space-y-6">
      <form onSubmit={handleSubmit}>
        <Card className="p-6">
          <CardHeader className="p-0 pb-6 flex flex-row items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="p-2 rounded-2xl bg-primary/10 text-primary">
                <Settings className="size-4" />
              </div>
              <div>
                <CardTitle className="text-base">全局核心配置</CardTitle>
                <p className="text-xs text-muted-foreground mt-0.5">修改商户通信密钥、回调地址与收款策略</p>
              </div>
            </div>
            <Button type="submit" disabled={updateMutation.isPending} className="gap-1.5 rounded-2xl">
              <Save className="size-3.5" />
              保存配置
            </Button>
          </CardHeader>

          <CardContent className="p-0 space-y-5">
            {/* 商户通讯密钥 */}
            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <label className="text-xs font-medium text-muted-foreground flex items-center gap-1">
                  <Key className="size-3.5 text-primary" /> 商户通讯密钥 (Key)
                </label>
                <div className="flex items-center gap-1">
                  <CopyButton text={formData.key || ''} label="商户 Key" />
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={generateRandomKey}
                    className="h-7 px-2 text-xs rounded-xl hover:bg-primary/10 hover:text-primary"
                  >
                    <RefreshCw className="size-3" />
                    随机生成
                  </Button>
                </div>
              </div>
              <Input
                value={formData.key}
                onChange={(e) => setFormData({ ...formData, key: e.target.value })}
                placeholder="用于与监控端及商户系统进行签名验证的密钥"
                className="font-mono text-xs"
              />
            </div>

            {/* 异步通知与同步跳转 URL */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">全局默认异步通知 URL (Notify URL)</label>
                <Input
                  value={formData.notifyUrl}
                  onChange={(e) => setFormData({ ...formData, notifyUrl: e.target.value })}
                  placeholder="http://..."
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">全局默认同步跳转 URL (Return URL)</label>
                <Input
                  value={formData.returnUrl}
                  onChange={(e) => setFormData({ ...formData, returnUrl: e.target.value })}
                  placeholder="http://..."
                />
              </div>
            </div>

            {/* 超时与加减价 */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">订单超时自动关闭 (分钟)</label>
                <Input
                  type="number"
                  value={formData.close}
                  onChange={(e) => setFormData({ ...formData, close: e.target.value })}
                  placeholder="默认 5 分钟"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">金额区分策略 (payQf)</label>
                <select
                  value={formData.payQf}
                  onChange={(e) => setFormData({ ...formData, payQf: e.target.value })}
                  className="flex h-10 w-full rounded-2xl border border-input bg-background/80 px-3.5 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  <option value="1">递减区分金额 (0.01, 0.02...)</option>
                  <option value="2">递增区分金额 (0.01, 0.02...)</option>
                </select>
              </div>
            </div>

            {/* 修改管理员凭据 */}
            <div className="pt-4 border-t border-border/60">
              <div className="text-xs font-semibold text-foreground mb-3 flex items-center gap-1.5">
                <Shield className="size-3.5 text-primary" /> 管理员凭据修改
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted-foreground">管理员用户名</label>
                  <Input
                    value={formData.user}
                    onChange={(e) => setFormData({ ...formData, user: e.target.value })}
                    placeholder="管理员用户名"
                  />
                </div>
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted-foreground">新密码 (留空则不修改)</label>
                  <Input
                    type="password"
                    value={formData.pass}
                    onChange={(e) => setFormData({ ...formData, pass: e.target.value })}
                    placeholder="不修改请留空"
                  />
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </form>
    </div>
  )
}
