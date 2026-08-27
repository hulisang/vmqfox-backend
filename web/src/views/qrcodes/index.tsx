import React, { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { qrcodeApi } from '@/api'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableHeader, TableBody, TableHead, TableRow, TableCell } from '@/components/ui/table'
import { Switch } from '@/components/ui/switch'
import { QRCodeView } from '@/components/common/qr-code-view'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Plus, Trash2, QrCode, Eye, Upload } from 'lucide-react'
import { toast } from 'sonner'
import jsQR from 'jsqr'

interface QrcodesViewProps {
  type: 'wechat' | 'alipay'
}

export const QrcodesView: React.FC<QrcodesViewProps> = ({ type }) => {
  const queryClient = useQueryClient()
  const [page] = useState(1)
  const limit = 20

  const [dialogOpen, setDialogOpen] = useState(false)
  const [addPrice, setAddPrice] = useState('')
  const [addPayUrl, setAddPayUrl] = useState('')
  const [previewQr, setPreviewQr] = useState<string | null>(null)

  const isWechat = type === 'wechat'
  const payTypeName = isWechat ? '微信' : '支付宝'

  const { data, isLoading } = useQuery({
    queryKey: ['qrcodes-list', type, page],
    queryFn: () => qrcodeApi.list({ type, page, limit }),
  })

  // 添加二维码 Mutation
  const createMutation = useMutation({
    mutationFn: (params: { price: string; payUrl: string }) =>
      qrcodeApi.create({
        type: isWechat ? 1 : 2,
        price: params.price,
        payUrl: params.payUrl,
      }),
    onSuccess: () => {
      toast.success(`${payTypeName} 二维码添加成功`)
      setDialogOpen(false)
      setAddPrice('')
      setAddPayUrl('')
      queryClient.invalidateQueries({ queryKey: ['qrcodes-list', type] })
    },
    onError: (err: any) => {
      toast.error(err.message || '添加失败')
    },
  })

  // 状态切换 Mutation
  const toggleStateMutation = useMutation({
    mutationFn: ({ id, state }: { id: number; state: number }) =>
      qrcodeApi.setState(id, state),
    onSuccess: () => {
      toast.success('状态已更新')
      queryClient.invalidateQueries({ queryKey: ['qrcodes-list', type] })
    },
    onError: (err: any) => {
      toast.error(err.message || '状态切换失败')
    },
  })

  // 删除 Mutation
  const deleteMutation = useMutation({
    mutationFn: (id: number) => qrcodeApi.delete(type, id),
    onSuccess: () => {
      toast.success('二维码已删除')
      queryClient.invalidateQueries({ queryKey: ['qrcodes-list', type] })
    },
    onError: (err: any) => {
      toast.error(err.message || '删除失败')
    },
  })

  // 图片解析二维码
  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (event) => {
      const img = new Image()
      img.onload = () => {
        const canvas = document.createElement('canvas')
        const ctx = canvas.getContext('2d')
        if (!ctx) return
        canvas.width = img.width
        canvas.height = img.height
        ctx.drawImage(img, 0, 0)
        const imageData = ctx.getImageData(0, 0, img.width, img.height)
        const code = jsQR(imageData.data, imageData.width, imageData.height)
        if (code) {
          setAddPayUrl(code.data)
          toast.success('二维码识别成功')
        } else {
          toast.error('未能识别出有效的二维码内容，请手动输入')
        }
      }
      img.src = event.target?.result as string
    }
    reader.readAsDataURL(file)
  }

  return (
    <div className="space-y-6">
      {/* 顶部操作区 */}
      <div className="flex items-center justify-between bg-card/60 border border-border/60 p-4 rounded-3xl backdrop-blur-xs">
        <div className="flex items-center gap-2">
          <div className={`p-2 rounded-2xl ${isWechat ? 'bg-emerald-500/10 text-emerald-600' : 'bg-blue-500/10 text-blue-600'}`}>
            <QrCode className="size-4" />
          </div>
          <div>
            <div className="text-sm font-semibold">{payTypeName}通用/固定金额码库</div>
            <div className="text-xs text-muted-foreground">共配置 {data?.total ?? 0} 个收款码</div>
          </div>
        </div>

        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTrigger asChild>
            <Button size="sm" className="gap-1.5 rounded-2xl">
              <Plus className="size-3.5" />
              添加收款码
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>添加 {payTypeName} 收款码</DialogTitle>
            </DialogHeader>
            <div className="space-y-4 py-2">
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">收款金额 (元)</label>
                <Input
                  placeholder="留空或填 0 表示通用任意金额二维码"
                  value={addPrice}
                  onChange={(e) => setAddPrice(e.target.value)}
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">上传二维码图片自动识别</label>
                <div className="relative border-2 border-dashed border-border/80 rounded-2xl p-4 flex flex-col items-center justify-center hover:bg-muted/40 transition-colors cursor-pointer">
                  <input
                    type="file"
                    accept="image/*"
                    onChange={handleFileUpload}
                    className="absolute inset-0 opacity-0 cursor-pointer"
                  />
                  <Upload className="size-6 text-muted-foreground mb-1" />
                  <span className="text-xs text-muted-foreground">点击或拖拽上传收款码图片</span>
                </div>
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">二维码原始链接 / 解析内容</label>
                <Input
                  placeholder={isWechat ? 'wxp://...' : 'https://qr.alipay.com/...'}
                  value={addPayUrl}
                  onChange={(e) => setAddPayUrl(e.target.value)}
                />
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <Button variant="outline" onClick={() => setDialogOpen(false)}>
                  取消
                </Button>
                <Button
                  onClick={() =>
                    createMutation.mutate({
                      price: addPrice || '0.00',
                      payUrl: addPayUrl,
                    })
                  }
                  disabled={!addPayUrl || createMutation.isPending}
                >
                  确认添加
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* 列表卡片 */}
      <Card className="p-6">
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>面额</TableHead>
                <TableHead>二维码内容</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-10 text-muted-foreground text-xs">
                    数据加载中...
                  </TableCell>
                </TableRow>
              ) : data?.items && data.items.length > 0 ? (
                data.items.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-semibold text-xs text-primary">
                      {Number(row.price) > 0 ? `¥ ${row.price}` : '通用任意金额'}
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground max-w-xs truncate">
                      {row.pay_url}
                    </TableCell>
                    <TableCell>
                      <Switch
                        checked={row.state === 1}
                        onCheckedChange={(checked) =>
                          toggleStateMutation.mutate({ id: row.id, state: checked ? 1 : 0 })
                        }
                      />
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground font-mono">
                      {row.create_date || '-'}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setPreviewQr(row.pay_url)}
                          className="h-8 px-2 text-xs"
                        >
                          <Eye className="size-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => deleteMutation.mutate(row.id)}
                          disabled={deleteMutation.isPending}
                          className="h-8 px-2 text-xs text-destructive hover:bg-destructive/10"
                        >
                          <Trash2 className="size-3.5" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-12 text-muted-foreground text-xs">
                    暂未添加任何 {payTypeName} 收款码
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* 预览二维码 Dialog */}
      <Dialog open={!!previewQr} onOpenChange={() => setPreviewQr(null)}>
        <DialogContent className="max-w-xs flex flex-col items-center">
          <DialogHeader>
            <DialogTitle>{payTypeName} 收款码预览</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            {previewQr && (
              <QRCodeView
                url={previewQr}
                size={220}
                payType={isWechat ? 1 : 2}
              />
            )}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
