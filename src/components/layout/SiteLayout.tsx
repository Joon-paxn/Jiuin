import type { PropsWithChildren } from 'react'
import { BackgroundLayer, type BackgroundConfig } from '../background/BackgroundLayer'
import { Footer } from '../footer/Footer'
import { Header } from '../header/Header'
import { ScrollProgress } from '../progress/ScrollProgress'

type SiteLayoutProps = PropsWithChildren<{
  background?: BackgroundConfig
}>

export function SiteLayout({ children, background }: SiteLayoutProps) {
  return (
    <div className="site-shell">
      <BackgroundLayer config={background} />
      <ScrollProgress />
      <Header />
      <main className="site-main">{children}</main>
      <Footer />
    </div>
  )
}
