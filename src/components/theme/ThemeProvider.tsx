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
  '--theme-background-mid'?: string
  '--theme-background-end'?: string
  '--theme-glass'?: string
  '--theme-glass-strong'?: string
  '--theme-progress'?: string
  '--theme-highlight'?: string
  '--theme-overlay'?: string
  '--theme-overlay-strong'?: string
  '--theme-primary-wash'?: string
  '--theme-accent-wash'?: string
  '--theme-shadow'?: string
  '--theme-shadow-soft'?: string
  '--theme-focus-ring'?: string
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
    ...(overrides.backgroundMid ? { '--theme-background-mid': overrides.backgroundMid } : {}),
    ...(overrides.backgroundEnd ? { '--theme-background-end': overrides.backgroundEnd } : {}),
    ...(overrides.glass ? { '--theme-glass': overrides.glass } : {}),
    ...(overrides.glassStrong ? { '--theme-glass-strong': overrides.glassStrong } : {}),
    ...(overrides.progress ? { '--theme-progress': overrides.progress } : {}),
    ...(overrides.highlight ? { '--theme-highlight': overrides.highlight } : {}),
    ...(overrides.overlay ? { '--theme-overlay': overrides.overlay } : {}),
    ...(overrides.overlayStrong ? { '--theme-overlay-strong': overrides.overlayStrong } : {}),
    ...(overrides.primaryWash ? { '--theme-primary-wash': overrides.primaryWash } : {}),
    ...(overrides.accentWash ? { '--theme-accent-wash': overrides.accentWash } : {}),
    ...(overrides.shadow ? { '--theme-shadow': overrides.shadow } : {}),
    ...(overrides.shadowSoft ? { '--theme-shadow-soft': overrides.shadowSoft } : {}),
    ...(overrides.focusRing ? { '--theme-focus-ring': overrides.focusRing } : {}),
  }
}

/**
 * 主题挂载点。BackgroundSystem 更新背景主题变量后，所有消费 --theme-* 的 UI 会同步刷新。
 */
export function ThemeProvider({ children, name = 'default' }: ThemeProviderProps) {
  const background = useOptionalBackground()
  const overrides = background?.background.theme?.overrides

  return <div data-theme={name} style={createThemeStyle(overrides)}>{children}</div>
}
