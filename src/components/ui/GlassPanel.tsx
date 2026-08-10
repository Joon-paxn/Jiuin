import type { HTMLAttributes } from 'react'
import { classNames } from '../../utils/classNames'

type GlassPanelTone = 'default' | 'soft' | 'strong'

export type GlassPanelProps = HTMLAttributes<HTMLDivElement> & {
  tone?: GlassPanelTone
}

export function GlassPanel({ className, tone = 'default', ...props }: GlassPanelProps) {
  return <div {...props} className={classNames('glass-panel', `glass-panel--${tone}`, className)} />
}
