import { Button, GlassPanel } from '../ui'

export function HeroSection() {
  return (
    <section className="home-hero" aria-labelledby="hero-title">
      <div className="home-hero__copy">
        <p className="home-hero__welcome motion-fade-in">欢迎来到</p>
        <h1 id="hero-title" className="home-hero__title motion-fade-in motion-delay-1">
          <span>霁雪居</span>
          <em>Jiuin</em>
        </h1>
        <p className="home-hero__description motion-slide-up motion-delay-2">
          一个为灵感、片段与温柔想象留出空间的个人小世界。
        </p>
        <div className="home-hero__actions motion-slide-up motion-delay-3" aria-label="首页操作">
          <Button href="#content-preview">探索内容</Button>
          <Button href="#introduction" variant="glass">了解这里</Button>
        </div>
      </div>

      <GlassPanel className="hero-visual motion-reveal motion-delay-2" role="region" aria-label="未来视觉展示区域" tone="soft">
        <div className="hero-visual__glow" aria-hidden="true" />
        <div className="hero-visual__orbit hero-visual__orbit--outer" aria-hidden="true" />
        <div className="hero-visual__orbit hero-visual__orbit--inner" aria-hidden="true" />
        <div className="hero-visual__content">
          <span className="hero-visual__eyebrow">VISUAL PLACEHOLDER</span>
          <strong>Future scene</strong>
          <span>为下一段视觉故事预留</span>
        </div>
      </GlassPanel>
    </section>
  )
}
