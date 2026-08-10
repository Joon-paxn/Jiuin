import type { HTMLAttributes } from 'react'
import { classNames } from '../../utils/classNames'

export type DividerProps = HTMLAttributes<HTMLHRElement> & {
  orientation?: 'horizontal' | 'vertical'
}

export function Divider({ className, orientation = 'horizontal', ...props }: DividerProps) {
  return <hr {...props} className={classNames('ui-divider', `ui-divider--${orientation}`, className)} />
}
