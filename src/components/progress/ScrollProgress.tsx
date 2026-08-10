import type { CSSProperties } from 'react'
import { classNames } from '../../utils/classNames'

export type ScrollProgressProps = {
  /** 由未来的滚动系统传入 0–100 数值；本阶段不监听页面滚动。 */
  value?: number
  className?: string
}

type ScrollProgressStyle = CSSProperties & {
  '--scroll-progress'?: number
}

function clampProgress(value: number) {
  return Math.min(100, Math.max(0, value))
}

/** 页面顶部的视觉占位与未来进度接口。 */
export function ScrollProgress({ value, className }: ScrollProgressProps) {
  const progress = typeof value === 'number' ? clampProgress(value) : undefined
  const style: ScrollProgressStyle | undefined = progress === undefined
    ? undefined
    : { '--scroll-progress': progress / 100 }

  return (
    <div
      aria-hidden={progress === undefined ? true : undefined}
      aria-label={progress === undefined ? undefined : '页面滚动进度'}
      aria-valuemax={progress === undefined ? undefined : 100}
      aria-valuemin={progress === undefined ? undefined : 0}
      aria-valuenow={progress}
      className={classNames('scroll-progress', className)}
      role={progress === undefined ? undefined : 'progressbar'}
    >
      <span className="scroll-progress__value" style={style} />
    </div>
  )
}
