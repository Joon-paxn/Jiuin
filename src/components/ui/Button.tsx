import type { ButtonHTMLAttributes } from 'react'
import { classNames } from '../../utils/classNames'

type ButtonVariant = 'primary' | 'secondary' | 'glass' | 'ghost'
type ButtonSize = 'sm' | 'md' | 'lg'

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant
  size?: ButtonSize
}

export function Button({
  className,
  variant = 'primary',
  size = 'md',
  type,
  ...props
}: ButtonProps) {
  return (
    <button
      {...props}
      type={type ?? 'button'}
      className={classNames('ui-button', 'glass-button', `ui-button--${variant}`, `ui-button--${size}`, className)}
    />
  )
}
