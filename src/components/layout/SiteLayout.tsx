import type { PropsWithChildren, ReactNode } from 'react'
import { BackgroundSystem, type BackgroundConfig } from '../background'
import { Footer } from '../footer/Footer'
import { Header } from '../header/Header'
import { Live2DFloating } from '../Live2D'
import { MusicPlayer } from '../music'
import { ScrollProgress } from '../progress/ScrollProgress'

type SiteLayoutProps = PropsWithChildren<{
  background?: BackgroundConfig
  mainClassName?: string
  footer?: ReactNode
}>

export function SiteLayout({ children, background, mainClassName, footer }: SiteLayoutProps) {
  return (
    <div className="site-shell">
      <BackgroundSystem config={background} />
      <ScrollProgress />
      <Header />
      <main className={mainClassName ? `site-main ${mainClassName}` : 'site-main'}>{children}</main>
      {footer === undefined ? <Footer /> : footer}
      <Live2DFloating />
      <MusicPlayer />
    </div>
  )
}
