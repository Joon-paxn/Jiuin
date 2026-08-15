import { SiteLayout } from '../components/layout/SiteLayout'
import { ContentPreviewSection } from '../components/home/ContentPreviewSection'
import { HeroSection } from '../components/home/HeroSection'
import { IntroductionSection } from '../components/home/IntroductionSection'

export function HomePage() {
  return (
    <SiteLayout mainClassName="home-page">
      <HeroSection />
      <IntroductionSection />
      <ContentPreviewSection />
    </SiteLayout>
  )
}
