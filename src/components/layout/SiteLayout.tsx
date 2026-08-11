import type { PropsWithChildren } from 'react'
import { BackgroundSystem, type BackgroundConfig } from '../background'
import { Footer } from '../footer/Footer'
import { Header } from '../header/Header'
import { Live2DFloating } from '../Live2D'
import { MusicPlayer } from '../music'
import { ScrollProgress } from '../progress/ScrollProgress'

type SiteLayoutProps = PropsWithChildren<{
  background?: BackgroundConfig
}>

export function SiteLayout({ children, background }: SiteLayoutProps) {
  return (
    <div className="site-shell">
      <BackgroundSystem config={background} />
      <ScrollProgress />
      <Header />
      <main className="site-main">{children}</main>
      <Footer />
      <Live2DFloating />
      <MusicPlayer />
    </div>
  )
}
