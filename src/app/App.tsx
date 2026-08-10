import { HomePage } from '../pages/HomePage'
import { ThemeProvider } from '../components/theme/ThemeProvider'

export function App() {
  return (
    <ThemeProvider>
      <HomePage />
    </ThemeProvider>
  )
}
