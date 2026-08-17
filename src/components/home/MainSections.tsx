import { useLayoutEffect, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { ExternalLinkConfirmation } from '../links'
import { Footer } from '../footer/Footer'
import { useEcosystemStatus } from '../../hooks/useEcosystemStatus'
import { useExternalLinks } from '../../hooks/useExternalLinks'
import { useSectionReveal } from '../../hooks/useSectionReveal'
import { useSiteInformation } from '../../hooks/useSiteInformation'
import { updatesData } from './updatesData'

type SectionProps = { className?: string }

function SectionFrame({ id, number, eyebrow, title, description, children, className = '' }: SectionProps & {
  id: string
  number: string
  eyebrow: string
  title: string
  description: string
  children?: ReactNode
}) {
  const { ref, isVisible } = useSectionReveal<HTMLElement>()
  return (
    <section ref={ref} id={id} className={`full-screen-section ${className}`} data-revealed={isVisible} aria-labelledby={`${id}-title`}>
      <div className="section-shell">
        <div className="section-rail" aria-hidden="true">
          <span>0{number}</span>
          <i />
          <span>JIUIN</span>
        </div>
        <div className="section-intro">
          <p className="section-eyebrow">{eyebrow}</p>
          <h2 id={`${id}-title`} className="section-title">{title}</h2>
          <p className="section-description">{description}</p>
        </div>
        {children}
      </div>
    </section>
  )
}

export function IntroductionSection() {
  return (
    <SectionFrame
      id="introduction"
      number="2"
      eyebrow="INTRODUCTION / 关于"
      title="关于霁雪居"
      description="霁雪居是一个持续构建中的个人空间。这里收纳作品、声音、想法与仍在生长的实验，让每一次停留都有一点安静的余韵。"
      className="introduction-page"
    >
      <div className="introduction-page__body reveal-stagger">
        <div className="introduction-page__quote">
          <span className="quote-mark">“</span>
          <p>把注意力交还给当下，在缓慢中保存真实的质地。</p>
          <span className="quote-credit">A PERSONAL SPACE IN PROGRESS</span>
        </div>
        <div className="introduction-page__notes">
          <div><span>01</span><strong>收藏正在形成</strong><p>把值得再次打开的作品与参考，放进可以呼吸的秩序里。</p></div>
          <div><span>02</span><strong>灵感缓慢生长</strong><p>不追逐即时答案，允许未完成的想法继续发酵。</p></div>
          <div><span>03</span><strong>故事等待相遇</strong><p>每个来到这里的人，都可以带走一小段属于自己的回声。</p></div>
        </div>
      </div>
      <div className="introduction-page__seal" aria-hidden="true"><span>霁</span><i /></div>
    </SectionFrame>
  )
}

export function UpdatesSection() {
  const { ref, isVisible } = useSectionReveal<HTMLElement>()
  const viewportRef = useRef<HTMLDivElement>(null)
  const trackRef = useRef<HTMLDivElement>(null)
  const [timelineShift, setTimelineShift] = useState(0)

  useLayoutEffect(() => {
    if (!isVisible || !viewportRef.current || !trackRef.current || window.innerWidth <= 768) return
    const alignLatest = () => {
      const viewport = viewportRef.current
      const track = trackRef.current
      if (!viewport || !track) return
      setTimelineShift(Math.max(0, track.scrollWidth - viewport.clientWidth))
    }
    alignLatest()
    const resizeObserver = new ResizeObserver(alignLatest)
    resizeObserver.observe(viewportRef.current)
    return () => resizeObserver.disconnect()
  }, [isVisible])

  return (
    <section ref={ref} id="updates" className="full-screen-section updates-page" data-revealed={isVisible} aria-labelledby="updates-title">
      <div className="section-shell section-shell--wide">
        <div className="section-rail" aria-hidden="true"><span>03</span><i /><span>HISTORY</span></div>
        <div className="section-intro updates-page__intro">
          <p className="section-eyebrow">UPDATES / 开发记录</p>
          <h2 id="updates-title" className="section-title">正在发生的事</h2>
          <p className="section-description">一条仍在向右延伸的时间轴。这里先使用结构化 Mock Data，未来可以直接替换为 GitHub Commit 响应。</p>
        </div>
        <div ref={viewportRef} className="updates-timeline" aria-label="Jiuin 开发历史">
          <div ref={trackRef} className="updates-timeline__track" style={{ transform: `translate3d(-${timelineShift}px, 0, 0)` }}>
            <div className="updates-timeline__axis" />
            {updatesData.map((update, index) => (
              <article key={`${update.date}-${update.title}`} className={`update-node update-node--${update.period}`}>
                <span className="update-node__dot" aria-hidden="true" />
                <div className="update-node__card">
                  <div className="update-node__meta"><time>{update.date} · {update.time}</time><span>0{index + 1}</span></div>
                  <h3>{update.title}</h3>
                  <p>{update.description}</p>
                </div>
              </article>
            ))}
          </div>
        </div>
        <p className="updates-page__hint">OLD → NEW <span>最新提交位于时间轴尽头</span></p>
      </div>
    </section>
  )
}

export function BlogSection() {
  return (
    <SectionFrame
      id="blog"
      number="4"
      eyebrow="BLOG / 预留空间"
      title="故事还在路上"
      description="Blog 暂时不承担内容系统的职责。它会在合适的时机，成为记录长文、观察和过程的另一扇门。"
      className="blog-page"
    >
      <div className="coming-soon-panel reveal-stagger">
        <span className="coming-soon-panel__index">04 / FUTURE MODULE</span>
        <strong>Coming later</strong>
        <p>不是缺席，而是为值得写下的内容保留位置。</p>
        <span className="coming-soon-panel__line" aria-hidden="true" />
      </div>
    </SectionFrame>
  )
}

export function ImageSection() {
  return (
    <SectionFrame
      id="image"
      number="5"
      eyebrow="IMAGE / 图像服务"
      title="让图像有一个安静的住处"
      description="Image 是未来的图片服务入口。上传、管理、预览与分享，会在这里形成一套轻盈而清楚的工作流。"
      className="image-page"
    >
      <div className="image-service-panel reveal-stagger">
        <div className="image-service-panel__canvas" aria-hidden="true">
          <div className="image-service-panel__frame"><span>J / 05</span><i /><b /></div>
          <span className="image-service-panel__caption">A PLACE FOR IMAGES</span>
        </div>
        <div className="image-service-panel__details">
          <span className="section-eyebrow">IMAGE HOSTING</span>
          <h3>轻量、清晰、可分享。</h3>
          <p>上传与资源管理功能将在后续阶段接入。当前这里只展示它在站内的位置与气质。</p>
          <span className="template-status">TEMPLATE / NOT CONNECTED</span>
        </div>
      </div>
    </SectionFrame>
  )
}

const publicApis = [
  { method: 'GET', path: '/api/v1/site/info', name: 'Site Information', description: '获取公开的网站名称、项目与域名信息。' },
  { method: 'GET', path: '/api/v1/statistics', name: 'Site Statistics', description: '读取公开的站内访问统计摘要。' },
  { method: 'GET', path: '/api/v1/status', name: 'Service Status', description: '查看主站与公开服务的运行状态。' },
  { method: 'GET', path: '/api/v1/resources', name: 'Resource Manifest', description: '读取公开资源清单与缓存策略。' },
] as const

export function ApiSection() {
  return (
    <SectionFrame
      id="api"
      number="6"
      eyebrow="API / PUBLIC SERVICES"
      title="给公开服务一张清楚的名片"
      description="这里仅展示当前后端已经确认的公开接口，不包含管理、鉴权、数据库或内部服务地址。"
      className="api-page"
    >
      <div className="api-page__grid reveal-stagger">
        {publicApis.map((api, index) => (
          <article key={api.path} className="api-card">
            <div className="api-card__top"><span className="api-card__number">0{index + 1}</span><span className="api-card__method">{api.method}</span></div>
            <h3>{api.name}</h3>
            <code>{api.path}</code>
            <p>{api.description}</p>
            <span className="api-card__arrow" aria-hidden="true">↗</span>
          </article>
        ))}
      </div>
    </SectionFrame>
  )
}

export function AboutSection() {
  const siteInformation = useSiteInformation()
  const status = useEcosystemStatus()
  const externalLinks = useExternalLinks()

  return (
    <SectionFrame
      id="about"
      number="7"
      eyebrow="ABOUT / 关于"
      title="在这里，继续保持好奇"
      description="谢谢你走到最后一页。霁雪居会继续更新，也会继续为新的声音、图像与故事留出空间。"
      className="about-page"
    >
      <div className="about-page__grid reveal-stagger">
        <div className="about-page__column"><span className="section-label">SITE DATA</span><strong>{siteInformation.domain}</strong><p>公开站点信息由 Shared API 提供。</p><dl><div><dt>API</dt><dd>{status.api === 'online' ? 'ONLINE' : 'PENDING'}</dd></div><div><dt>SERVICES</dt><dd>{status.services.length || '—'}</dd></div></dl></div>
        <div className="about-page__column"><span className="section-label">CONTACT</span><strong>保持联系</strong><p>公开联系方式尚未配置，新的入口会在确认后出现在这里。</p><span className="about-page__placeholder">CONTACT / TO BE CONFIGURED</span></div>
        <div className="about-page__column"><span className="section-label">LINKS</span><strong>邻近的空间</strong><div className="about-page__links">{externalLinks.length > 0 ? externalLinks.map((link) => <ExternalLinkConfirmation key={link.url} link={link} />) : <span>暂无已确认的友情链接</span>}</div></div>
      </div>
      <Footer />
    </SectionFrame>
  )
}
