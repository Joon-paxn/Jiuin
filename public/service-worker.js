// Bump the namespace when cache eligibility changes so an older worker cannot
// retain resources under a policy that no longer applies.
const CACHE_PREFIX = 'jiuin-ecosystem-v5'
const STATIC_CACHE = `${CACHE_PREFIX}-static`
const CONFIG_CACHE = `${CACHE_PREFIX}-config`
const MEDIA_CACHE = `${CACHE_PREFIX}-media`
const APP_BASE_PATH = new URL(self.registration.scope).pathname.replace(/\/$/, '')
const LIVE2D_PATH = `${APP_BASE_PATH}/live2d/`
const LIVE2D_RUNTIME_PATH = `${LIVE2D_PATH}runtime/`
// The Go API deliberately exposes public music at the origin-level route,
// independent of an optional Vite /jiuin/ asset base.
const PUBLIC_MUSIC_PATH = '/media/music/'

const CONFIG_ENDPOINTS = new Set([
  '/api/v1/site',
  '/api/v1/site/info',
  '/api/v1/site/copyright',
  '/api/v1/status',
  '/api/v1/links',
  '/api/v1/resources',
  '/api/v1/music',
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

  if (
    request.method !== 'GET'
    || request.headers.has('range')
    || request.headers.has('authorization')
    || request.cache === 'no-store'
  ) {
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

  if (url.pathname.startsWith(LIVE2D_PATH) && /\.model(?:3)?\.json$/.test(url.pathname)) {
    event.respondWith(networkFirst(request, CONFIG_CACHE, isNonHtmlResponse))
    return
  }

  if (url.pathname.startsWith(LIVE2D_RUNTIME_PATH)) {
    event.respondWith(networkFirst(request, STATIC_CACHE, isJavaScriptResponse))
    return
  }

  if (url.pathname.startsWith(LIVE2D_PATH)) {
    // A Runtime/model request must never stay pinned to a stale successful
    // response after deployment. Network failures may still fall back offline.
    event.respondWith(networkFirst(request, MEDIA_CACHE, isNonHtmlResponse))
    return
  }

  // Only the current public, opaque music endpoint may use media caching.
  // Future signed/private media commonly carries query credentials and must
  // remain network-only rather than being retained in Cache Storage.
  if (request.destination === 'audio') {
    if (url.pathname.startsWith(PUBLIC_MUSIC_PATH) && !url.search) {
      event.respondWith(cacheFirst(request, MEDIA_CACHE))
    }
    return
  }

  if (request.destination === 'image' || url.pathname.startsWith('/models/')) {
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

function isNonHtmlResponse(response) {
  return response.ok && !response.headers.get('content-type')?.toLowerCase().includes('text/html')
}

function isJavaScriptResponse(response) {
  const contentType = response.headers.get('content-type')?.toLowerCase() ?? ''
  return response.ok && /^(?:application|text)\/(?:javascript|ecmascript)(?:;|$)|^application\/x-javascript(?:;|$)/
    .test(contentType)
}

async function networkFirst(request, cacheName, shouldCache = (response) => response.ok) {
  const cache = await caches.open(cacheName)
  try {
    const response = await fetch(request)
    if (shouldCache(response)) {
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
