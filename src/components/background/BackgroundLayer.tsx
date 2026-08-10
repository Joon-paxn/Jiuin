import { useEffect, useState, type CSSProperties } from 'react'
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
 * 全站背景视觉层：支持双图层交叉淡入、图片参数与覆盖层配置。
 * 自动颜色提取暂不实现，主题同步由 BackgroundProvider 的配置协议预留。
 */
export function BackgroundLayer({ config, className, style }: BackgroundLayerProps) {
  const managedBackground = useOptionalBackground()
  const settings = config
    ? resolveBackgroundConfig(config)
    : managedBackground?.background ?? resolveBackgroundConfig()
  const [primaryImage, setPrimaryImage] = useState(settings.image)
  const [secondaryImage, setSecondaryImage] = useState<string | undefined>()
  const [activeSlot, setActiveSlot] = useState<BackgroundSlot>('primary')
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

  const layerStyle: BackgroundLayerStyle = {
    ...style,
    '--background-blur': `${clamp(settings.blur, 0, 40)}px`,
    '--background-overlay-opacity': clamp(settings.overlayOpacity, 0, 1),
  }

  return (
    <div
      aria-hidden="true"
      className={classNames('background-layer', `background-layer--${settings.transition}`, className)}
      style={layerStyle}
    >
      <div className="background-layer__base" />
      <div
        className={classNames('background-layer__image', activeSlot === 'primary' && 'is-active')}
        style={imageStyle(primaryImage, settings)}
      />
      <div
        className={classNames('background-layer__image', activeSlot === 'secondary' && 'is-active')}
        style={imageStyle(secondaryImage, settings)}
      />
      <div className="background-layer__blur" />
      <div className="background-layer__overlay" />
      <div className="background-layer__orb background-layer__orb--primary" />
      <div className="background-layer__orb background-layer__orb--secondary" />
      <div className="background-layer__texture" />
    </div>
  )
}
