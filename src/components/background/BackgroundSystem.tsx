import { useEffect } from 'react'
import { BackgroundLayer } from './BackgroundLayer'
import { useBackground } from './BackgroundProvider'
import { analyzeBackgroundTheme, createFallbackBackgroundTheme } from './backgroundTheme'
import { backgrounds as defaultBackgrounds, backgroundSystemDefaults } from './backgrounds'
import type { BackgroundConfig } from './background.types'

const imageLoadTimeout = 12_000

export type BackgroundSystemProps = {
  /** 可选的单张首选背景；加载失败后仍会回退至背景池。 */
  config?: BackgroundConfig
  /** 默认使用集中配置的 Jiuin 背景池。 */
  backgrounds?: readonly string[]
  backgroundBlur?: number
  backgroundOverlayOpacity?: number
}

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value))
}

function randomIndex(maximum: number) {
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

function shuffleBackgrounds(pool: readonly string[]) {
  const shuffled = [...new Set(pool.map((url) => url.trim()).filter(Boolean))]

  for (let index = shuffled.length - 1; index > 0; index -= 1) {
    const swapIndex = randomIndex(index + 1)
    ;[shuffled[index], shuffled[swapIndex]] = [shuffled[swapIndex], shuffled[index]]
  }

  return shuffled
}

function preloadBackgroundImage(url: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const image = new Image()
    let settled = false
    const timeout = window.setTimeout(
      () => fail(new Error('Background image loading timed out.')),
      imageLoadTimeout,
    )

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

    function fail(error: Error) {
      if (cleanup()) {
        reject(error)
      }
    }

    image.decoding = 'async'
    image.onload = () => {
      if (image.naturalWidth === 0 || image.naturalHeight === 0) {
        fail(new Error('Background image has no dimensions.'))
        return
      }

      image.decode().then(
        succeed,
        succeed,
      )
    }
    image.onerror = () => fail(new Error('Background image could not be loaded.'))
    image.src = url
  })
}

/**
 * 全站随机背景控制器：随机选择、异步加载、失败回退、主题取色与下一张的低优先级预热。
 */
export function BackgroundSystem({
  config,
  backgrounds = defaultBackgrounds,
  backgroundBlur,
  backgroundOverlayOpacity,
}: BackgroundSystemProps) {
  const { setBackground, updateBackground } = useBackground()
  const requestedImage = config?.image ?? config?.background
  const requestedId = config?.id
  const blur = clamp(
    config?.blur ?? config?.backgroundBlur ?? backgroundBlur ?? backgroundSystemDefaults.backgroundBlur,
    0,
    40,
  )
  const overlayOpacity = clamp(
    config?.overlayOpacity ?? config?.backgroundOverlayOpacity ?? backgroundOverlayOpacity ?? backgroundSystemDefaults.backgroundOverlayOpacity,
    0,
    1,
  )
  const opacity = clamp(config?.opacity ?? backgroundSystemDefaults.backgroundImageOpacity, 0, 1)
  const brightness = clamp(config?.brightness ?? backgroundSystemDefaults.backgroundImageBrightness, 0.2, 1.5)
  const transition = config?.transition ?? 'crossfade'
  const transitionDuration = clamp(
    config?.transitionDuration ?? backgroundSystemDefaults.backgroundTransitionDuration,
    500,
    1_000,
  )
  const poolKey = backgrounds.join('\u001f')

  useEffect(() => {
    let cancelled = false
    let prefetchTimer: number | undefined
    const pool = shuffleBackgrounds(backgrounds)
    const candidates = requestedImage
      ? [requestedImage, ...pool.filter((url) => url !== requestedImage)]
      : pool

    async function selectBackground() {
      for (const image of candidates) {
        try {
          await preloadBackgroundImage(image)

          if (cancelled) {
            return
          }

          const id = image === requestedImage && requestedId ? requestedId : `background:${image}`
          const theme = createFallbackBackgroundTheme(image)

          setBackground({
            id,
            image,
            blur,
            opacity,
            brightness,
            overlayOpacity,
            transition,
            transitionDuration,
            theme: { mode: 'auto', overrides: theme },
          })

          void analyzeBackgroundTheme(image).then((overrides) => {
            if (!cancelled) {
              updateBackground({
                id,
                image,
                theme: { mode: 'auto', overrides },
              })
            }
          }).catch(() => {
            // CORS 取色失败时保留已应用的、基于 URL 的回退主题。
          })

          const nextImage = candidates.find((candidate) => candidate !== image)

          if (nextImage) {
            prefetchTimer = window.setTimeout(() => {
              void preloadBackgroundImage(nextImage).catch(() => {
                // 预热只是缓存优化；失败会在真正选择时走正常回退逻辑。
              })
            }, 1_200)
          }

          return
        } catch {
          // 候选图片不可用时继续尝试随机打乱后的下一张。
        }
      }
    }

    void selectBackground()

    return () => {
      cancelled = true

      if (prefetchTimer !== undefined) {
        window.clearTimeout(prefetchTimer)
      }
    }
  }, [backgrounds, blur, brightness, opacity, overlayOpacity, poolKey, requestedId, requestedImage, setBackground, transition, transitionDuration, updateBackground])

  return <BackgroundLayer />
}
