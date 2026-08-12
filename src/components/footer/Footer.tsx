import { ExternalLinkConfirmation } from '../links'
import { useEcosystemStatus } from '../../hooks/useEcosystemStatus'
import { useExternalLinks } from '../../hooks/useExternalLinks'
import { useSiteInformation } from '../../hooks/useSiteInformation'
import { siteRoutes } from '../../routes/siteRoutes'

const statusLabels = {
  online: '正常运行',
  degraded: '服务降级',
  offline: '暂不可用',
  unknown: '状态待确认',
} as const

export function Footer() {
  const siteInformation = useSiteInformation()
  const status = useEcosystemStatus()
  const externalLinks = useExternalLinks()

  return (
    <footer className="site-footer">
      <div className="site-footer__inner motion-reveal motion-reveal--footer">
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
            {externalLinks.length > 0
              ? externalLinks.map((link) => <ExternalLinkConfirmation key={link.url} link={link} />)
              : <span className="site-footer__link-placeholder">暂无外链</span>}
          </div>
        </div>

        <p className="site-footer__copyright">
          <span>{siteInformation.domain}</span>
          <span className={`site-footer__status is-${status.site}`} role="status">
            <i aria-hidden="true" /> 网站状态：{statusLabels[status.site]}
          </span>
          <span>© <span aria-label="版权年份由服务端提供">{siteInformation.copyrightYear ?? '——'}</span> {siteInformation.copyrightText ?? siteInformation.project}</span>
        </p>
      </div>
    </footer>
  )
}
