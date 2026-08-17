import { AnimatePresence, LayoutGroup, motion, useReducedMotion } from 'framer-motion'
import { useEffect, useState } from 'react'
import { site } from '../../config/site'
import { useScrollThreshold } from '../../hooks/useScrollThreshold'
import { siteRoutes } from '../../routes/siteRoutes'
import { classNames } from '../../utils/classNames'

const transition = {
  duration: 0.3,
  ease: [0.23, 1, 0.32, 1] as const,
}

function currentPath() {
  return typeof window === 'undefined' ? '/' : window.location.pathname
}

export function Header() {
  const isScrolled = useScrollThreshold()
  const reduceMotion = useReducedMotion()
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)
  const [activePath, setActivePath] = useState(currentPath)

  useEffect(() => {
    const syncPath = () => setActivePath(currentPath())
    window.addEventListener('popstate', syncPath)
    return () => window.removeEventListener('popstate', syncPath)
  }, [])

  useEffect(() => {
    if (!isMobileMenuOpen) {
      return
    }

    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsMobileMenuOpen(false)
      }
    }

    window.addEventListener('keydown', closeOnEscape)
    return () => window.removeEventListener('keydown', closeOnEscape)
  }, [isMobileMenuOpen])

  const navigate = (path: string) => {
    setActivePath(path)
    setIsMobileMenuOpen(false)
  }

  return (
    <header className={classNames('site-header', isScrolled && 'is-floating')} data-mode={isScrolled ? 'floating' : 'top'}>
      <LayoutGroup id="site-header-navigation">
        <motion.div
          initial={false}
          animate={{ transform: reduceMotion || !isScrolled ? 'translateY(0)' : 'translateY(14px)' }}
          transition={transition}
          className="site-header__frame"
        >
          <div className="site-header__inner">
            <a className="brand" href="/" aria-label={`${site.chineseName}首页`} onClick={() => navigate('/')}>
              <img className="brand__mark" src={site.iconPath} alt="" />
              <span className="brand__copy">
                <strong>{site.chineseName}</strong>
                <small>{site.name}</small>
              </span>
            </a>

            <nav className="site-nav" aria-label="主导航">
              {siteRoutes.map((route) => {
                const isActive = activePath === route.path
                return (
                  <a
                    key={route.path}
                    href={route.path}
                    aria-current={isActive ? 'page' : undefined}
                    className={route.status === 'planned' ? 'site-nav__link is-planned' : 'site-nav__link'}
                    onClick={() => navigate(route.path)}
                  >
                    {isActive && <motion.span className="site-nav__indicator" layoutId="site-nav-indicator" transition={transition} />}
                    <span>{route.label}</span>
                  </a>
                )
              })}
            </nav>

            <button
              type="button"
              className="site-header__menu-trigger"
              aria-controls="site-mobile-navigation"
              aria-expanded={isMobileMenuOpen}
              aria-label={isMobileMenuOpen ? '关闭主导航' : '打开主导航'}
              onClick={() => setIsMobileMenuOpen((isOpen) => !isOpen)}
            >
              <span />
              <span />
            </button>
          </div>
        </motion.div>

        <AnimatePresence initial={false}>
          {isMobileMenuOpen && (
            <motion.nav
              id="site-mobile-navigation"
              className="site-mobile-nav"
              aria-label="移动端主导航"
              initial={reduceMotion ? { opacity: 0 } : { opacity: 0, transform: 'translateY(-0.5rem) scale(0.97)' }}
              animate={reduceMotion ? { opacity: 1 } : { opacity: 1, transform: 'translateY(0) scale(1)' }}
              exit={reduceMotion ? { opacity: 0 } : { opacity: 0, transform: 'translateY(-0.5rem) scale(0.97)' }}
              transition={{ duration: 0.22, ease: transition.ease }}
            >
              {siteRoutes.map((route) => {
                const isActive = activePath === route.path
                return (
                  <a
                    key={route.path}
                    href={route.path}
                    aria-current={isActive ? 'page' : undefined}
                    className={route.status === 'planned' ? 'site-mobile-nav__link is-planned' : 'site-mobile-nav__link'}
                    onClick={() => navigate(route.path)}
                  >
                    <span>{route.label}</span>
                    {isActive && <span className="site-mobile-nav__active" aria-hidden="true" />}
                  </a>
                )
              })}
            </motion.nav>
          )}
        </AnimatePresence>
      </LayoutGroup>
    </header>
  )
}
