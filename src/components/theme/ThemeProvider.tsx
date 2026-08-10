import type { PropsWithChildren } from 'react'

type ThemeProviderProps = PropsWithChildren<{
  name?: 'default'
}>

/**
 * 主题挂载点。Phase 1 仅使用 CSS 变量，后续可在这里接入持久化主题和动态取色。
 */
export function ThemeProvider({ children, name = 'default' }: ThemeProviderProps) {
  return <div data-theme={name}>{children}</div>
}
