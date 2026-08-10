import { Card } from '../ui'
import { SectionHeading } from './SectionHeading'

const introductionPoints = ['收藏正在形成', '灵感缓慢生长', '故事等待相遇'] as const

export function IntroductionSection() {
  return (
    <section id="introduction" className="home-section introduction-section motion-reveal motion-delay-2" aria-labelledby="introduction-title">
      <SectionHeading
        eyebrow="ABOUT THIS SPACE"
        title="给思绪一个可以停靠的角落"
        description="这里不是匆忙的信息流，而是一处会随时间慢慢长出轮廓的个人空间。"
      />

      <Card className="introduction-card" variant="soft">
        <p className="introduction-card__lead">霁雪居以温和、清晰的方式，收纳创作、记录与未完成的想象。</p>
        <div className="introduction-card__points" aria-label="空间关键词">
          {introductionPoints.map((point) => (
            <span key={point}>{point}</span>
          ))}
        </div>
      </Card>
    </section>
  )
}
