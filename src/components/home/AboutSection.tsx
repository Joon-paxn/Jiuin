import { motion, useReducedMotion } from 'framer-motion'
import { useEcosystemStatus } from '../../hooks/useEcosystemStatus'
import { useExternalLinks } from '../../hooks/useExternalLinks'
import { useOnlinePresence } from '../../hooks/useOnlinePresence'
import { useSiteInformation } from '../../hooks/useSiteInformation'
import { useSiteStatistics } from '../../hooks/useSiteStatistics'
import { ExternalLinkConfirmation } from '../links'

const enterEase = [0.23, 1, 0.32, 1] as const

const presenceLabels = {
  connecting: 'CONNECTING',
  online: 'LIVE NOW',
  offline: 'UNAVAILABLE',
  reconnecting: 'RECONNECTING',
} as const

function OnlinePresence() {
  const reduceMotion = useReducedMotion()
  const presence = useOnlinePresence()

  return (
    <motion.div
      className={`about-section__presence is-${presence.status}`}
      initial={reduceMotion ? false : { opacity: 0, transform: 'translate3d(0, 1rem, 0)' }}
      whileInView={{ opacity: 1, transform: 'translate3d(0, 0, 0)' }}
      transition={{ duration: 0.6, delay: 0.16, ease: enterEase }}
      viewport={{ once: true, amount: 0.4 }}
      role="status"
      aria-live="polite"
    >
      <div className="about-section__presence-label">
        <i aria-hidden="true" />
        <span>{presenceLabels[presence.status]}</span>
      </div>
      <motion.span
        key={presence.count}
        className="about-section__presence-count"
        initial={reduceMotion ? false : { opacity: 0, transform: 'translate3d(0, 0.45rem, 0)' }}
        animate={{ opacity: 1, transform: 'translate3d(0, 0, 0)' }}
        transition={{ duration: 0.28, ease: enterEase }}
      >
        {presence.status === 'online' ? presence.count.toString() : '—'}
      </motion.span>
      <span className="about-section__presence-copy">
        {presence.status === 'online' ? '当前保持有效连接的浏览器标签页' : '暂时无法获取实时在线人数'}
      </span>
    </motion.div>
  )
}

export function AboutSection() {
  const reduceMotion = useReducedMotion()
  const siteInformation = useSiteInformation()
  const statistics = useSiteStatistics()
  const ecosystemStatus = useEcosystemStatus()
  const externalLinks = useExternalLinks()
  const initial = reduceMotion ? false : { opacity: 0, transform: 'translate3d(0, 1.2rem, 0)' }
  const visible = { opacity: 1, transform: 'translate3d(0, 0, 0)' }

  return (
    <section id="about" className="home-section about-section" aria-labelledby="about-section-title">
      <div className="about-section__hero">
        <motion.div
          className="about-section__intro"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.6, ease: enterEase }}
          viewport={{ once: true, amount: 0.4 }}
        >
          <p className="about-section__eyebrow"><span>07</span> ABOUT</p>
          <h2 id="about-section-title" className="about-section__title">关于<span>霁雪居</span></h2>
          <p>一个持续构建中的个人空间，在这里留下正在发生的内容，也为下一次抵达保留一点位置。</p>
        </motion.div>
        <OnlinePresence />
      </div>

      <div className="about-section__ledger">
        <motion.div
          className="about-section__detail about-section__detail--statistics"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.5, delay: 0.08, ease: enterEase }}
          viewport={{ once: true, amount: 0.25 }}
        >
          <span className="about-section__label">SITE STATISTICS</span>
          <strong>{statistics?.totalViews ?? '—'}</strong>
          <span>公开访问记录</span>
        </motion.div>

        <motion.div
          className="about-section__detail"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.5, delay: 0.14, ease: enterEase }}
          viewport={{ once: true, amount: 0.25 }}
        >
          <span className="about-section__label">CONTACT</span>
          <p>公开联系方式将在这里开放。</p>
          <span>PUBLIC CONTACT / LATER</span>
        </motion.div>

        <motion.div
          className="about-section__detail about-section__detail--links"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.5, delay: 0.2, ease: enterEase }}
          viewport={{ once: true, amount: 0.25 }}
        >
          <span className="about-section__label">FRIENDS / LINKS</span>
          <div>
            {externalLinks.length > 0
              ? externalLinks.map((link) => <ExternalLinkConfirmation key={link.url} link={link} />)
              : <span className="about-section__placeholder">友链将在这里相遇。</span>}
          </div>
        </motion.div>

        <motion.div
          className="about-section__detail about-section__detail--site"
          initial={initial}
          whileInView={visible}
          transition={{ duration: 0.5, delay: 0.26, ease: enterEase }}
          viewport={{ once: true, amount: 0.25 }}
        >
          <span className="about-section__label">SITE INFORMATION</span>
          <p>{siteInformation.project} / {siteInformation.name}</p>
          <span>{siteInformation.domain} · {ecosystemStatus.site.toUpperCase()}</span>
          <small>© {siteInformation.copyrightYear ?? '—'} {siteInformation.copyrightText ?? siteInformation.project}</small>
        </motion.div>
      </div>
    </section>
  )
}
