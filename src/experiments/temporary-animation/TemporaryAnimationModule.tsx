import type { PropsWithChildren } from 'react'
import { temporaryAnimationEnabled } from './config'
import { ParallaxController } from './effects/ParallaxController'
import { useReducedMotion } from './effects/useReducedMotion'
import { OpeningScene } from './opening/OpeningScene'
import { useOpeningSequence } from './opening/useOpeningSequence'
import { ScrollRevealController } from './scroll/ScrollRevealController'
import { PageTransition } from './transition/PageTransition'
import './temporary-animation.css'

export function TemporaryAnimationModule({ children }: PropsWithChildren) {
  const reducedMotion = useReducedMotion(temporaryAnimationEnabled)
  const opening = useOpeningSequence({
    enabled: temporaryAnimationEnabled,
    reducedMotion,
  })

  if (!temporaryAnimationEnabled) {
    return <>{children}</>
  }

  return (
    <div className="jiuin-temporary-animation" data-page-ready={opening.isPageReady ? 'true' : 'false'}>
      {opening.isPageReady && (
        <PageTransition>
          {children}
          <ScrollRevealController reducedMotion={reducedMotion} />
          <ParallaxController reducedMotion={reducedMotion} />
        </PageTransition>
      )}
      {opening.isOpening && (
        <OpeningScene
          isExiting={opening.isPageReady}
          isReady={opening.isReady}
          onSkip={opening.finishOpening}
        />
      )}
    </div>
  )
}
