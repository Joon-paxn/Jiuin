import { useCallback, useEffect, useRef, useState } from 'react'
import { useLoadingResourceManager } from './useLoadingResourceManager'

const webLoadingAsset = '/loading/web_Loading.png'
const MASK_HOLD = 140
const MASK_STEP_DURATION = 340
const EXIT_DURATION = 540
const REDUCED_MOTION_DURATION = 140

export type LoadingStage = 'mask' | 'artwork'
export type LoadingState = 'INIT' | 'LOADING' | 'READY' | 'EXITING' | 'COMPLETED' | 'ERROR'
export type LoadingAssets = { webLoading: string; artwork: string }

type OpeningSequenceOptions = { enabled: boolean; reducedMotion: boolean }

function wait(timers: number[], duration: number) {
  return new Promise<void>((resolve) => {
    const timer = window.setTimeout(() => {
      const index = timers.indexOf(timer)
      if (index >= 0) timers.splice(index, 1)
      resolve()
    }, duration)
    timers.push(timer)
  })
}

function clearTimers(timers: number[]) {
  timers.forEach((timer) => window.clearTimeout(timer))
  timers.length = 0
}

export function useOpeningSequence({ enabled, reducedMotion }: OpeningSequenceOptions) {
  const manager = useLoadingResourceManager(enabled)
  const [isOpening, setIsOpening] = useState(enabled)
  const [isPageReady, setIsPageReady] = useState(!enabled)
  const [isExiting, setIsExiting] = useState(false)
  const [maskStep, setMaskStep] = useState(0)
  const [stage, setStage] = useState<LoadingStage>('mask')
  const [state, setState] = useState<LoadingState>(enabled ? 'INIT' : 'COMPLETED')
  const timersRef = useRef<number[]>([])
  const cancelledRef = useRef(false)
  const exitingRef = useRef(false)
  const criticalReadyRef = useRef(manager.allCriticalResourcesLoaded)

  useEffect(() => {
    criticalReadyRef.current = manager.allCriticalResourcesLoaded
  }, [manager.allCriticalResourcesLoaded])

  const finishOpening = useCallback(() => {
    if (exitingRef.current || !criticalReadyRef.current) return
    exitingRef.current = true
    setState('EXITING')
    setIsPageReady(true)
    setIsExiting(true)
    timersRef.current.push(window.setTimeout(() => {
      setIsOpening(false)
      setState('COMPLETED')
    }, reducedMotion ? REDUCED_MOTION_DURATION : EXIT_DURATION))
  }, [reducedMotion])

  useEffect(() => {
    if (!enabled) {
      setIsOpening(false)
      setIsPageReady(true)
      setState('COMPLETED')
      return
    }

    cancelledRef.current = false
    exitingRef.current = false
    setIsOpening(true)
    setIsPageReady(false)
    setIsExiting(false)
    setState('LOADING')
    setStage('mask')
    setMaskStep(0)

    async function runLoop() {
      await wait(timersRef.current, reducedMotion ? 0 : MASK_HOLD)
      while (!cancelledRef.current && !exitingRef.current) {
        for (let step = 1; step <= 4; step += 1) {
          if (cancelledRef.current || exitingRef.current) return
          setStage('mask')
          setMaskStep(step)
          await wait(timersRef.current, reducedMotion ? REDUCED_MOTION_DURATION : MASK_STEP_DURATION)
        }

        if (cancelledRef.current || exitingRef.current) return
        setStage('artwork')
        if (criticalReadyRef.current) setState('READY')
        await wait(timersRef.current, reducedMotion ? REDUCED_MOTION_DURATION : MASK_STEP_DURATION)

        if (criticalReadyRef.current) {
          finishOpening()
          return
        }
        setMaskStep(0)
      }
    }

    void runLoop()
    return () => {
      cancelledRef.current = true
      clearTimers(timersRef.current)
    }
  }, [enabled, finishOpening, reducedMotion])

  useEffect(() => {
    if (manager.error && !exitingRef.current) setState('ERROR')
  }, [manager.error])

  return {
    assets: { webLoading: webLoadingAsset, artwork: manager.artwork },
    error: manager.error,
    finishOpening,
    isExiting,
    isOpening,
    isPageReady,
    maskStep,
    retry: manager.retry,
    stage,
    state,
    resources: manager.resources,
  }
}
