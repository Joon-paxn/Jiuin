import { motion, useReducedMotion } from 'framer-motion'
import { Button } from '../ui/Button'
import { publicApis } from './publicApiData'

const enterEase = [0.23, 1, 0.32, 1] as const

export function ApiShowcaseSection() {
  const reduceMotion = useReducedMotion()
  const initial = reduceMotion
    ? false
    : { opacity: 0, transform: 'translate3d(0, 1.2rem, 0)' }
  const visible = { opacity: 1, transform: 'translate3d(0, 0, 0)' }

  return (
    <section id="api" className="home-section api-showcase-section" aria-labelledby="api-showcase-title">
      <motion.header
        className="api-showcase__hero"
        initial={initial}
        whileInView={visible}
        transition={{ duration: 0.6, ease: enterEase }}
        viewport={{ once: true, amount: 0.38 }}
      >
        <p className="api-showcase__eyebrow"><span>06</span> PUBLIC SERVICES</p>
        <h2 id="api-showcase-title" className="api-showcase__title">
          API
          <span>公开接口</span>
        </h2>
        <p className="api-showcase__description">为开发者提供部分公开的网站服务与数据接口。</p>
        <div className="api-showcase__actions">
          <Button variant="glass" size="lg" disabled aria-describedby="api-showcase-documentation-status">
            <span>API Documentation</span>
            <span aria-hidden="true">↗</span>
          </Button>
          <span id="api-showcase-documentation-status" className="api-showcase__documentation-status">DOCUMENTATION LATER</span>
        </div>
      </motion.header>

      <motion.div
        className="api-showcase__catalog"
        initial={initial}
        whileInView={visible}
        transition={{ duration: 0.65, delay: 0.12, ease: enterEase }}
        viewport={{ once: true, amount: 0.32 }}
      >
        <div className="api-showcase__catalog-heading">
          <span>PUBLIC API CATALOG</span>
          <span>STATIC PREVIEW / {publicApis.length.toString().padStart(2, '0')}</span>
        </div>

        <ul className="api-showcase__cards">
          {publicApis.map((api, index) => (
            <motion.li
              key={api.id}
              initial={reduceMotion ? false : { opacity: 0, transform: 'translate3d(0, 0.8rem, 0)' }}
              whileInView={visible}
              transition={{ duration: 0.45, delay: 0.2 + index * 0.07, ease: enterEase }}
              viewport={{ once: true, amount: 0.3 }}
            >
              <article className="api-showcase__card">
                <div className="api-showcase__card-topline">
                  <span className="api-showcase__method">{api.method}</span>
                  <span className="api-showcase__card-status">{api.status}</span>
                </div>
                <h3>{api.name}</h3>
                <code>{api.endpoint}</code>
                <p>{api.description}</p>
                <footer>
                  <span>PUBLIC</span>
                  <span>Documentation later</span>
                </footer>
              </article>
            </motion.li>
          ))}
        </ul>
      </motion.div>
    </section>
  )
}
