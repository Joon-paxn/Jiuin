import type { HTMLAttributes } from 'react'
import { classNames } from '../../utils/classNames'

type CardVariant = 'default' | 'soft' | 'interactive'

export type CardProps = HTMLAttributes<HTMLDivElement> & {
  variant?: CardVariant
}

export function Card({ className, variant = 'default', ...props }: CardProps) {
  return <div {...props} className={classNames('ui-card', 'glass-card', `ui-card--${variant}`, className)} />
}
