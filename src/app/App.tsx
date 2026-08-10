import { HomePage } from '../pages/HomePage'
import { BackgroundProvider } from '../components/background'
import { EcosystemResourceBootstrap } from '../components/ecosystem'
import { ThemeProvider } from '../components/theme/ThemeProvider'
import { TemporaryAnimationModule } from '../experiments/temporary-animation'

export function App() {
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
