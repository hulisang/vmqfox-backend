import React, { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { orderApi } from '@/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Table, TableHeader, TableBody, TableHead, TableRow, TableCell } from '@/components/ui/table'
import { StatusBadge } from '@/components/common/status-badge'
import { CopyButton } from '@/components/common/copy-button'
import {
  ReceiptText,
  RotateCcw,
  Trash2,
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react'
import { toast } from 'sonner'

export const OrdersView: React.FC = () => {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [limit] = useState(10)
  const [stateFilter, setStateFilter] = useState<number | undefined>(undefined)

  const { data, isLoading } = useQuery({
    queryKey: ['orders-list', page, limit, stateFilter],
    queryFn: () => orderApi.list({ page, limit, state: stateFilter }),
  })

  // 补单 Mutation
  const reissueMutation = useMutation({
    mutationFn: (id: number) => orderApi.reissue(id),
    onSuccess: () => {
      toast.success('补单通知已成功触发')
      queryClient.invalidateQueries({ queryKey: ['orders-list'] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : '补单失败')
    },
  })

  // 单条删除 Mutation
  const deleteMutation = useMutation({
    mutationFn: (id: number) => orderApi.delete(id),
    onSuccess: () => {
      toast.success('订单已删除')
      queryClient.invalidateQueries({ queryKey: ['orders-list'] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : '删除失败')
    },
  })

  // 关闭超时订单 Mutation
  const closeExpiredMutation = useMutation({
    mutationFn: () => orderApi.closeExpired(),
    onSuccess: (res) => {
      toast.success(`已关闭 ${res?.count ?? 0} 笔超时未支付订单`)
      queryClient.invalidateQueries({ queryKey: ['orders-list'] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : '关闭超时订单失败')
    },
  })

  // 清理过期订单 Mutation
  const deleteExpiredMutation = useMutation({
    mutationFn: () => orderApi.deleteExpired(),
    onSuccess: (res) => {
      toast.success(`已删除 ${res?.count ?? 0} 笔过期/关闭订单`)
      queryClient.invalidateQueries({ queryKey: ['orders-list'] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : '清理过期订单失败')
    },
  })

  const totalPages = Math.ceil((data?.total || 0) / limit) || 1

  return (
    <div className="space-y-6">
      {/* 顶部操作与筛选 */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 bg-card/60 border border-border/60 p-4 rounded-3xl backdrop-blur-xs">
        {/* 状态过滤选项 */}
        <div className="flex items-center gap-1.5 flex-wrap">
          {[
            { label: '全部状态', value: undefined },
            { label: '等待支付', value: 0 },
            { label: '已支付', value: 1 },
            { label: '通知失败', value: 2 },
            { label: '已关闭', value: -1 },
          ].map((item) => (
            <button
              key={String(item.value)}
              onClick={() => {
                setStateFilter(item.value)
                setPage(1)
              }}
              className={`px-3 py-1.5 rounded-xl text-xs font-medium transition-all ${
                stateFilter === item.value
                  ? 'bg-primary text-primary-foreground shadow-xs'
                  : 'text-muted-foreground hover:bg-muted'
              }`}
            >
              {item.label}
            </button>
          ))}
        </div>

        {/* 批量处理按钮 */}
        <div className="flex items-center gap-2 flex-wrap">
          <Button
            variant="outline"
            size="sm"
            onClick={() => closeExpiredMutation.mutate()}
            disabled={closeExpiredMutation.isPending}
            className="text-xs"
          >
            <AlertTriangle className="size-3.5 text-amber-500" />
            关闭超时
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => deleteExpiredMutation.mutate()}
            disabled={deleteExpiredMutation.isPending}
            className="text-xs text-destructive hover:bg-destructive/10"
          >
            <Trash2 className="size-3.5" />
            清理过期
          </Button>
        </div>
      </div>

      {/* 订单表格卡片 */}
      <Card className="p-6">
        <CardHeader className="p-0 pb-4 flex flex-row items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="p-2 rounded-2xl bg-primary/10 text-primary">
              <ReceiptText className="size-4" />
            </div>
            <div>
              <CardTitle className="text-base">订单列表</CardTitle>
              <p className="text-xs text-muted-foreground mt-0.5">共 {data?.total ?? 0} 笔交易记录</p>
            </div>
          </div>
        </CardHeader>

        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>商户单号 / 系统单号</TableHead>
                <TableHead>支付渠道</TableHead>
                <TableHead>标价 / 实付</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-10 text-muted-foreground">
                    数据加载中...
                  </TableCell>
                </TableRow>
              ) : data?.items && data.items.length > 0 ? (
                data.items.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>
                      <div className="flex flex-col gap-0.5">
                        <div className="font-semibold text-xs text-foreground flex items-center gap-1">
                          <span>{row.pay_id}</span>
                          <CopyButton text={row.pay_id} label="商户单号" />
                        </div>
                        <span className="text-[11px] text-muted-foreground font-mono">{row.order_id}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-medium ${
                        row.type === 1 ? 'bg-emerald-500/10 text-emerald-600' : 'bg-blue-500/10 text-blue-600'
                      }`}>
                        {row.type === 1 ? '微信支付' : '支付宝'}
                      </span>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-col">
                        <span className="font-semibold text-xs text-primary">¥ {row.really_price}</span>
                        <span className="text-[11px] text-muted-foreground line-through">¥ {row.price}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <StatusBadge state={row.state} stateText={row.state_text} />
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground font-mono">
                      {row.create_time}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1.5">
                        {row.state !== 1 && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => reissueMutation.mutate(row.id)}
                            disabled={reissueMutation.isPending}
                            className="h-8 px-2.5 text-xs text-primary hover:bg-primary/10"
                          >
                            <RotateCcw className="size-3" />
                            补单
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => deleteMutation.mutate(row.id)}
                          disabled={deleteMutation.isPending}
                          className="h-8 px-2.5 text-xs text-destructive hover:bg-destructive/10"
                        >
                          <Trash2 className="size-3" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-12 text-muted-foreground text-xs">
                    暂无匹配的订单记录
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>

          {/* 分页控制栏 */}
          <div className="flex items-center justify-between mt-4 px-1">
            <span className="text-xs text-muted-foreground">
              第 {page} / {totalPages} 页
            </span>
            <div className="flex items-center gap-1.5">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="h-8 px-2.5 rounded-xl"
              >
                <ChevronLeft className="size-4" />
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="h-8 px-2.5 rounded-xl"
              >
                <ChevronRight className="size-4" />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
