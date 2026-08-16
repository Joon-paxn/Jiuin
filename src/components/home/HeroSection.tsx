import { Button } from '../ui'
import { useHeroVisual } from './heroVisual'

export function HeroSection() {
  const heroVisual = useHeroVisual()

  return (
    <section className="hero-section home-hero" aria-labelledby="hero-title">
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

      <div
        className="hero-visual motion-reveal motion-delay-2"
        role="img"
        aria-label="霁雪居视觉插画"
        data-hero-visual-state={heroVisual.state}
        data-hero-visual-hd={heroVisual.hdReady ? 'ready' : 'pending'}
      >
        {heroVisual.lowReady && (
          <img className="hero-visual__image hero-visual__image--low" src={heroVisual.low} alt="" />
        )}
        {heroVisual.hdReady && (
          <img className="hero-visual__image hero-visual__image--hd" src={heroVisual.hd} alt="" />
        )}
      </div>
    </section>
  )
}
