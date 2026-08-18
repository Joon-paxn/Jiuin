import { motion, useReducedMotion } from 'framer-motion'
import { Button } from '../ui/Button'

const enterEase = [0.23, 1, 0.32, 1] as const

export function BlogPreviewSection() {
  const reduceMotion = useReducedMotion()
  const initial = reduceMotion
    ? false
    : { opacity: 0, transform: 'translate3d(0, 1.5rem, 0)' }
  const visible = { opacity: 1, transform: 'translate3d(0, 0, 0)' }
  const backPageVisible = reduceMotion
    ? { transform: 'none' }
    : { transform: 'translate3d(0, 0, 0) rotate(5deg)' }
  const middlePageVisible = reduceMotion
    ? { transform: 'none' }
    : { transform: 'translate3d(0, 0, 0) rotate(-2deg)' }

  return (
    <section id="blog" className="home-section blog-preview-section" aria-labelledby="blog-preview-title">
      <div className="blog-preview__copy">
        <motion.p
          className="blog-preview__eyebrow"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.55, ease: enterEase }}
          viewport={{ once: true, amount: 0.42 }}
        >
          <span>04</span>
          THE SECOND FLOOR
        </motion.p>

        <motion.h2
          id="blog-preview-title"
          className="blog-preview__title"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.7, delay: 0.08, ease: enterEase }}
          viewport={{ once: true, amount: 0.42 }}
        >
          <span>Jiuin</span>
          BLOG
        </motion.h2>

        <motion.p
          className="blog-preview__lead"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.6, delay: 0.16, ease: enterEase }}
          viewport={{ once: true, amount: 0.42 }}
        >
          开发日志 / 文章空间
        </motion.p>

        <motion.p
          className="blog-preview__body"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.6, delay: 0.24, ease: enterEase }}
          viewport={{ once: true, amount: 0.42 }}
        >
          这里将记录开发过程、技术文章、想法，以及一些值得长期积累的内容。
        </motion.p>

        <motion.div
          className="blog-preview__entry"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.6, delay: 0.32, ease: enterEase }}
          viewport={{ once: true, amount: 0.42 }}
        >
          <Button variant="glass" size="lg" disabled aria-describedby="blog-preview-status">
            <span>进入 Blog</span>
            <span aria-hidden="true">↗</span>
          </Button>
          <span id="blog-preview-status" className="blog-preview__status">COMING LATER</span>
        </motion.div>
      </div>

      <motion.div
        className="blog-preview__visual"
        aria-hidden="true"
        initial={reduceMotion ? false : { opacity: 0, transform: 'translate3d(1.5rem, 0.75rem, 0)' }}
        whileInView={{ opacity: 1, transform: 'translate3d(0, 0, 0)' }}
        transition={{ duration: 0.85, delay: 0.2, ease: enterEase }}
        viewport={{ once: true, amount: 0.35 }}
      >
        <motion.div
          className="blog-preview__page blog-preview__page--back"
          initial={reduceMotion ? false : { transform: 'translate3d(1rem, 0.75rem, 0) rotate(7deg)' }}
          whileInView={backPageVisible}
          transition={{ duration: 0.9, delay: 0.25, ease: enterEase }}
          viewport={{ once: true, amount: 0.35 }}
        />
        <motion.div
          className="blog-preview__page blog-preview__page--middle"
          initial={reduceMotion ? false : { transform: 'translate3d(0.6rem, 0.45rem, 0) rotate(-4deg)' }}
          whileInView={middlePageVisible}
          transition={{ duration: 0.85, delay: 0.32, ease: enterEase }}
          viewport={{ once: true, amount: 0.35 }}
        />
        <div className="blog-preview__page blog-preview__page--front">
          <span className="blog-preview__page-index">J / 04</span>
          <span className="blog-preview__page-word">BLOG</span>
          <span className="blog-preview__page-caption">A ROOM FOR WORDS</span>
          <i className="blog-preview__page-line blog-preview__page-line--vertical" />
          <i className="blog-preview__page-line blog-preview__page-line--horizontal" />
          <span className="blog-preview__page-corner blog-preview__page-corner--top" />
          <span className="blog-preview__page-corner blog-preview__page-corner--bottom" />
        </div>
      </motion.div>
    </section>
  )
}
