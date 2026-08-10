import { HomePage } from '../pages/HomePage'
import { BackgroundProvider } from '../components/background'
import { EcosystemResourceBootstrap } from '../components/ecosystem'
import { ThemeProvider } from '../components/theme/ThemeProvider'

export function App() {
  return (
    <BackgroundProvider>
      <ThemeProvider>
        <EcosystemResourceBootstrap />
        <HomePage />
      </ThemeProvider>
    </BackgroundProvider>
  )
}
