import { motion, useReducedMotion } from 'framer-motion'

const enterTransition = {
  duration: 0.6,
  ease: [0.23, 1, 0.32, 1] as const,
}

export function IntroductionSection() {
  const reduceMotion = useReducedMotion()
  const initial = reduceMotion ? false : { opacity: 0, transform: 'translate3d(0, 1.5rem, 0)' }
  const visible = { opacity: 1, transform: 'translate3d(0, 0, 0)' }

  return (
    <section id="introduction" className="about-section home-section introduction-section" aria-labelledby="introduction-title">
      <div className="introduction-section__copy">
        <motion.p
          className="introduction-section__eyebrow"
          initial={initial}
          whileInView={visible}
          transition={enterTransition}
          viewport={{ once: true, amount: 0.45 }}
        >
          <span>02</span>
          ABOUT THIS SPACE
        </motion.p>
        <motion.h2
          id="introduction-title"
          className="introduction-section__title"
          initial={initial}
          whileInView={visible}
          transition={{ ...enterTransition, delay: 0.08 }}
          viewport={{ once: true, amount: 0.45 }}
        >
          关于<span>霁雪居</span>
        </motion.h2>
        <motion.p
          className="introduction-section__lead"
          initial={initial}
          whileInView={visible}
          transition={{ ...enterTransition, delay: 0.16 }}
          viewport={{ once: true, amount: 0.45 }}
        >
          这是一个持续构建中的个人空间。
        </motion.p>
        <motion.p
          className="introduction-section__body"
          initial={initial}
          whileInView={visible}
          transition={{ ...enterTransition, delay: 0.24 }}
          viewport={{ once: true, amount: 0.45 }}
        >
          这里会安放项目、创作、开发过程，以及一些愿意反复回看的内容。霁雪居不是一次完成的网站，而是一处会随着时间慢慢改变的所在。
        </motion.p>
        <motion.div
          className="introduction-section__signature"
          initial={initial}
          whileInView={visible}
          transition={{ ...enterTransition, delay: 0.32 }}
          viewport={{ once: true, amount: 0.45 }}
        >
          <span>PERSONAL SPACE</span>
          <i aria-hidden="true" />
          <span>IN PROGRESS</span>
        </motion.div>
      </div>

      <motion.div
        className="introduction-section__visual"
        aria-hidden="true"
        initial={reduceMotion ? false : { opacity: 0, transform: 'translate3d(1.25rem, 0.75rem, 0) scale(0.97)' }}
        whileInView={{ opacity: 1, transform: 'translate3d(0, 0, 0) scale(1)' }}
        transition={{ ...enterTransition, delay: 0.2 }}
        viewport={{ once: true, amount: 0.35 }}
      >
        <div className="introduction-section__visual-panel">
          <span className="introduction-section__visual-index">J / 02</span>
          <div className="introduction-section__visual-window">
            <span className="introduction-section__visual-character">居</span>
            <i className="introduction-section__visual-line introduction-section__visual-line--vertical" />
            <i className="introduction-section__visual-line introduction-section__visual-line--horizontal" />
            <span className="introduction-section__visual-corner introduction-section__visual-corner--top" />
            <span className="introduction-section__visual-corner introduction-section__visual-corner--bottom" />
          </div>
          <span className="introduction-section__visual-caption">ROOM FOR IDEAS</span>
        </div>
        <p>留一扇窗，给仍在发生的想法。</p>
      </motion.div>
    </section>
  )
}
