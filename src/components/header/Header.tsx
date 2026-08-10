import { site } from '../../config/site'
import { useScrollThreshold } from '../../hooks/useScrollThreshold'
import { siteRoutes } from '../../routes/siteRoutes'
import { classNames } from '../../utils/classNames'

export function Header() {
  const isScrolled = useScrollThreshold()

  return (
    <header className={classNames('site-header', 'motion-page-enter', 'motion-page-enter--header', isScrolled && 'is-scrolled')}>
      <a className="brand" href="/" aria-label={`${site.chineseName}首页`}>
        <span className="brand__mark" aria-hidden="true">J</span>
        <span className="brand__copy">
          <strong>{site.chineseName}</strong>
          <small>{site.name}</small>
        </span>
      </a>

      <nav className="site-nav" aria-label="主导航">
        {siteRoutes.map((route) => (
          <a
            key={route.path}
            href={route.path}
            aria-current={route.path === '/' ? 'page' : undefined}
            className={route.status === 'planned' ? 'site-nav__link is-planned' : 'site-nav__link'}
          >
            {route.label}
          </a>
        ))}
      </nav>
    </header>
  )
}
