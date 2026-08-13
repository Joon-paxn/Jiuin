import { useCallback, useEffect, useRef, useState } from 'react'

const webLoadingAsset = '/loading/web_Loading.png'
const loadingAssets = Array.from({ length: 12 }, (_, index) => `/loading/Loading${index + 1}.png`)
const finalImageAsset = 'https://picui.ogmua.cn/s1/2026/08/13/6a7ddf4caa020.webp'

const MASK_HOLD = 140
const MASK_STEP_DURATION = 340
const ARTWORK_ENTER_DURATION = 620
const ARTWORK_HOLD_DURATION = 1_040
const EXIT_DURATION = 540
const REDUCED_MOTION_DURATION = 140

export type LoadingStage = 'mask' | 'artwork'

export type LoadingAssets = {
  webLoading: string
  artwork: string
  finalImage: string
}

type OpeningSequenceOptions = {
  enabled: boolean
  reducedMotion: boolean
}

function randomIndex(maximum: number) {
  if (maximum <= 1) {
    return 0
  }

  const browserCrypto = globalThis.crypto

  if (browserCrypto?.getRandomValues) {
    const upperBound = 0x1_0000_0000
    const unbiasedLimit = upperBound - (upperBound % maximum)
    const buffer = new Uint32Array(1)

    do {
      browserCrypto.getRandomValues(buffer)
    } while (buffer[0] >= unbiasedLimit)

    return buffer[0] % maximum
  }

  return Math.floor(Math.random() * maximum)
}

function preloadImage(source: string, timeoutMs = 3_000) {
  return new Promise<boolean>((resolve) => {
    const image = new Image()
    let settled = false
    const finish = (loaded: boolean) => {
      if (settled) {
        return
      }

      settled = true
      window.clearTimeout(timeout)
      image.onload = null
      image.onerror = null
      resolve(loaded)
    }
    const timeout = window.setTimeout(() => finish(false), timeoutMs)

    image.decoding = 'async'
    image.onload = () => {
      if (image.naturalWidth === 0 || image.naturalHeight === 0) {
        finish(false)
        return
      }

      image.decode().then(() => finish(true), () => finish(true))
    }
    image.onerror = () => finish(false)
    image.src = source
  })
}

async function selectPreloadedArtwork(preferred: string) {
  if (await preloadImage(preferred)) {
    return preferred
  }

  const candidates = loadingAssets.filter((asset) => asset !== preferred)

  while (candidates.length > 0) {
    const index = randomIndex(candidates.length)
    const [candidate] = candidates.splice(index, 1)

    if (await preloadImage(candidate)) {
      return candidate
    }
  }

  return preferred
}

function clearTimers(timers: number[]) {
  timers.forEach((timer) => window.clearTimeout(timer))
  timers.length = 0
}

function waitFor(timers: number[], duration: number) {
  return new Promise<void>((resolve) => {
    const timer = window.setTimeout(() => {
      const index = timers.indexOf(timer)
      if (index >= 0) {
        timers.splice(index, 1)
      }
      resolve()
    }, duration)
    timers.push(timer)
  })
}

/**
 * Opening state machine. The mask and artwork phases remain deliberately
 * separate: the first reveals one complete web_Loading image; the second
 * selects exactly one independent Loading1–Loading12 asset.
 */
export function useOpeningSequence({ enabled, reducedMotion }: OpeningSequenceOptions) {
  const [isOpening, setIsOpening] = useState(enabled)
  const [isPageReady, setIsPageReady] = useState(!enabled)
  const [isExiting, setIsExiting] = useState(false)
  const [maskStep, setMaskStep] = useState(0)
  const [stage, setStage] = useState<LoadingStage>('mask')
  const [assets, setAssets] = useState<LoadingAssets>(() => ({
    webLoading: webLoadingAsset,
    artwork: loadingAssets[0],
    finalImage: finalImageAsset,
  }))
  const timersRef = useRef<number[]>([])
  const hasFinishedRef = useRef(false)

  useEffect(() => {
    if (!isOpening) {
      return
    }

    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = previousOverflow
    }
  }, [isOpening])

  const finishOpening = useCallback(() => {
    if (hasFinishedRef.current) {
      return
    }

    hasFinishedRef.current = true
    clearTimers(timersRef.current)
    setStage('artwork')
    setIsPageReady(true)
    setIsExiting(true)
    timersRef.current.push(window.setTimeout(() => setIsOpening(false), reducedMotion ? REDUCED_MOTION_DURATION : EXIT_DURATION))
  }, [reducedMotion])

  useEffect(() => {
    if (!enabled) {
      setIsOpening(false)
      setIsPageReady(true)
      return
    }

    let cancelled = false
    const duration = reducedMotion ? REDUCED_MOTION_DURATION : MASK_STEP_DURATION
    hasFinishedRef.current = false
    setAssets({ webLoading: webLoadingAsset, artwork: loadingAssets[0], finalImage: finalImageAsset })
    setIsOpening(true)
    setIsPageReady(false)
    setIsExiting(false)
    setStage('mask')
    setMaskStep(0)

    // The base image is the only loading asset requested before the mask completes.
    void preloadImage(webLoadingAsset)
    void preloadImage(finalImageAsset)

    async function runSequence() {
      await waitFor(timersRef.current, reducedMotion ? 0 : MASK_HOLD)

      for (let step = 1; step <= 4; step += 1) {
        if (cancelled || hasFinishedRef.current) {
          return
        }
        setMaskStep(step)
        await waitFor(timersRef.current, duration)
      }

      if (cancelled || hasFinishedRef.current) {
        return
      }

      // Loading1–12 is selected and loaded only after web_Loading is complete.
      const selectedArtwork = loadingAssets[randomIndex(loadingAssets.length)]
      const artwork = await selectPreloadedArtwork(selectedArtwork)
      if (cancelled || hasFinishedRef.current) {
        return
      }
      setAssets({ webLoading: webLoadingAsset, artwork, finalImage: finalImageAsset })
      setStage('artwork')
      await waitFor(timersRef.current, reducedMotion ? REDUCED_MOTION_DURATION : ARTWORK_ENTER_DURATION + ARTWORK_HOLD_DURATION)

      if (!cancelled) {
        finishOpening()
      }
    }

    void runSequence()

    return () => {
      cancelled = true
      clearTimers(timersRef.current)
    }
  }, [enabled, finishOpening, reducedMotion])

  return { assets, finishOpening, isExiting, isOpening, isPageReady, maskStep, stage }
}
