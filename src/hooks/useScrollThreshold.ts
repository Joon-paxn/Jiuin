import { useEffect, useState } from 'react'

function isPastThreshold(threshold: number) {
  return typeof window !== 'undefined' && window.scrollY > threshold
}

/** 供粘性界面读取简洁的滚动状态；不承担滚动进度计算。 */
export function useScrollThreshold(threshold = 24) {
  const [isPast, setIsPast] = useState(() => isPastThreshold(threshold))

  useEffect(() => {
    let frameHandle = 0

    const syncScrollState = () => {
      frameHandle = 0
      const nextValue = isPastThreshold(threshold)
      setIsPast((currentValue) => (currentValue === nextValue ? currentValue : nextValue))
    }

    const requestSync = () => {
      if (!frameHandle) {
        frameHandle = window.requestAnimationFrame(syncScrollState)
      }
    }

    syncScrollState()
    window.addEventListener('scroll', requestSync, { passive: true })

    return () => {
      window.removeEventListener('scroll', requestSync)
      if (frameHandle) {
        window.cancelAnimationFrame(frameHandle)
      }
    }
  }, [threshold])

  return isPast
}
