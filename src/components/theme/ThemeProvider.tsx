import type { CSSProperties, PropsWithChildren } from 'react'
import { useOptionalBackground } from '../background/BackgroundProvider'
import type { BackgroundThemeOverrides } from '../background/background.types'

type ThemeProviderProps = PropsWithChildren<{
  name?: 'default'
}>

type ThemeStyle = CSSProperties & {
  '--theme-primary'?: string
  '--theme-primary-strong'?: string
  '--theme-secondary'?: string
  '--theme-accent'?: string
  '--theme-background'?: string
  '--theme-glass'?: string
  '--theme-glass-strong'?: string
  '--theme-progress'?: string
}

function createThemeStyle(overrides?: BackgroundThemeOverrides): ThemeStyle | undefined {
  if (!overrides) {
    return undefined
  }

  return {
    ...(overrides.primary ? { '--theme-primary': overrides.primary } : {}),
    ...(overrides.primaryStrong ? { '--theme-primary-strong': overrides.primaryStrong } : {}),
    ...(overrides.secondary ? { '--theme-secondary': overrides.secondary } : {}),
    ...(overrides.accent ? { '--theme-accent': overrides.accent } : {}),
    ...(overrides.background ? { '--theme-background': overrides.background } : {}),
    ...(overrides.glass ? { '--theme-glass': overrides.glass } : {}),
    ...(overrides.glassStrong ? { '--theme-glass-strong': overrides.glassStrong } : {}),
    ...(overrides.progress ? { '--theme-progress': overrides.progress } : {}),
  }
}

/**
 * 主题挂载点。Phase 1 仅使用 CSS 变量，后续可在这里接入持久化主题和动态取色。
 */
export function ThemeProvider({ children, name = 'default' }: ThemeProviderProps) {
  const background = useOptionalBackground()
  const overrides = background?.background.theme?.mode === 'manual'
    ? background.background.theme.overrides
    : undefined

  return <div data-theme={name} style={createThemeStyle(overrides)}>{children}</div>
}
