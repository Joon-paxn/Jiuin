import type { LoadingAssets, LoadingStage } from './useOpeningSequence'

type OpeningSceneProps = {
  assets: LoadingAssets
  isExiting: boolean
  maskStep: number
  stage: LoadingStage
}

const maskLabels = ['左上', '右上', '左下', '右下']

export function OpeningScene({ assets, isExiting, maskStep, stage }: OpeningSceneProps) {
  return (
    <section
      aria-labelledby="jiuin-opening-title"
      aria-modal="true"
      className="jiuin-temporary-opening"
      data-exiting={isExiting ? 'true' : 'false'}
      data-mask-step={maskStep}
      data-stage={stage}
      role="dialog"
    >
      <div className="jiuin-temporary-opening__scene" aria-hidden="true">
        <div className="jiuin-temporary-opening__web-loading">
          <img src={assets.webLoading} alt="" />
          <div className="jiuin-temporary-opening__mask-grid">
            {maskLabels.map((label, index) => (
              <span key={label} data-revealed={maskStep > index} />
            ))}
          </div>
        </div>
        <img className="jiuin-temporary-opening__artwork" src={assets.artwork} alt="" />
      </div>

      <div className="jiuin-temporary-opening__caption">
        <span className="jiuin-temporary-opening__eyebrow">JIUIN</span>
        <h1 id="jiuin-opening-title">霁雪居</h1>
      </div>
    </section>
  )
}
