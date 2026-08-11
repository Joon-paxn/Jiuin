import { useEffect, useState } from 'react'
import type { CSSProperties } from 'react'
import { classNames } from '../../utils/classNames'

export type ScrollProgressProps = {
  /** 可选受控进度值（0–100）；省略时自动读取页面滚动位置。 */
  value?: number
  className?: string
}

type ScrollProgressStyle = CSSProperties & {
  '--scroll-progress'?: string
}

function clampProgress(value: number) {
  if (!Number.isFinite(value)) {
    return 0
  }

  return Math.min(100, Math.max(0, value))
}

function getPageProgress() {
  if (typeof window === 'undefined') {
    return 0
  }

  const root = document.documentElement
  const pageHeight = Math.max(root.scrollHeight, document.body?.scrollHeight ?? 0)
  const scrollableHeight = pageHeight - window.innerHeight

  if (scrollableHeight <= 0) {
    return 0
  }

  if (window.scrollY <= 0) {
    return 0
  }

  return Math.max(1, clampProgress(Math.round((window.scrollY / scrollableHeight) * 100)))
}

function usePageProgress(enabled: boolean) {
  const [progress, setProgress] = useState(getPageProgress)

  useEffect(() => {
    if (!enabled) {
      return
    }

    let frameHandle = 0
    const syncProgress = () => {
      frameHandle = 0
      const nextProgress = getPageProgress()
      setProgress((currentProgress) => (
        currentProgress === nextProgress ? currentProgress : nextProgress
      ))
    }
    const requestSync = () => {
      if (!frameHandle) {
        frameHandle = window.requestAnimationFrame(syncProgress)
      }
    }

    requestSync()
    window.addEventListener('scroll', requestSync, { passive: true })
    window.addEventListener('resize', requestSync, { passive: true })

    return () => {
      window.removeEventListener('scroll', requestSync)
      window.removeEventListener('resize', requestSync)
      if (frameHandle) {
        window.cancelAnimationFrame(frameHandle)
      }
    }
  }, [enabled])

  return progress
}

/** 页面阅读进度与返回顶部按钮。 */
export function ScrollProgress({ value, className }: ScrollProgressProps) {
  const isControlled = typeof value === 'number'
  const automaticProgress = usePageProgress(!isControlled)
  const progress = isControlled ? clampProgress(value) : automaticProgress
  const isVisible = progress > 0
  const style: ScrollProgressStyle = { '--scroll-progress': `${progress}%` }

  const scrollToTop = () => {
    if (!isVisible) {
      return
    }

    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    window.scrollTo({ top: 0, behavior: prefersReducedMotion ? 'auto' : 'smooth' })
  }

  return (
    <button
      aria-hidden={!isVisible || undefined}
      aria-label={`返回页面顶部，当前阅读进度 ${progress}%`}
      className={classNames('scroll-progress', className)}
      data-visible={isVisible}
      onClick={scrollToTop}
      style={style}
      tabIndex={isVisible ? undefined : -1}
      type="button"
    >
      <span className="scroll-progress__icon" aria-hidden="true">↑</span>
      <span className="scroll-progress__value" aria-hidden="true">{progress}%</span>
    </button>
  )
}
