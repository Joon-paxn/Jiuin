import { useEffect } from 'react'
import { BackgroundLayer } from './BackgroundLayer'
import { useBackground } from './BackgroundProvider'
import { loadCurrentBackground } from './backgroundResource'
import { analyzeBackgroundTheme, createFallbackBackgroundTheme } from './backgroundTheme'
import { backgroundSystemDefaults } from './backgrounds'
import type { BackgroundConfig } from './background.types'

export type BackgroundSystemProps = {
  config?: BackgroundConfig
  backgroundBlur?: number
  backgroundOverlayOpacity?: number
}

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value))
}

// Background selection belongs to the backend. This component only consumes
// the single resource selected for this page lifetime.
export function BackgroundSystem({
  config,
  backgroundBlur,
  backgroundOverlayOpacity,
}: BackgroundSystemProps) {
  const { setBackground, updateBackground } = useBackground()
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

  useEffect(() => {
    let cancelled = false

    void loadCurrentBackground().then(({ url, image }) => {
      if (cancelled) return

      const id = `background:${url}`
      setBackground({
        id,
        image: url,
        blur,
        opacity,
        brightness,
        overlayOpacity,
        transition,
        transitionDuration,
        theme: { mode: 'auto', overrides: createFallbackBackgroundTheme(url) },
      })

      // Theme analysis receives the already-loaded image. It never constructs
      // a second Image instance or requests the CDN URL again.
      const overrides = analyzeBackgroundTheme(image, url)
      if (!cancelled) {
        updateBackground({
          id,
          image: url,
          theme: { mode: 'auto', overrides },
        })
      }
    }).catch(() => {
      // The Loading scene retains its white fallback after bounded retries.
      // A later remount may safely request a new server-selected URL.
    })

    return () => { cancelled = true }
  }, [blur, brightness, opacity, overlayOpacity, setBackground, transition, transitionDuration, updateBackground])

  return <BackgroundLayer />
}
