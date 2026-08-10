import { useEffect } from 'react'

const MAX_SCROLL_TRAVEL = 960

const parallaxLayers = [
  { selector: '.background-layer__orb--primary', multiplier: -0.009 },
  { selector: '.background-layer__orb--secondary', multiplier: 0.011 },
  { selector: '.hero-visual__glow', multiplier: -0.006 },
  { selector: '.hero-visual__orbit--outer', multiplier: 0.007 },
] as const

type ParallaxControllerProps = {
  reducedMotion: boolean
}

/** A single rAF-throttled scroll path for decorative layers only. */
export function ParallaxController({ reducedMotion }: ParallaxControllerProps) {
  useEffect(() => {
    if (reducedMotion) {
      return
    }

    const layers = parallaxLayers.flatMap(({ selector, multiplier }) => {
      const element = document.querySelector<HTMLElement>(selector)
      return element ? [{ element, multiplier }] : []
    })
    const hero = document.querySelector<HTMLElement>('.home-hero')

    if (layers.length === 0) {
      return
    }

    let frameId: number | undefined
    let isHeroVisible = true

    const render = () => {
      frameId = undefined

      if (!isHeroVisible) {
        return
      }

      const scrollPosition = Math.min(window.scrollY, MAX_SCROLL_TRAVEL)
      layers.forEach(({ element, multiplier }) => {
        const offset = Math.round(scrollPosition * multiplier * 10) / 10
        element.style.transform = `translate3d(0, ${offset}px, 0)`
      })
    }

    const scheduleRender = () => {
      if (frameId === undefined) {
        frameId = window.requestAnimationFrame(render)
      }
    }

    const handleScroll = () => {
      if (isHeroVisible) {
        scheduleRender()
      }
    }

    const visibilityObserver = hero && 'IntersectionObserver' in window
      ? new IntersectionObserver(([entry]) => {
        isHeroVisible = entry.isIntersecting

        if (isHeroVisible) {
          scheduleRender()
        } else if (frameId !== undefined) {
          window.cancelAnimationFrame(frameId)
          frameId = undefined
        }
      }, { rootMargin: '15% 0px' })
      : undefined

    visibilityObserver?.observe(hero!)
    window.addEventListener('scroll', handleScroll, { passive: true })
    scheduleRender()

    return () => {
      window.removeEventListener('scroll', handleScroll)
      visibilityObserver?.disconnect()

      if (frameId !== undefined) {
        window.cancelAnimationFrame(frameId)
      }

      layers.forEach(({ element }) => element.style.removeProperty('transform'))
    }
  }, [reducedMotion])

  return null
}
