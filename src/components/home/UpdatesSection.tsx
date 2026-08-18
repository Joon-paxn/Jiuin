import { useEffect, useLayoutEffect, useRef, useState, type WheelEvent } from 'react'
import { animate, motion, useInView, useMotionValue, useReducedMotion, useTransform } from 'framer-motion'
import { updatesData } from './updatesData'

const timelineEase = [0.23, 1, 0.32, 1] as const

export function UpdatesSection() {
  const viewportRef = useRef<HTMLDivElement>(null)
  const trackRef = useRef<HTMLDivElement>(null)
  const isInView = useInView(viewportRef, { once: true, amount: 0.42 })
  const reduceMotion = useReducedMotion()
  const [latestShift, setLatestShift] = useState(0)
  const [measureReady, setMeasureReady] = useState(false)
  const timelineX = useMotionValue(0)
  const timelineTransform = useTransform(timelineX, (value) => `translate3d(${value}px, 0, 0)`)
  const targetXRef = useRef(0)
  const wheelAnimationRef = useRef<{ stop: () => void } | null>(null)
  const latestIndex = updatesData.length - 1
  const animationDuration = Math.min(3, Math.max(1.8, 1.2 + updatesData.length * 0.3))

  useLayoutEffect(() => {
    const viewport = viewportRef.current
    const track = trackRef.current
    if (!viewport || !track || !isInView || reduceMotion) {
      return
    }

    const alignLatest = () => {
      setLatestShift(Math.max(0, track.scrollWidth - viewport.clientWidth))
      setMeasureReady(true)
    }

    alignLatest()
    const resizeObserver = new ResizeObserver(alignLatest)
    resizeObserver.observe(viewport)
    resizeObserver.observe(track)
    return () => resizeObserver.disconnect()
  }, [isInView, reduceMotion])

  useEffect(() => {
    if (!isInView || !measureReady) {
      return
    }

    targetXRef.current = reduceMotion ? 0 : -latestShift
    wheelAnimationRef.current?.stop()
    wheelAnimationRef.current = animate(timelineX, targetXRef.current, {
      duration: reduceMotion ? 0.18 : animationDuration,
      ease: timelineEase,
    })

    return () => wheelAnimationRef.current?.stop()
  }, [animationDuration, isInView, latestShift, measureReady, reduceMotion, timelineX])

  const handleTimelineWheel = (event: WheelEvent<HTMLDivElement>) => {
    if (reduceMotion || !measureReady || latestShift <= 0 || window.matchMedia('(pointer: coarse)').matches) {
      return
    }

    const delta = Math.abs(event.deltaX) > Math.abs(event.deltaY) ? event.deltaX : event.deltaY
    if (!delta) {
      return
    }

    // A desktop pointer over the archive owns the wheel gesture. The page
    // resumes native vertical scrolling as soon as the pointer leaves it.
    event.preventDefault()
    const nextTarget = Math.min(0, Math.max(-latestShift, targetXRef.current - delta))
    if (nextTarget === targetXRef.current) {
      return
    }

    targetXRef.current = nextTarget
    wheelAnimationRef.current?.stop()
    wheelAnimationRef.current = animate(timelineX, nextTarget, {
      duration: 0.36,
      ease: timelineEase,
    })
  }

  return (
    <section id="content-preview" className="home-section updates-section" aria-labelledby="content-preview-title">
      <motion.header
        className="updates-section__heading"
        initial={reduceMotion ? false : { opacity: 0, transform: 'translate3d(0, 1rem, 0)' }}
        whileInView={{ opacity: 1, transform: 'translate3d(0, 0, 0)' }}
        transition={{ duration: 0.5, ease: timelineEase }}
        viewport={{ once: true, amount: 0.42 }}
      >
        <p className="updates-section__eyebrow"><span>03</span> DEVELOPMENT HISTORY</p>
        <h2 id="content-preview-title" className="updates-section__title">
          Website Updates
          <span>网站更新数据</span>
        </h2>
        <p className="updates-section__description">记录霁雪居从过去到现在的每一次重要变化。</p>
      </motion.header>

      <div ref={viewportRef} className="updates-timeline" aria-label="Jiuin 开发历史时间线" onWheel={handleTimelineWheel}>
        <motion.div
          ref={trackRef}
          className="updates-timeline__track"
          style={{ transform: timelineTransform }}
        >
          <div className="updates-timeline__axis" aria-hidden="true" />
          {updatesData.map((update, index) => (
            <article
              key={update.id}
              className={`updates-timeline__node updates-timeline__node--${update.period}${index === latestIndex ? ' is-latest' : ''}`}
            >
              <span className="updates-timeline__dot" aria-hidden="true" />
              <span className="updates-timeline__connector" aria-hidden="true" />
              <div className="updates-timeline__card">
                <div className="updates-timeline__meta">
                  <time dateTime={update.timestamp}>{update.dateLabel} · {update.timeLabel}</time>
                  {index === latestIndex && <span className="updates-timeline__latest">LATEST</span>}
                </div>
                <h3>{update.title}</h3>
                <p>{update.description}</p>
              </div>
            </article>
          ))}
        </motion.div>
      </div>

      <div className="updates-section__footer">
        <span>OLDEST</span>
        <i aria-hidden="true" />
        <span>LATEST / {updatesData.length.toString().padStart(2, '0')}</span>
      </div>
    </section>
  )
}
