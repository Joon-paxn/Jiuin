import type { CSSProperties } from 'react'
import { classNames } from '../../utils/classNames'

export type BackgroundConfig = {
  image?: string
  blur?: number
  opacity?: number
  brightness?: number
}

export type BackgroundLayerProps = {
  config?: BackgroundConfig
  className?: string
  style?: CSSProperties
}

type BackgroundImageStyle = CSSProperties & {
  '--background-image-blur': string
  '--background-image-brightness': number
  '--background-image-opacity': number
}

export const defaultBackgroundConfig: Required<Omit<BackgroundConfig, 'image'>> = {
  blur: 0,
  opacity: 1,
  brightness: 1,
}

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value))
}

/**
 * 全站背景层。图片、模糊、透明度与亮度均由配置传入，动态主题取色暂不在本阶段实现。
 */
export function BackgroundLayer({ config, className, style }: BackgroundLayerProps) {
  const settings = { ...defaultBackgroundConfig, ...config }
  const imageStyle: BackgroundImageStyle = {
    '--background-image-blur': `${clamp(settings.blur, 0, 40)}px`,
    '--background-image-brightness': clamp(settings.brightness, 0.2, 1.5),
    '--background-image-opacity': clamp(settings.opacity, 0, 1),
    ...(settings.image ? { backgroundImage: `url(${JSON.stringify(settings.image)})` } : {}),
  }

  return (
    <div
      aria-hidden="true"
      className={classNames('background-layer', className)}
      style={style}
    >
      <div className="background-layer__base" />
      <div className="background-layer__image" style={imageStyle} />
      <div className="background-layer__blur" />
      <div className="background-layer__overlay" />
      <div className="background-layer__orb background-layer__orb--primary" />
      <div className="background-layer__orb background-layer__orb--secondary" />
      <div className="background-layer__texture" />
    </div>
  )
}
