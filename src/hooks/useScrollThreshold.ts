import { useEffect, useState } from 'react'

function isPastThreshold(threshold: number) {
  return typeof window !== 'undefined' && window.scrollY > threshold
}

/** 供粘性界面读取简洁的滚动状态；不承担滚动进度计算。 */
export function useScrollThreshold(threshold = 24) {
  const [isPast, setIsPast] = useState(() => isPastThreshold(threshold))

  useEffect(() => {
    const syncScrollState = () => setIsPast(isPastThreshold(threshold))

    syncScrollState()
    window.addEventListener('scroll', syncScrollState, { passive: true })

    return () => window.removeEventListener('scroll', syncScrollState)
  }, [threshold])

  return isPast
}
