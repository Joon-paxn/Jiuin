import { Card } from '../ui'
import { SectionHeading } from './SectionHeading'

const previewItems = [
  {
    category: 'PROJECTS',
    title: '项目展示',
    description: '为正在探索的作品、实验与协作保留一片清晰的陈列空间。',
  },
  {
    category: 'NOTES',
    title: '文章记录',
    description: '未来会在这里安放观察、灵感与值得反复翻阅的片段。',
  },
  {
    category: 'COLLECTION',
    title: '资源收藏',
    description: '收集那些让创作过程变得更轻盈的小工具与参考。',
  },
] as const

export function ContentPreviewSection() {
  return (
    <section id="content-preview" className="home-section content-preview motion-reveal motion-delay-3" aria-labelledby="content-preview-title">
      <SectionHeading
        eyebrow="IN THE MAKING"
        title="正在等待展开的内容"
        description="项目、文字与资源会在合适的时刻抵达；现在先为它们准备好位置。"
      />

      <div className="content-preview__grid">
        {previewItems.map((item, index) => (
          <Card key={item.category} className="preview-card" variant="interactive">
            <div className="preview-card__meta">
              <span>{item.category}</span>
              <span>0{index + 1}</span>
            </div>
            <h3>{item.title}</h3>
            <p>{item.description}</p>
            <span className="preview-card__status">Coming softly</span>
          </Card>
        ))}
      </div>
    </section>
  )
}
