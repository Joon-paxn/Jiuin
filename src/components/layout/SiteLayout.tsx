import type { PropsWithChildren } from 'react'
import { BackgroundLayer } from '../background/BackgroundLayer'
import { Footer } from '../footer/Footer'
import { Header } from '../header/Header'

export function SiteLayout({ children }: PropsWithChildren) {
  return (
    <div className="site-shell">
      <BackgroundLayer />
      <Header />
      <main className="site-main">{children}</main>
      <Footer />
    </div>
  )
}
