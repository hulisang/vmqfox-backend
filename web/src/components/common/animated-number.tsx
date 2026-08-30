import React, { useEffect, useRef, useState } from 'react'

interface AnimatedNumberProps {
  value: number
  prefix?: string
  suffix?: string
  decimals?: number
  duration?: number
  className?: string
}

export const AnimatedNumber: React.FC<AnimatedNumberProps> = ({
  value,
  prefix = '',
  suffix = '',
  decimals = 2,
  duration = 800,
  className = '',
}) => {
  const [displayValue, setDisplayValue] = useState(0)
  // 用 ref 保存动画起点，避免把 displayValue 写进依赖数组导致每帧重启动画
  const displayValueRef = useRef(0)

  useEffect(() => {
    let startTimestamp: number | null = null
    const startValue = displayValueRef.current
    let frameId = 0

    const step = (timestamp: number) => {
      if (startTimestamp === null) startTimestamp = timestamp
      const progress = Math.min((timestamp - startTimestamp) / duration, 1)
      const easeProgress = 1 - Math.pow(1 - progress, 3) // easeOutCubic
      const current = startValue + (value - startValue) * easeProgress
      displayValueRef.current = current
      setDisplayValue(current)

      if (progress < 1) {
        frameId = window.requestAnimationFrame(step)
      }
    }

    frameId = window.requestAnimationFrame(step)
    // 组件卸载或数值再次变化时取消未完成的动画帧
    return () => window.cancelAnimationFrame(frameId)
  }, [value, duration])

  return (
    <span className={className}>
      {prefix}
      {displayValue.toFixed(decimals)}
      {suffix}
    </span>
  )
}
