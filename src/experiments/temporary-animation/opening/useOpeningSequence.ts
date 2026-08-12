import { useCallback, useEffect, useRef, useState } from 'react'
import { analyzeBackgroundTheme, createFallbackBackgroundTheme } from '../../../components/background/backgroundTheme'
import { backgrounds } from '../../../components/background/backgrounds'
import type { BackgroundThemeOverrides } from '../../../components/background/background.types'

const BACKGROUND_LOAD_TIMEOUT = 1_500
// The cover is visual polish, not a gate for application startup. Keep the
// aggregate retry window bounded so a degraded image CDN cannot delay entry.
const BACKGROUND_LOAD_BUDGET = 3_500
const OPENING_EXIT_DURATION = 680
const REDUCED_MOTION_EXIT_DURATION = 200
const BRAND_SETTLE_DELAY = 180
const HANDOFF_SETTLE_DELAY = 260
const MINIMUM_PRESENTATION_DURATION = 1_680

export type LoadingStage = 'space' | 'light' | 'interface' | 'entering'

export type LoadingBackground = {
  image?: string
  imageReady: boolean
  theme: BackgroundThemeOverrides
}

type OpeningSequenceOptions = {
  enabled: boolean
  reducedMotion: boolean
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

function elapsedSince(startedAt: number) {
  return Date.now() - startedAt
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

function shuffledBackgrounds() {
  const candidates = [...new Set(backgrounds.map((url) => url.trim()).filter(Boolean))]

  for (let index = candidates.length - 1; index > 0; index -= 1) {
    const nextIndex = randomIndex(index + 1)
    ;[candidates[index], candidates[nextIndex]] = [candidates[nextIndex], candidates[index]]
  }

  return candidates
}

function preloadImage(url: string, timeoutMs: number) {
  return new Promise<void>((resolve, reject) => {
    const image = new Image()
    let settled = false
    const timeout = window.setTimeout(() => fail(), timeoutMs)

    function cleanup() {
      if (settled) {
        return false
      }

      settled = true
      window.clearTimeout(timeout)
      image.onload = null
      image.onerror = null
      return true
    }

    function succeed() {
      if (cleanup()) {
        resolve()
      }
    }

    function fail() {
      if (cleanup()) {
        reject(new Error('Loading background image could not be loaded.'))
      }
    }

    image.decoding = 'async'
    image.onload = () => {
      if (image.naturalWidth === 0 || image.naturalHeight === 0) {
        fail()
        return
      }

      image.decode().then(succeed, succeed)
    }
    image.onerror = fail
    image.src = url
  })
}

function createInitialLoadingBackground(): LoadingBackground {
  const image = backgrounds[randomIndex(backgrounds.length)]

  return {
    image,
    imageReady: false,
    theme: createFallbackBackgroundTheme(image),
  }
}

/**
 * The startup cover owns its own image loading rather than reusing BackgroundLayer.
 * That lets the page initialise in parallel and leaves the main background controller
 * free to make its own selection after the cover has begun to leave.
 */
export function useOpeningSequence({ enabled, reducedMotion }: OpeningSequenceOptions) {
  const [isOpening, setIsOpening] = useState(enabled)
  const [isReady, setIsReady] = useState(false)
  const [isPageReady, setIsPageReady] = useState(!enabled)
  const [isExiting, setIsExiting] = useState(false)
  const [stage, setStage] = useState<LoadingStage>('space')
  const [background, setBackground] = useState<LoadingBackground>(createInitialLoadingBackground)
  const timersRef = useRef<number[]>([])
  const exitingRef = useRef(false)
  const completedRef = useRef(false)

  const beginExit = useCallback(() => {
    if (exitingRef.current || completedRef.current) {
      return
    }

    exitingRef.current = true
    clearTimers(timersRef.current)
    setStage('entering')
    setIsPageReady(true)
    setIsExiting(true)

    const exitDuration = reducedMotion ? REDUCED_MOTION_EXIT_DURATION : OPENING_EXIT_DURATION
    timersRef.current.push(window.setTimeout(() => {
      completedRef.current = true
      setIsOpening(false)
    }, exitDuration))
  }, [reducedMotion])

  useEffect(() => {
    if (!enabled) {
      setIsOpening(false)
      setIsReady(false)
      setIsPageReady(true)
      setIsExiting(false)
      return
    }

    let cancelled = false
    const candidates = shuffledBackgrounds()
    const initialImage = candidates[0]

    exitingRef.current = false
    completedRef.current = false
    setIsOpening(true)
    setIsReady(false)
    setIsPageReady(false)
    setIsExiting(false)
    setStage('space')
    setBackground({
      image: initialImage,
      imageReady: false,
      theme: createFallbackBackgroundTheme(initialImage),
    })

    const animationFrame = window.requestAnimationFrame(() => {
      if (!cancelled) {
        setIsReady(true)
      }
    })

    async function prepareOpening() {
      const startedAt = Date.now()

      if (!reducedMotion) {
        await waitFor(timersRef.current, BRAND_SETTLE_DELAY)
      }

      if (cancelled || exitingRef.current) {
        return
      }

      setStage('light')
      const deadline = Date.now() + BACKGROUND_LOAD_BUDGET

      for (const image of candidates) {
        const remaining = deadline - Date.now()

        if (remaining <= 0 || cancelled || exitingRef.current) {
          break
        }

        setBackground({
          image,
          imageReady: false,
          theme: createFallbackBackgroundTheme(image),
        })

        try {
          await preloadImage(image, Math.min(BACKGROUND_LOAD_TIMEOUT, remaining))

          if (cancelled || exitingRef.current) {
            return
          }

          const fallbackTheme = createFallbackBackgroundTheme(image)
          setBackground({ image, imageReady: true, theme: fallbackTheme })

          // Pixel analysis is a progressive enhancement. Cross-origin restrictions or
          // an unavailable image must never prevent the site from entering.
          void analyzeBackgroundTheme(image).then((theme) => {
            if (!cancelled && !exitingRef.current) {
              setBackground((current) => current.image === image
                ? { ...current, theme }
                : current)
            }
          }).catch(() => {
            // The deterministic URL-derived theme remains in use as a safe fallback.
          })

          break
        } catch {
          // Continue through the shuffled pool. The visual cover remains on its
          // gradient fallback while the next candidate is being tried.
        }
      }

      if (cancelled || exitingRef.current) {
        return
      }

      setStage('interface')

      if (!reducedMotion) {
        const remainingPresentationTime = Math.max(
          HANDOFF_SETTLE_DELAY,
          MINIMUM_PRESENTATION_DURATION - elapsedSince(startedAt),
        )
        await waitFor(timersRef.current, remainingPresentationTime)
      }

      if (!cancelled) {
        beginExit()
      }
    }

    void prepareOpening()

    return () => {
      cancelled = true
      window.cancelAnimationFrame(animationFrame)
      clearTimers(timersRef.current)
    }
  }, [beginExit, enabled, reducedMotion])

  return {
    background,
    finishOpening: beginExit,
    isExiting,
    isOpening,
    isPageReady,
    isReady,
    stage,
  }
}
