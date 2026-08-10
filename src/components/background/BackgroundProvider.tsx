import { createContext, useCallback, useContext, useMemo, useState, type PropsWithChildren } from 'react'
import {
  defaultBackgroundConfig,
  resolveBackgroundConfig,
  type BackgroundConfig,
  type ResolvedBackgroundConfig,
} from './background.types'

type BackgroundContextValue = {
  background: ResolvedBackgroundConfig
  setBackground: (config: BackgroundConfig) => void
  updateBackground: (config: Partial<BackgroundConfig>) => void
  resetBackground: () => void
}

const BackgroundContext = createContext<BackgroundContextValue | null>(null)

type BackgroundProviderProps = PropsWithChildren<{
  initialBackground?: BackgroundConfig
}>

/**
 * 管理全站背景的可切换配置。当前不进行自动取色，但会保留主题同步协议供后续实现。
 */
export function BackgroundProvider({ children, initialBackground }: BackgroundProviderProps) {
  const [background, setResolvedBackground] = useState(() => resolveBackgroundConfig(initialBackground))

  const setBackground = useCallback((config: BackgroundConfig) => {
    setResolvedBackground(resolveBackgroundConfig(config))
  }, [])

  const updateBackground = useCallback((config: Partial<BackgroundConfig>) => {
    setResolvedBackground((current) => resolveBackgroundConfig({
      ...current,
      ...config,
      image: config.image ?? config.background ?? current.image,
    }))
  }, [])

  const resetBackground = useCallback(() => {
    setResolvedBackground(defaultBackgroundConfig)
  }, [])

  const value = useMemo(() => ({
    background,
    setBackground,
    updateBackground,
    resetBackground,
  }), [background, resetBackground, setBackground, updateBackground])

  return <BackgroundContext.Provider value={value}>{children}</BackgroundContext.Provider>
}

export function useBackground() {
  const context = useContext(BackgroundContext)

  if (!context) {
    throw new Error('useBackground must be used within BackgroundProvider')
  }

  return context
}

export function useOptionalBackground() {
  return useContext(BackgroundContext)
}
