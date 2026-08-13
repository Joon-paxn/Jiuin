import type { LoadingAssets, LoadingStage } from './useOpeningSequence'

type OpeningSceneProps = {
  assets: LoadingAssets
  isExiting: boolean
  maskStep: number
  stage: LoadingStage
  error?: string | null
  onRetry?: () => void
  backgroundReady?: boolean
}

function maskClipPath(maskStep: number) {
  switch (maskStep) {
    case 1:
      return 'polygon(0 0, 50% 0, 50% 50%, 0 50%)'
    case 2:
      return 'polygon(0 0, 100% 0, 100% 50%, 0 50%)'
    case 3:
      return 'polygon(0 0, 100% 0, 100% 50%, 50% 50%, 50% 100%, 0 100%)'
    case 4:
      return 'inset(0)'
    default:
      return 'polygon(0 0, 0 0, 0 0, 0 0)'
  }
}

export function OpeningScene({ assets, isExiting, maskStep, stage, error, onRetry, backgroundReady }: OpeningSceneProps) {
  return (
    <section
      aria-label="Loading"
      aria-modal="true"
      className="jiuin-temporary-opening"
      data-background-ready={backgroundReady ? 'true' : 'false'}
      data-exiting={isExiting ? 'true' : 'false'}
      data-mask-step={maskStep}
      data-stage={stage}
      role="dialog"
    >
      <div className="jiuin-temporary-opening__scene" aria-hidden="true">
        <img
          className="jiuin-temporary-opening__web-loading"
          src={assets.webLoading}
          alt=""
          style={{ clipPath: maskClipPath(maskStep) }}
        />
        {stage === 'artwork' && <img className="jiuin-temporary-opening__artwork" src={assets.artwork} alt="" />}
      </div>
      {error && (
        <div className="jiuin-temporary-opening__error" role="alert">
          <span>{error}</span>
          <button type="button" onClick={onRetry}>重试</button>
        </div>
      )}
    </section>
  )
}
