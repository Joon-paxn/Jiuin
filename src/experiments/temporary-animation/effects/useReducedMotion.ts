import { useEffect, useState } from 'react'

function getReducedMotionPreference() {
  return typeof window !== 'undefined'
    && window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

export function useReducedMotion(enabled: boolean) {
  const [reducedMotion, setReducedMotion] = useState(getReducedMotionPreference)

  useEffect(() => {
    if (!enabled) {
      return
    }

    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    const syncPreference = () => setReducedMotion(mediaQuery.matches)

    syncPreference()
    mediaQuery.addEventListener('change', syncPreference)

    return () => {
      mediaQuery.removeEventListener('change', syncPreference)
    }
  }, [enabled])

  return reducedMotion
}
