import type { ResourceDescriptor } from '../api/types'

type IdleWindow = Window & {
  requestIdleCallback?: (callback: () => void, options?: { timeout: number }) => number
}

function schedule(callback: () => void, timeout: number) {
  const idleWindow = window as IdleWindow
  if (idleWindow.requestIdleCallback) {
    idleWindow.requestIdleCallback(callback, { timeout })
    return
  }
  window.setTimeout(callback, Math.min(timeout, 250))
}

function preload(resource: ResourceDescriptor) {
  void fetch(resource.url, { credentials: 'same-origin' }).catch(() => {
    // Resource caching is opportunistic; normal page loading remains unaffected.
  })
}

/**
 * Level 1/2 resources can be fetched after first paint. Level 3 is deferred
 * to an idle period, while large Level 4 media stays demand-loaded.
 */
export function preloadResourcesByPriority(resources: ResourceDescriptor[]) {
  const sorted = [...resources].sort((left, right) => left.priority - right.priority)

  for (const resource of sorted) {
    if (resource.priority <= 2) {
      schedule(() => preload(resource), 750)
    } else if (resource.priority === 3) {
      schedule(() => preload(resource), 4_000)
    }
  }
}
