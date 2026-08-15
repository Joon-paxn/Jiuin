import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useOptionalBackground } from '../../../components/background'

const webLoadingAsset = '/loading/web_Loading.png'
const loadingAssets = Array.from({ length: 12 }, (_, index) => `/loading/Loading${index + 1}.png`)
const MAX_RETRIES = 3

export type LoadingResourceState = {
  background: boolean
  loadingAssets: boolean
  coreUI: boolean
  criticalImages: boolean
  live2d: boolean
  music: boolean
}

export type LoadingResourceManager = {
  resources: LoadingResourceState
  artwork: string
  allCriticalResourcesLoaded: boolean
  error: string | null
  retry: () => void
}

function randomIndex(maximum: number) {
  if (maximum <= 1) return 0
  const cryptoApi = globalThis.crypto
  if (cryptoApi?.getRandomValues) {
    const buffer = new Uint32Array(1)
    const limit = 0x1_0000_0000 - (0x1_0000_0000 % maximum)
    do cryptoApi.getRandomValues(buffer); while (buffer[0] >= limit)
    return buffer[0] % maximum
  }
  return Math.floor(Math.random() * maximum)
}

function preloadImage(source: string, timeoutMs = 3_000) {
  return new Promise<boolean>((resolve) => {
    const image = new Image()
    let settled = false
    const finish = (loaded: boolean) => {
      if (settled) return
      settled = true
      window.clearTimeout(timeout)
      image.onload = null
      image.onerror = null
      resolve(loaded)
    }
    const timeout = window.setTimeout(() => finish(false), timeoutMs)
    image.decoding = 'async'
    image.onload = () => {
      if (!image.naturalWidth || !image.naturalHeight) return finish(false)
      image.decode().then(() => finish(true), () => finish(true))
    }
    image.onerror = () => finish(false)
    image.src = source
  })
}

async function loadArtwork(preferred: string) {
  // The artwork is selected once when the manager is created. Retry that
  // same URL if the request is transient; do not walk the other 11 images,
  // which would unnecessarily download every loading asset.
  for (let attempt = 0; attempt < MAX_RETRIES; attempt += 1) {
    if (await preloadImage(preferred)) return preferred
  }
  return null
}

export function useLoadingResourceManager(enabled: boolean): LoadingResourceManager {
  const backgroundContext = useOptionalBackground()
  const [artwork, setArtwork] = useState(() => loadingAssets[randomIndex(loadingAssets.length)])
  const artworkRef = useRef(artwork)
  const [resources, setResources] = useState<LoadingResourceState>({
    background: false,
    loadingAssets: false,
    coreUI: true,
    criticalImages: true,
    live2d: true,
    music: true,
  })
  const [error, setError] = useState<string | null>(null)
  const [retryToken, setRetryToken] = useState(0)

  useEffect(() => {
    if (!enabled) return
    let cancelled = false
    setError(null)
    setResources((current) => ({ ...current, loadingAssets: false }))
    void (async () => {
      let webReady = false
      for (let attempt = 0; attempt < MAX_RETRIES && !webReady; attempt += 1) {
        webReady = await preloadImage(webLoadingAsset)
      }
    const selected = await loadArtwork(artworkRef.current)
      if (cancelled) return
      if (!webReady || !selected) {
        setError('部分 Loading 资源加载失败')
        return
      }
      setArtwork(selected)
      artworkRef.current = selected
      setResources((current) => ({ ...current, loadingAssets: true }))
    })()
    return () => { cancelled = true }
  }, [enabled, retryToken])

  useEffect(() => {
    if (!enabled) return
    if (backgroundContext?.background.image) {
      setResources((current) => ({ ...current, background: true }))
      return
    }
    const timeout = window.setTimeout(() => {
      // The CDN background is decorative. Keep the gradient fallback and let
      // the page enter even when the image CDN is temporarily unavailable.
      setResources((current) => ({ ...current, background: true }))
    }, 12_000)
    return () => window.clearTimeout(timeout)
  }, [backgroundContext?.background.image, enabled, retryToken])

  const allCriticalResourcesLoaded = useMemo(
    () => resources.loadingAssets && resources.coreUI && resources.criticalImages && !error,
    [error, resources],
  )

  const retry = useCallback(() => {
    setResources((current) => ({ ...current, background: Boolean(backgroundContext?.background.image), loadingAssets: false }))
    setRetryToken((token) => token + 1)
  }, [backgroundContext?.background.image])

  return { resources, artwork, allCriticalResourcesLoaded, error, retry }
}
