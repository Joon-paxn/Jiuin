import { useCallback, useEffect, useRef, useState } from 'react'

const PAGE_REVEAL_DELAY = 1750
const OPENING_COMPLETE_DELAY = 2420

type OpeningSequenceOptions = {
  enabled: boolean
  reducedMotion: boolean
}

function clearTimers(timers: number[]) {
  timers.forEach((timer) => window.clearTimeout(timer))
  timers.length = 0
}

export function useOpeningSequence({ enabled, reducedMotion }: OpeningSequenceOptions) {
  const [isOpening, setIsOpening] = useState(enabled)
  const [isReady, setIsReady] = useState(false)
  const [isPageReady, setIsPageReady] = useState(!enabled)
  const timersRef = useRef<number[]>([])

  const finishOpening = useCallback(() => {
    clearTimers(timersRef.current)
    setIsPageReady(true)
    setIsOpening(false)
  }, [])

  useEffect(() => {
    if (!enabled) {
      return
    }

    const animationFrame = window.requestAnimationFrame(() => setIsReady(true))
    const revealDelay = reducedMotion ? 0 : PAGE_REVEAL_DELAY
    const completeDelay = reducedMotion ? 0 : OPENING_COMPLETE_DELAY

    timersRef.current.push(
      window.setTimeout(() => setIsPageReady(true), revealDelay),
      window.setTimeout(() => setIsOpening(false), completeDelay),
    )

    return () => {
      window.cancelAnimationFrame(animationFrame)
      clearTimers(timersRef.current)
    }
  }, [enabled, reducedMotion])

  return {
    finishOpening,
    isOpening,
    isPageReady,
    isReady,
  }
}
