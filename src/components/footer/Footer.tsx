import { useSiteInformation } from '../../hooks/useSiteInformation'
import { siteRoutes } from '../../routes/siteRoutes'

const externalLinkPlaceholders = ['GitHub', '友链', '联系'] as const

export function Footer() {
  const siteInformation = useSiteInformation()

  return (
    <footer className="site-footer motion-reveal motion-reveal--footer">
      <div className="site-footer__brand">
        <span className="site-footer__name">{siteInformation.project}</span>
        <strong>{siteInformation.name}</strong>
        <p>一个正在缓缓展开的个人空间。</p>
      </div>

      <nav className="site-footer__navigation" aria-label="页脚站内导航">
        <span className="site-footer__label">站内导航</span>
        <div>
          {siteRoutes.map((route) => (
            <a key={route.path} href={route.path}>{route.label}</a>
          ))}
        </div>
      </nav>

      <div className="site-footer__links" aria-label="外部链接（预留）">
        <span className="site-footer__label">链接区域</span>
        <div>
          {externalLinkPlaceholders.map((label) => (
            <span key={label} className="site-footer__link-placeholder">{label}</span>
          ))}
        </div>
      </div>

      <p className="site-footer__copyright">
        <span>{siteInformation.domain}</span>
        <span>© <span aria-label="版权年份由服务端提供">{siteInformation.copyrightYear ?? '——'}</span> {siteInformation.copyrightText ?? siteInformation.project}</span>
      </p>
    </footer>
  )
}
