import type { AnchorHTMLAttributes, ButtonHTMLAttributes } from 'react'
import { classNames } from '../../utils/classNames'

type ButtonVariant = 'primary' | 'secondary' | 'glass' | 'ghost'
type ButtonSize = 'sm' | 'md' | 'lg'

type ButtonBaseProps = {
  variant?: ButtonVariant
  size?: ButtonSize
}

type ButtonElementProps = ButtonBaseProps & ButtonHTMLAttributes<HTMLButtonElement> & {
  href?: never
}

type ButtonLinkProps = ButtonBaseProps & AnchorHTMLAttributes<HTMLAnchorElement> & {
  href: string
}

export type ButtonProps = ButtonElementProps | ButtonLinkProps

function isLinkButton(props: ButtonProps): props is ButtonLinkProps {
  return 'href' in props && typeof props.href === 'string'
}

export function Button(props: ButtonProps) {
  const buttonClassName = classNames(
    'ui-button',
    'glass-button',
    `ui-button--${props.variant ?? 'primary'}`,
    `ui-button--${props.size ?? 'md'}`,
    props.className,
  )

  if (isLinkButton(props)) {
    const { className: _className, variant: _variant, size: _size, href, ...linkProps } = props

    return <a {...linkProps} className={buttonClassName} href={href} />
  }

  const { className: _className, variant: _variant, size: _size, type, ...buttonProps } = props

  return (
    <button
      {...buttonProps}
      type={type ?? 'button'}
      className={buttonClassName}
    />
  )
}
