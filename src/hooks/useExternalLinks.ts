import { useEffect, useState } from 'react'
import { ecosystemApi } from '../services/api/ecosystem'
import type { ExternalLink } from '../services/api/types'

export function useExternalLinks() {
  const [links, setLinks] = useState<ExternalLink[]>([])

  useEffect(() => {
    let ignored = false
    void ecosystemApi.listLinks()
      .then((nextLinks) => {
        if (!ignored) {
          setLinks(nextLinks)
        }
      })
      .catch(() => {
        // Links are optional site configuration and safely remain empty offline.
      })

    return () => {
      ignored = true
    }
  }, [])

  return links
}
