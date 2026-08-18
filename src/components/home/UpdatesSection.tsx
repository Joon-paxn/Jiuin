import { useEffect, useLayoutEffect, useRef, useState, type PointerEvent } from 'react'
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
  const timelineAnimationRef = useRef<{ stop: () => void } | null>(null)
  const dragStateRef = useRef<{ pointerId: number; startX: number; startTimelineX: number } | null>(null)
  const [isDragging, setIsDragging] = useState(false)
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
    timelineAnimationRef.current?.stop()
    timelineAnimationRef.current = animate(timelineX, targetXRef.current, {
      duration: reduceMotion ? 0.18 : animationDuration,
      ease: timelineEase,
    })

    return () => timelineAnimationRef.current?.stop()
  }, [animationDuration, isInView, latestShift, measureReady, reduceMotion, timelineX])

  const clampTimelineX = (value: number) => Math.min(0, Math.max(-latestShift, value))

  const handlePointerDown = (event: PointerEvent<HTMLDivElement>) => {
    if (event.pointerType !== 'mouse' || reduceMotion || !measureReady || latestShift <= 0) {
      return
    }

    event.preventDefault()
    event.currentTarget.setPointerCapture(event.pointerId)
    timelineAnimationRef.current?.stop()
    dragStateRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startTimelineX: timelineX.get(),
    }
    targetXRef.current = timelineX.get()
    setIsDragging(true)
  }

  const handlePointerMove = (event: PointerEvent<HTMLDivElement>) => {
    const dragState = dragStateRef.current
    if (!dragState || dragState.pointerId !== event.pointerId) {
      return
    }

    event.preventDefault()
    const nextTarget = clampTimelineX(dragState.startTimelineX + event.clientX - dragState.startX)
    targetXRef.current = nextTarget
    timelineX.set(nextTarget)
  }

  const finishPointerDrag = (event: PointerEvent<HTMLDivElement>) => {
    const dragState = dragStateRef.current
    if (!dragState || dragState.pointerId !== event.pointerId) {
      return
    }

    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    dragStateRef.current = null
    setIsDragging(false)
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

      <div
        ref={viewportRef}
        className="updates-timeline"
        aria-label="Jiuin 开发历史时间线"
        data-dragging={isDragging || undefined}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={finishPointerDrag}
        onPointerCancel={finishPointerDrag}
      >
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
