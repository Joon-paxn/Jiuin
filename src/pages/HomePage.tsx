import { site } from '../config/site'
import { SiteLayout } from '../components/layout/SiteLayout'

export function HomePage() {
  return (
    <SiteLayout>
      <section className="hero" aria-labelledby="hero-title">
        <p className="hero__eyebrow">PERSONAL SPACE · EST. SOON</p>
        <h1 id="hero-title" className="hero__title">
          <span>{site.chineseName}</span>
          <em>{site.name}</em>
        </h1>
        <p className="hero__description">{site.description}</p>
        <div className="hero__future-slot" aria-label="未来视觉内容区域">
          <span>Future visual space</span>
        </div>
      </section>
    </SiteLayout>
  )
}
