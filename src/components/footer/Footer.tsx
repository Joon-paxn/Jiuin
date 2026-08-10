import { site } from '../../config/site'

const externalLinkPlaceholders = ['GitHub', '友链', '联系'] as const

export function Footer() {
  return (
    <footer className="site-footer motion-reveal motion-reveal--footer">
      <div className="site-footer__identity">
        <span className="site-footer__name">{site.name}</span>
        <span>{site.chineseName}</span>
        <span>{site.domain}</span>
      </div>

      <div className="site-footer__links" aria-label="外部链接（预留）">
        {externalLinkPlaceholders.map((label) => (
          <span key={label} className="site-footer__link-placeholder">{label}</span>
        ))}
      </div>

      <p className="site-footer__copyright">© <span aria-label="版权年份由服务端提供">——</span> {site.name}</p>
    </footer>
  )
}
