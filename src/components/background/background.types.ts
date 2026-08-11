import { backgroundSystemDefaults } from './backgrounds'

export type BackgroundThemeOverrides = Partial<{
  primary: string
  primaryStrong: string
  secondary: string
  accent: string
  background: string
  backgroundMid: string
  backgroundEnd: string
  glass: string
  glassStrong: string
  progress: string
  highlight: string
  overlay: string
  overlayStrong: string
  primaryWash: string
  accentWash: string
  shadow: string
  shadowSoft: string
  focusRing: string
}>

export type BackgroundThemeAdaptation = {
  mode: 'auto' | 'manual' | 'future-auto'
  overrides?: BackgroundThemeOverrides
}

export type BackgroundConfig = {
  id?: string
  /** 与 image 等价，保留以兼容背景配置对象的常见命名。 */
  background?: string
  image?: string
  blur?: number
  backgroundBlur?: number
  opacity?: number
  brightness?: number
  overlayOpacity?: number
  backgroundOverlayOpacity?: number
  transition?: 'crossfade' | 'instant'
  transitionDuration?: number
  theme?: BackgroundThemeAdaptation
}

export type ResolvedBackgroundConfig = {
  id: string
  image?: string
  blur: number
  opacity: number
  brightness: number
  overlayOpacity: number
  transition: 'crossfade' | 'instant'
  transitionDuration: number
  theme?: BackgroundThemeAdaptation
}

export const defaultBackgroundConfig: ResolvedBackgroundConfig = {
  id: 'default',
  blur: backgroundSystemDefaults.backgroundBlur,
  opacity: backgroundSystemDefaults.backgroundImageOpacity,
  brightness: backgroundSystemDefaults.backgroundImageBrightness,
  overlayOpacity: backgroundSystemDefaults.backgroundOverlayOpacity,
  transition: 'crossfade',
  transitionDuration: backgroundSystemDefaults.backgroundTransitionDuration,
}

export function resolveBackgroundConfig(config?: BackgroundConfig): ResolvedBackgroundConfig {
  return {
    ...defaultBackgroundConfig,
    ...config,
    image: config?.image ?? config?.background ?? defaultBackgroundConfig.image,
    blur: config?.blur ?? config?.backgroundBlur ?? defaultBackgroundConfig.blur,
    overlayOpacity: config?.overlayOpacity ?? config?.backgroundOverlayOpacity ?? defaultBackgroundConfig.overlayOpacity,
  }
}
