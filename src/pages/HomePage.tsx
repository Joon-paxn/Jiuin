import { SiteLayout } from '../components/layout/SiteLayout'
import { HeroSection } from '../components/home/HeroSection'
import { IntroductionSection } from '../components/home/IntroductionSection'
import { UpdatesSection } from '../components/home/UpdatesSection'

export function HomePage() {
  return (
    <SiteLayout mainClassName="home-page">
      <HeroSection />
      <IntroductionSection />
      <UpdatesSection />
    </SiteLayout>
  )
}
