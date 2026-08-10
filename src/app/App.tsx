import { HomePage } from '../pages/HomePage'
import { BackgroundProvider } from '../components/background'
import { ThemeProvider } from '../components/theme/ThemeProvider'

export function App() {
  return (
    <BackgroundProvider>
      <ThemeProvider>
        <HomePage />
      </ThemeProvider>
    </BackgroundProvider>
  )
}
