import type { HTMLAttributes } from 'react'
import { classNames } from '../../utils/classNames'

type ContainerSize = 'narrow' | 'content' | 'wide'

export type ContainerProps = HTMLAttributes<HTMLDivElement> & {
  size?: ContainerSize
}

export function Container({ className, size = 'content', ...props }: ContainerProps) {
  return <div {...props} className={classNames('ui-container', `ui-container--${size}`, className)} />
}
