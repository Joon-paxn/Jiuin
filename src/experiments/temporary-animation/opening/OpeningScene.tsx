import { useEffect, useMemo, useRef, type CSSProperties } from 'react'
import type { BackgroundThemeOverrides } from '../../../components/background/background.types'
import type { LoadingBackground, LoadingStage } from './useOpeningSequence'

type OpeningSceneProps = {
  background: LoadingBackground
  isExiting: boolean
  isReady: boolean
  onSkip: () => void
  stage: LoadingStage
}

type OpeningStyle = CSSProperties & {
  '--loading-accent': string
  '--loading-accent-wash': string
  '--loading-background': string
  '--loading-background-end': string
  '--loading-background-mid': string
  '--loading-focus-ring': string
  '--loading-glass': string
  '--loading-glass-strong': string
  '--loading-overlay': string
  '--loading-overlay-strong': string
  '--loading-primary': string
  '--loading-primary-strong': string
  '--loading-primary-wash': string
  '--loading-secondary': string
  '--loading-shadow': string
}

const stageCopy: Record<LoadingStage, string> = {
  space: '正在准备空间',
  light: '正在读取光影',
  interface: '正在整理界面',
  entering: '即将进入霁雪居',
}

function createLoadingThemeStyle(overrides: BackgroundThemeOverrides): OpeningStyle {
  return {
    '--loading-accent': overrides.accent ?? 'var(--theme-accent)',
    '--loading-accent-wash': overrides.accentWash ?? 'var(--theme-accent-wash)',
    '--loading-background': overrides.background ?? 'var(--theme-background)',
    '--loading-background-end': overrides.backgroundEnd ?? 'var(--theme-background-end)',
    '--loading-background-mid': overrides.backgroundMid ?? 'var(--theme-background-mid)',
    '--loading-focus-ring': overrides.focusRing ?? 'var(--theme-focus-ring)',
    '--loading-glass': overrides.glass ?? 'var(--theme-glass)',
    '--loading-glass-strong': overrides.glassStrong ?? 'var(--theme-glass-strong)',
    '--loading-overlay': overrides.overlay ?? 'var(--theme-overlay)',
    '--loading-overlay-strong': overrides.overlayStrong ?? 'var(--theme-overlay-strong)',
    '--loading-primary': overrides.primary ?? 'var(--theme-primary)',
    '--loading-primary-strong': overrides.primaryStrong ?? 'var(--theme-primary-strong)',
    '--loading-primary-wash': overrides.primaryWash ?? 'var(--theme-primary-wash)',
    '--loading-secondary': overrides.secondary ?? 'var(--theme-secondary)',
    '--loading-shadow': overrides.shadow ?? 'var(--theme-shadow)',
  }
}

export function OpeningScene({ background, isExiting, isReady, onSkip, stage }: OpeningSceneProps) {
  const skipButtonRef = useRef<HTMLButtonElement>(null)
  const style = useMemo(() => createLoadingThemeStyle(background.theme), [background.theme])
  const backgroundStyle = useMemo<CSSProperties>(() => background.image
    ? { backgroundImage: `url(${JSON.stringify(background.image)})` }
    : {}, [background.image])

  useEffect(() => {
    skipButtonRef.current?.focus({ preventScroll: true })

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        onSkip()
        return
      }

      if (event.key === 'Tab') {
        event.preventDefault()
        skipButtonRef.current?.focus({ preventScroll: true })
      }
    }

    window.addEventListener('keydown', handleKeyDown)

    return () => {
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [onSkip])

  return (
    <section
      aria-describedby="jiuin-temporary-opening-description"
      aria-labelledby="jiuin-temporary-opening-title"
      aria-modal="true"
      className="jiuin-temporary-opening"
      data-exiting={isExiting ? 'true' : 'false'}
      data-image-ready={background.imageReady ? 'true' : 'false'}
      data-ready={isReady ? 'true' : 'false'}
      data-stage={stage}
      role="dialog"
      style={style}
    >
      <div className="jiuin-temporary-opening__background" aria-hidden="true">
        <div className="jiuin-temporary-opening__image" style={backgroundStyle} />
        <div className="jiuin-temporary-opening__overlay" />
        <div className="jiuin-temporary-opening__glow jiuin-temporary-opening__glow--primary" />
        <div className="jiuin-temporary-opening__glow jiuin-temporary-opening__glow--secondary" />
        <div className="jiuin-temporary-opening__texture" />
      </div>
      <div className="jiuin-temporary-opening__ambient" aria-hidden="true" />
      <div className="jiuin-temporary-opening__content">
        <p className="jiuin-temporary-opening__prelude">JIUIN · 霁雪居</p>
        <span className="jiuin-temporary-opening__line" aria-hidden="true" />
        <div className="jiuin-temporary-opening__brand">
          <span>Jiuin</span>
          <h1 id="jiuin-temporary-opening-title">霁雪居</h1>
          <p id="jiuin-temporary-opening-description">一处正在缓缓展开的个人空间</p>
        </div>
        <div className="jiuin-temporary-opening__status" aria-live="polite" aria-atomic="true">
          <span className="jiuin-temporary-opening__status-mark" aria-hidden="true" />
          <span>{stageCopy[stage]}</span>
        </div>
      </div>
      <button ref={skipButtonRef} className="jiuin-temporary-opening__skip" type="button" onClick={onSkip}>
        立即进入
      </button>
    </section>
  )
}
