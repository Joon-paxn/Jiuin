import { motion, useReducedMotion } from 'framer-motion'
import { Button } from '../ui/Button'

const enterEase = [0.23, 1, 0.32, 1] as const

export function ImageSpaceSection() {
  const reduceMotion = useReducedMotion()
  const initial = reduceMotion
    ? false
    : { opacity: 0, transform: 'translate3d(0, 1.5rem, 0)' }
  const visible = { opacity: 1, transform: 'translate3d(0, 0, 0)' }
  const frameInitial = (transform: string) => reduceMotion ? false : { opacity: 0, transform }
  const frameVisible = (transform: string) => reduceMotion ? {} : { opacity: 1, transform }

  return (
    <section id="image-space" className="home-section image-space-section" aria-labelledby="image-space-title">
      <div className="image-space__copy">
        <motion.p
          className="image-space__eyebrow"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.55, ease: enterEase }}
          viewport={{ once: true, amount: 0.42 }}
        >
          <span>05</span>
          IMAGE
        </motion.p>

        <motion.h2
          id="image-space-title"
          className="image-space__title"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.7, delay: 0.08, ease: enterEase }}
          viewport={{ once: true, amount: 0.42 }}
        >
          Image <span>Space</span>
        </motion.h2>

        <motion.p
          className="image-space__lead"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.6, delay: 0.16, ease: enterEase }}
          viewport={{ once: true, amount: 0.42 }}
        >
          图像空间 / 图床
        </motion.p>

        <motion.p
          className="image-space__body"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.6, delay: 0.24, ease: enterEase }}
          viewport={{ once: true, amount: 0.42 }}
        >
          用于存放、展示和分享 Jiuin 的图片资源。一个为视觉内容保留的安静空间。
        </motion.p>

        <motion.div
          className="image-space__entry"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.6, delay: 0.48, ease: enterEase }}
          viewport={{ once: true, amount: 0.42 }}
        >
          <Button variant="glass" size="lg" disabled aria-describedby="image-space-status">
            <span>Open Image Space</span>
            <span aria-hidden="true">↗</span>
          </Button>
          <span id="image-space-status" className="image-space__status">COMING LATER</span>
        </motion.div>
      </div>

      <div className="image-space__stage" aria-hidden="true">
        <motion.div
          className="image-space__frame image-space__frame--main"
          initial={frameInitial('translate3d(1.3rem, 1rem, 0) rotate(-5deg) scale(0.97)')}
          whileInView={frameVisible('translate3d(0, 0, 0) rotate(-3deg) scale(1)')}
          transition={{ duration: 0.85, delay: 0.24, ease: enterEase }}
          viewport={{ once: true, amount: 0.35 }}
        >
          <span className="image-space__frame-index">01 / VISUAL</span>
          <span className="image-space__frame-orbit" />
          <span className="image-space__frame-sun" />
          <span className="image-space__frame-ridge image-space__frame-ridge--one" />
          <span className="image-space__frame-ridge image-space__frame-ridge--two" />
          <span className="image-space__frame-caption">JIUIN IMAGE SPACE</span>
        </motion.div>

        <motion.div
          className="image-space__frame image-space__frame--top"
          initial={frameInitial('translate3d(1.8rem, -0.8rem, 0) rotate(7deg) scale(0.96)')}
          whileInView={frameVisible('translate3d(0, 0, 0) rotate(5deg) scale(1)')}
          transition={{ duration: 0.78, delay: 0.32, ease: enterEase }}
          viewport={{ once: true, amount: 0.35 }}
        >
          <span>02</span>
          <i />
        </motion.div>

        <motion.div
          className="image-space__frame image-space__frame--side"
          initial={frameInitial('translate3d(1.2rem, 1.1rem, 0) rotate(8deg) scale(0.96)')}
          whileInView={frameVisible('translate3d(0, 0, 0) rotate(6deg) scale(1)')}
          transition={{ duration: 0.78, delay: 0.38, ease: enterEase }}
          viewport={{ once: true, amount: 0.35 }}
        >
          <span>03</span>
          <i />
        </motion.div>

        <motion.div
          className="image-space__upload-hint"
          initial={frameInitial('translate3d(0.8rem, 1rem, 0)')}
          whileInView={frameVisible('translate3d(0, 0, 0)')}
          transition={{ duration: 0.65, delay: 0.44, ease: enterEase }}
          viewport={{ once: true, amount: 0.35 }}
        >
          <span>DROP IMAGE HERE</span>
          <i aria-hidden="true">+</i>
        </motion.div>
      </div>
    </section>
  )
}
