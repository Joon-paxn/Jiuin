import { apiConfig } from '../../services/api/config'

export type BackgroundResource = {
  url: string
  image: HTMLImageElement
}

type BackgroundResponse = {
  code?: number
  data?: {
    url?: unknown
  }
}

const randomBackgroundEndpoint = `${apiConfig.baseUrl}/api/v1/background/random`
const maxAttempts = 3
const imageLoadTimeout = 12_000

let backgroundLoadPromise: Promise<BackgroundResource> | undefined

function loadImage(url: string) {
  return new Promise<HTMLImageElement>((resolve, reject) => {
    const image = new Image()
    let settled = false
    const timeout = window.setTimeout(() => finish(new Error('Background image loading timed out.')), imageLoadTimeout)

    const finish = (error?: Error) => {
      if (settled) return
      settled = true
      window.clearTimeout(timeout)
      image.onload = null
      image.onerror = null
      if (error) {
        reject(error)
        return
      }
      resolve(image)
    }

    image.decoding = 'async'
    image.onload = () => {
      if (!image.naturalWidth || !image.naturalHeight) {
        finish(new Error('Background image has no dimensions.'))
        return
      }
      image.decode().then(() => finish(), () => finish())
    }
    image.onerror = () => finish(new Error('Background image could not be loaded.'))
    image.src = url
  })
}

async function requestRandomBackgroundURL() {
  const response = await fetch(randomBackgroundEndpoint, {
    headers: { Accept: 'application/json' },
    credentials: 'same-origin',
  })
  if (!response.ok) {
    throw new Error(`Background API returned ${response.status}.`)
  }

  const body = await response.json() as BackgroundResponse
  const url = body.data?.url
  if (body.code !== 200 || typeof url !== 'string' || !url) {
    throw new Error('Background API returned an invalid URL.')
  }
  return url
}

async function requestAndLoadBackground() {
  let lastError: unknown
  for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
    try {
      const url = await requestRandomBackgroundURL()
      const image = await loadImage(url)
      return { url, image }
    } catch (error) {
      lastError = error
    }
  }
  throw lastError instanceof Error ? lastError : new Error('Background resource loading failed.')
}

// The page owns one promise for both the Loading scene and the visible
// background. Consumers never receive the CDN pool and cannot start another
// random selection during the same page lifetime.
export function loadCurrentBackground(): Promise<BackgroundResource> {
  if (!backgroundLoadPromise) {
    backgroundLoadPromise = requestAndLoadBackground().catch((error: unknown) => {
      backgroundLoadPromise = undefined
      throw error
    })
  }
  return backgroundLoadPromise
}
