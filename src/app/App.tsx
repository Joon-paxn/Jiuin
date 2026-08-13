import { HomePage } from '../pages/HomePage'
import { BackgroundProvider } from '../components/background'
import { EcosystemResourceBootstrap } from '../components/ecosystem'
import { ThemeProvider } from '../components/theme/ThemeProvider'
import { TemporaryAnimationModule } from '../experiments/temporary-animation'
import { useLenisScroll } from '../hooks/useLenisScroll'

export function App() {
  useLenisScroll()

  return (
    <BackgroundProvider>
      <ThemeProvider>
        <EcosystemResourceBootstrap />
        <TemporaryAnimationModule>
          <HomePage />
        </TemporaryAnimationModule>
      </ThemeProvider>
    </BackgroundProvider>
  )
}
