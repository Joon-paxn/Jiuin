import type { CSSProperties } from 'react'

type BackgroundLayerProps = {
  className?: string
  style?: CSSProperties
}

/** 全站背景容器；后续可接入动态背景和主题色计算。 */
export function BackgroundLayer({ className, style }: BackgroundLayerProps) {
  return (
    <div
      aria-hidden="true"
      className={['background-layer', className].filter(Boolean).join(' ')}
      style={style}
    >
      <div className="background-layer__orb background-layer__orb--primary" />
      <div className="background-layer__orb background-layer__orb--secondary" />
      <div className="background-layer__texture" />
    </div>
  )
}
