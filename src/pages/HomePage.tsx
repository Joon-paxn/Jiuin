import { SiteLayout } from '../components/layout/SiteLayout'
import { HeroSection } from '../components/home/HeroSection'
import { IntroductionSection } from '../components/home/IntroductionSection'
import { UpdatesSection } from '../components/home/UpdatesSection'
import { BlogPreviewSection } from '../components/home/BlogPreviewSection'
import { ImageSpaceSection } from '../components/home/ImageSpaceSection'
import { ApiShowcaseSection } from '../components/home/ApiShowcaseSection'
import { AboutSection } from '../components/home/AboutSection'

export function HomePage() {
  return (
    <SiteLayout mainClassName="home-page">
      <HeroSection />
      <IntroductionSection />
      <UpdatesSection />
      <BlogPreviewSection />
      <ImageSpaceSection />
      <ApiShowcaseSection />
      <AboutSection />
    </SiteLayout>
  )
}
