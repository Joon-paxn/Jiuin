import { useEffect } from 'react'
import { ecosystemApi } from '../../services/api/ecosystem'
import { registerResourceCache } from '../../services/cache/resourceCache'
import { preloadResourcesByPriority } from '../../services/cache/resourcePriority'

/** Invisible app-level bridge for cache registration and server resource priorities. */
export function EcosystemResourceBootstrap() {
  useEffect(() => {
    registerResourceCache()

    void ecosystemApi.listResources()
      .then(preloadResourcesByPriority)
      .catch(() => {
        // The shared API is optional while developing the visual layer locally.
      })
  }, [])

  return null
}
