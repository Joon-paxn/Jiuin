import { site } from '../config/site'
import { SiteLayout } from '../components/layout/SiteLayout'
import { Divider, GlassPanel } from '../components/ui'

export function HomePage() {
  return (
    <SiteLayout>
      <section className="hero motion-page-enter motion-page-enter--hero" aria-labelledby="hero-title">
        <p className="hero__eyebrow">PERSONAL SPACE · EST. SOON</p>
        <h1 id="hero-title" className="hero__title">
          <span>{site.chineseName}</span>
          <em>{site.name}</em>
        </h1>
        <p className="hero__description">{site.description}</p>
        <GlassPanel className="hero__future-slot" aria-label="未来视觉内容区域" role="region" tone="soft">
          <div className="hero__future-copy">
            <span className="hero__future-label">Future visual space</span>
            <Divider />
            <span>Visual layer reserved for the next story.</span>
          </div>
          <div className="hero__future-orbit" aria-hidden="true" />
        </GlassPanel>
      </section>
    </SiteLayout>
  )
}
