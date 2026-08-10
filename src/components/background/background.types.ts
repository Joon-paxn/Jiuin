export type BackgroundThemeOverrides = Partial<{
  primary: string
  primaryStrong: string
  secondary: string
  accent: string
  background: string
  glass: string
  glassStrong: string
  progress: string
}>

/** 自动取色仅保留协议；当前仅应用显式提供的手动覆盖值。 */
export type BackgroundThemeAdaptation = {
  mode: 'manual' | 'future-auto'
  overrides?: BackgroundThemeOverrides
}

export type BackgroundConfig = {
  id?: string
  /** 与 image 等价，保留以兼容背景配置对象的常见命名。 */
  background?: string
  image?: string
  blur?: number
  opacity?: number
  brightness?: number
  overlayOpacity?: number
  transition?: 'crossfade' | 'instant'
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
  theme?: BackgroundThemeAdaptation
}

export const defaultBackgroundConfig: ResolvedBackgroundConfig = {
  id: 'default',
  blur: 0,
  opacity: 1,
  brightness: 1,
  overlayOpacity: 1,
  transition: 'crossfade',
}

export function resolveBackgroundConfig(config?: BackgroundConfig): ResolvedBackgroundConfig {
  return {
    ...defaultBackgroundConfig,
    ...config,
    image: config?.image ?? config?.background ?? defaultBackgroundConfig.image,
  }
}
