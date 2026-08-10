import { useEffect, useRef } from 'react'

type OpeningSceneProps = {
  isExiting: boolean
  isReady: boolean
  onSkip: () => void
}

export function OpeningScene({ isExiting, isReady, onSkip }: OpeningSceneProps) {
  const skipButtonRef = useRef<HTMLButtonElement>(null)

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
      data-ready={isReady ? 'true' : 'false'}
      role="dialog"
    >
      <div className="jiuin-temporary-opening__ambient" aria-hidden="true" />
      <div className="jiuin-temporary-opening__content">
        <p className="jiuin-temporary-opening__prelude">01 — OPENING THE QUIET</p>
        <span className="jiuin-temporary-opening__line" aria-hidden="true" />
        <div className="jiuin-temporary-opening__brand">
          <span>Jiuin</span>
          <h1 id="jiuin-temporary-opening-title">霁雪居</h1>
          <p id="jiuin-temporary-opening-description">一处正在缓缓展开的个人空间</p>
        </div>
      </div>
      <button ref={skipButtonRef} className="jiuin-temporary-opening__skip" type="button" onClick={onSkip}>
        跳过片头
      </button>
    </section>
  )
}
