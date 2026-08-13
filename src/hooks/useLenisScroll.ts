import { useEffect } from 'react'
import Lenis from 'lenis'

/** Native-window Lenis instance: fixed overlays remain viewport anchored and
 * anchors keep their semantic browser URLs while their journey is smoothed. */
export function useLenisScroll(enabled = true) {
  useEffect(() => {
    if (!enabled || window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      return
    }

    const lenis = new Lenis({
      autoRaf: true,
      anchors: { offset: 92 },
      lerp: 0.085,
      smoothWheel: true,
      touchMultiplier: 1,
      wheelMultiplier: 0.9,
    })

    return () => lenis.destroy()
  }, [enabled])
}
