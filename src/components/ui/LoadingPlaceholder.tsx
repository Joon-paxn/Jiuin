import type { CSSProperties, HTMLAttributes } from 'react'
import { classNames } from '../../utils/classNames'

type LoadingPlaceholderShape = 'line' | 'block' | 'circle'

export type LoadingPlaceholderProps = HTMLAttributes<HTMLDivElement> & {
  shape?: LoadingPlaceholderShape
  width?: string
  height?: string
}

type PlaceholderStyle = CSSProperties & {
  '--placeholder-width'?: string
  '--placeholder-height'?: string
}

/** 静态占位骨架；动态加载编排留待未来 Loading System。 */
export function LoadingPlaceholder({
  className,
  shape = 'line',
  width,
  height,
  style,
  ...props
}: LoadingPlaceholderProps) {
  const placeholderStyle: PlaceholderStyle = {
    ...style,
    ...(width ? { '--placeholder-width': width } : {}),
    ...(height ? { '--placeholder-height': height } : {}),
  }

  return (
    <div
      {...props}
      aria-busy="true"
      className={classNames('loading-placeholder', `loading-placeholder--${shape}`, className)}
      style={placeholderStyle}
    />
  )
}
