const CACHE_PREFIX = 'jiuin-ecosystem-v2'
const STATIC_CACHE = `${CACHE_PREFIX}-static`
const CONFIG_CACHE = `${CACHE_PREFIX}-config`
const MEDIA_CACHE = `${CACHE_PREFIX}-media`

const CONFIG_ENDPOINTS = new Set([
  '/api/v1/site',
  '/api/v1/site/info',
  '/api/v1/site/copyright',
  '/api/v1/status',
  '/api/v1/links',
  '/api/v1/resources',
  '/api/v1/music/list',
])

self.addEventListener('install', () => {
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil((async () => {
    const keys = await caches.keys()
    await Promise.all(keys
      .filter((key) => key.startsWith('jiuin-') && !key.startsWith(CACHE_PREFIX))
      .map((key) => caches.delete(key)))
    await self.clients.claim()
  })())
})

self.addEventListener('fetch', (event) => {
  const { request } = event
  const url = new URL(request.url)

  if (request.method !== 'GET' || request.headers.has('range')) {
    return
  }

  if (CONFIG_ENDPOINTS.has(url.pathname)) {
    event.respondWith(networkFirst(request, CONFIG_CACHE))
    return
  }

  if (url.origin !== self.location.origin) {
    return
  }

  if (request.mode === 'navigate') {
    event.respondWith(networkFirst(request, STATIC_CACHE))
    return
  }

  if (/\/live2d\/.*\.model(?:3)?\.json$/.test(url.pathname)) {
    event.respondWith(networkFirst(request, CONFIG_CACHE))
    return
  }

  if (
    request.destination === 'audio'
    || request.destination === 'image'
    || url.pathname.startsWith('/models/')
    || url.pathname.startsWith('/live2d/')
  ) {
    event.respondWith(cacheFirst(request, MEDIA_CACHE))
    return
  }

  if (request.destination === 'script' || request.destination === 'style' || request.destination === 'font') {
    event.respondWith(staleWhileRevalidate(request, STATIC_CACHE))
  }
})

async function cacheFirst(request, cacheName) {
  const cache = await caches.open(cacheName)
  const cached = await cache.match(request)
  if (cached) {
    return cached
  }

  const response = await fetch(request)
  if (response.ok) {
    await cache.put(request, response.clone())
  }
  return response
}

async function networkFirst(request, cacheName) {
  const cache = await caches.open(cacheName)
  try {
    const response = await fetch(request)
    if (response.ok) {
      await cache.put(request, response.clone())
    }
    return response
  } catch {
    const cached = await cache.match(request)
    if (cached) {
      return cached
    }
    throw new Error('No cached response is available.')
  }
}

async function staleWhileRevalidate(request, cacheName) {
  const cache = await caches.open(cacheName)
  const cached = await cache.match(request)
  const network = fetch(request)
    .then(async (response) => {
      if (response.ok) {
        await cache.put(request, response.clone())
      }
      return response
    })

  return cached || network
}
