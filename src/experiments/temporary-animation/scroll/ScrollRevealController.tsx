import { useEffect } from 'react'

const revealSelector = [
  '.about-section .section-heading',
  '.introduction-card',
  '.content-preview .section-heading',
  '.content-preview__grid > .preview-card',
].join(', ')

function reveal(target: HTMLElement) {
  target.dataset.jiuinReveal = 'visible'
}

function reset(target: HTMLElement) {
  delete target.dataset.jiuinReveal
  target.style.removeProperty('--jiuin-reveal-delay')
}

type ScrollRevealControllerProps = {
  reducedMotion: boolean
}

/** Uses one observer and leaves every target revealed after its first intersection. */
export function ScrollRevealController({ reducedMotion }: ScrollRevealControllerProps) {
  useEffect(() => {
    const targets = Array.from(document.querySelectorAll<HTMLElement>(revealSelector))

    if (targets.length === 0) {
      return
    }

    if (reducedMotion || !('IntersectionObserver' in window)) {
      targets.forEach(reveal)
      return () => {
        targets.forEach(reset)
      }
    }

    targets.forEach((target, index) => {
      target.dataset.jiuinReveal = 'pending'
      target.style.setProperty('--jiuin-reveal-delay', `${Math.min(index, 3) * 80}ms`)
    })

    const observer = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (!entry.isIntersecting) {
          return
        }

        reveal(entry.target as HTMLElement)
        observer.unobserve(entry.target)
      })
    }, {
      rootMargin: '0px 0px -8% 0px',
      threshold: 0.1,
    })

    targets.forEach((target) => observer.observe(target))

    return () => {
      observer.disconnect()
      targets.forEach(reset)
    }
  }, [reducedMotion])

  return null
}
