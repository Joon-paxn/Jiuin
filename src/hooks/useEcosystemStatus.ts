import { useEffect, useState } from 'react'
import { ecosystemApi } from '../services/api/ecosystem'
import type { EcosystemStatus } from '../services/api/types'

const fallbackStatus: EcosystemStatus = {
  site: 'unknown',
  api: 'unknown',
  services: [],
  checkedAt: '',
}

export function useEcosystemStatus() {
  const [status, setStatus] = useState<EcosystemStatus>(fallbackStatus)

  useEffect(() => {
    let ignored = false
    void ecosystemApi.getStatus()
      .then((nextStatus) => {
        if (!ignored) {
          setStatus(nextStatus)
        }
      })
      .catch(() => {
        // A status failure should not prevent the site footer from rendering.
      })

    return () => {
      ignored = true
    }
  }, [])

  return status
}
