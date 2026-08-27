import React from 'react'
import { BadgeCheck, Clock, XCircle, HelpCircle } from 'lucide-react'
import { cn } from '@/lib/utils'

interface StatusBadgeProps {
  state: number
  stateText?: string
  className?: string
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ state, stateText, className }) => {
  // state 0: 未支付, 1: 已支付, -1: 已过期/已关闭
  let bg = 'bg-muted/80 text-muted-foreground border-border/80'
  let Icon = HelpCircle
  let text = stateText || '未知'

  if (state === 1) {
    bg = 'bg-primary/15 text-primary border-primary/20'
    Icon = BadgeCheck
    text = stateText || '已支付'
  } else if (state === 0) {
    bg = 'bg-amber-500/15 text-amber-600 dark:text-amber-400 border-amber-500/20'
    Icon = Clock
    text = stateText || '等待支付'
  } else if (state === -1 || state === 2) {
    bg = 'bg-destructive/15 text-destructive border-destructive/20'
    Icon = XCircle
    text = stateText || '已关闭/已超时'
  }

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-xl text-xs font-medium border shadow-2xs backdrop-blur-2xs transition-all',
        bg,
        className
      )}
    >
      <Icon className="size-3.5 shrink-0" />
      <span>{text}</span>
    </span>
  )
}
