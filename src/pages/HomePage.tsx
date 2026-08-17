import { SiteLayout } from '../components/layout/SiteLayout'
import { HeroSection } from '../components/home/HeroSection'
import {
  AboutSection,
  ApiSection,
  BlogSection,
  ImageSection,
  IntroductionSection,
  UpdatesSection,
} from '../components/home/MainSections'

export function HomePage() {
  return (
    <SiteLayout mainClassName="home-page" footer={null}>
      <HeroSection />
      <IntroductionSection />
      <UpdatesSection />
      <BlogSection />
      <ImageSection />
      <ApiSection />
      <AboutSection />
    </SiteLayout>
  )
}
