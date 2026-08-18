import { useEffect, useState } from 'react'
import { ecosystemApi } from '../services/api/ecosystem'
import type { SiteStatistics } from '../services/api/types'

export function useSiteStatistics() {
  const [statistics, setStatistics] = useState<SiteStatistics | null>(null)

  useEffect(() => {
    let ignored = false
    void ecosystemApi.getStatistics()
      .then((nextStatistics) => {
        if (!ignored) {
          setStatistics(nextStatistics)
        }
      })
      .catch(() => {
        // Statistics remain a neutral placeholder when the public API is unavailable.
      })
    return () => {
      ignored = true
    }
  }, [])

  return statistics
}
