import { useEffect, useRef, useState, type CSSProperties } from 'react'
import { classNames } from '../../utils/classNames'
import { resolveBackgroundConfig, type BackgroundConfig, type ResolvedBackgroundConfig } from './background.types'
import { useOptionalBackground } from './BackgroundProvider'

export type { BackgroundConfig } from './background.types'

export type BackgroundLayerProps = {
  config?: BackgroundConfig
  className?: string
  style?: CSSProperties
}

type BackgroundLayerStyle = CSSProperties & {
  '--background-blur': string
  '--background-overlay-opacity': number
  '--background-transition-duration': string
}

type BackgroundImageStyle = CSSProperties & {
  '--background-image-brightness': number
  '--background-image-opacity': number
}

type BackgroundSlot = 'primary' | 'secondary'

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value))
}

function imageStyle(image: string | undefined, settings: ResolvedBackgroundConfig): BackgroundImageStyle {
  return {
    '--background-image-brightness': clamp(settings.brightness, 0.2, 1.5),
    '--background-image-opacity': clamp(settings.opacity, 0, 1),
    ...(image ? { backgroundImage: `url(${JSON.stringify(image)})` } : {}),
  }
}

/**
 * 全站背景视觉层：仅绘制背景，双槽位负责图片交叉淡入。
 * 随机选择、加载与主题同步由 BackgroundSystem 负责。
 */
export function BackgroundLayer({ config, className, style }: BackgroundLayerProps) {
  const managedBackground = useOptionalBackground()
  const settings = config
    ? resolveBackgroundConfig(config)
    : managedBackground?.background ?? resolveBackgroundConfig()
  const [primaryImage, setPrimaryImage] = useState(settings.image)
  const [secondaryImage, setSecondaryImage] = useState<string | undefined>()
  const [activeSlot, setActiveSlot] = useState<BackgroundSlot>('primary')
  const primaryImageRef = useRef<HTMLDivElement>(null)
  const secondaryImageRef = useRef<HTMLDivElement>(null)
  const activeImage = activeSlot === 'primary' ? primaryImage : secondaryImage

  useEffect(() => {
    if (settings.image === activeImage) {
      return
    }

    if (settings.transition === 'instant') {
      setPrimaryImage(settings.image)
      setSecondaryImage(undefined)
      setActiveSlot('primary')
      return
    }

    if (activeSlot === 'primary') {
      setSecondaryImage(settings.image)
      setActiveSlot('secondary')
      return
    }

    setPrimaryImage(settings.image)
    setActiveSlot('primary')
  }, [activeImage, activeSlot, settings.image, settings.transition])

  useEffect(() => {
    const primary = primaryImageRef.current
    const secondary = secondaryImageRef.current
    if (!primary || !secondary) return

    const maxMove = 12
    const interpolation = 0.08
    let targetX = 0
    let targetY = 0
    let currentX = 0
    let currentY = 0
    let frame = 0
    let running = false

    const applyTransform = () => {
      const transform = `translate3d(${currentX.toFixed(3)}px, ${currentY.toFixed(3)}px, 0) scale(1.08)`
      primary.style.transform = transform
      secondary.style.transform = transform
    }

    const tick = () => {
      currentX += (targetX - currentX) * interpolation
      currentY += (targetY - currentY) * interpolation
      applyTransform()

      const settled = Math.abs(targetX - currentX) < 0.01 && Math.abs(targetY - currentY) < 0.01
      if (settled) {
        currentX = targetX
        currentY = targetY
        applyTransform()
        running = false
        frame = 0
        return
      }
      frame = window.requestAnimationFrame(tick)
    }

    const schedule = () => {
      if (running) return
      running = true
      frame = window.requestAnimationFrame(tick)
    }

    const resetToCenter = () => {
      targetX = 0
      targetY = 0
      schedule()
    }

    const pointerMove = (event: PointerEvent) => {
      if (!window.innerWidth || !window.innerHeight) return
      const normalizedX = (event.clientX / window.innerWidth) * 2 - 1
      const normalizedY = (event.clientY / window.innerHeight) * 2 - 1
      targetX = -normalizedX * maxMove
      targetY = -normalizedY * maxMove
      schedule()
    }

    applyTransform()
    const pointerFine = window.matchMedia('(hover: hover) and (pointer: fine)').matches
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (pointerFine && !reducedMotion) {
      window.addEventListener('pointermove', pointerMove, { passive: true })
      window.addEventListener('pointerleave', resetToCenter)
      window.addEventListener('blur', resetToCenter)
    }

    return () => {
      window.removeEventListener('pointermove', pointerMove)
      window.removeEventListener('pointerleave', resetToCenter)
      window.removeEventListener('blur', resetToCenter)
      if (frame) window.cancelAnimationFrame(frame)
    }
  }, [activeImage])

  const layerStyle: BackgroundLayerStyle = {
    ...style,
    '--background-blur': `${clamp(settings.blur, 0, 40)}px`,
    '--background-overlay-opacity': clamp(settings.overlayOpacity, 0, 1),
    '--background-transition-duration': `${clamp(settings.transitionDuration, 500, 1_000)}ms`,
  }

  return (
    <div
      aria-hidden="true"
      className={classNames('background-layer', `background-layer--${settings.transition}`, className)}
      style={layerStyle}
    >
      <div className="background-layer__base" />
      <div
        ref={primaryImageRef}
        className={classNames('background-layer__image', activeSlot === 'primary' && 'is-active')}
        style={imageStyle(primaryImage, settings)}
      />
      <div
        ref={secondaryImageRef}
        className={classNames('background-layer__image', activeSlot === 'secondary' && 'is-active')}
        style={imageStyle(secondaryImage, settings)}
      />
      <div className="background-layer__overlay" />
      <div className="background-layer__orb background-layer__orb--primary" />
      <div className="background-layer__orb background-layer__orb--secondary" />
      <div className="background-layer__texture" />
    </div>
  )
}
