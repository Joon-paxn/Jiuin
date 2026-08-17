import { useEffect, useRef, useState } from 'react'

/** Reveals a section once it owns the viewport, without animating off-screen content. */
export function useSectionReveal<T extends HTMLElement>() {
  const ref = useRef<T>(null)
  const [isVisible, setIsVisible] = useState(false)

  useEffect(() => {
    const element = ref.current
    if (!element) return

    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches || !('IntersectionObserver' in window)) {
      setIsVisible(true)
      return
    }

    const observer = new IntersectionObserver(([entry]) => {
      if (!entry?.isIntersecting) return
      setIsVisible(true)
      observer.disconnect()
    }, { threshold: 0.24, rootMargin: '-8% 0px -8% 0px' })

    observer.observe(element)
    return () => observer.disconnect()
  }, [])

  return { ref, isVisible }
}
