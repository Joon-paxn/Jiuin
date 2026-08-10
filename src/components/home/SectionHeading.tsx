type SectionHeadingProps = {
  eyebrow: string
  title: string
  description: string
}

export function SectionHeading({ eyebrow, title, description }: SectionHeadingProps) {
  return (
    <header className="section-heading">
      <p className="section-heading__eyebrow">{eyebrow}</p>
      <h2 className="section-heading__title">{title}</h2>
      <p className="section-heading__description">{description}</p>
    </header>
  )
}
