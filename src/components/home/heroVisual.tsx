import { createContext, useContext, useEffect, useMemo, useState, type PropsWithChildren } from 'react'

export type HeroVisualState = 'idle' | 'loading-low' | 'low-ready' | 'loading-hd' | 'hd-ready' | 'error'

type HeroVisualAsset = {
  low: string
  hd: string
}

type HeroVisualContextValue = HeroVisualAsset & {
  state: HeroVisualState
  lowReady: boolean
  hdReady: boolean
}

const heroVisuals: readonly HeroVisualAsset[] = [
  {
    low: '/hero-visual/Hero-Visual1.png',
    hd: '/hero-visual/Hero-Visual1HD.png',
  },
  {
    low: '/hero-visual/Hero-Visual2.png',
    hd: '/hero-visual/Hero-Visual2HD.png',
  },
]

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

// Selection is module-scoped so remounting HeroSection cannot choose or request
// a different image during the current page session.
const selectedHeroVisual = heroVisuals[randomIndex(heroVisuals.length)]
let lowPromise: Promise<boolean> | undefined
let hdPromise: Promise<boolean> | undefined

function preloadImage(source: string, timeoutMs: number) {
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

export function loadHeroVisualLow() {
  lowPromise ??= preloadImage(selectedHeroVisual.low, 8_000)
  return lowPromise
}

function loadHeroVisualHD() {
  hdPromise ??= preloadImage(selectedHeroVisual.hd, 12_000)
  return hdPromise
}

const HeroVisualContext = createContext<HeroVisualContextValue | null>(null)

type HeroVisualProviderProps = PropsWithChildren<{
  pageReady: boolean
}>

export function HeroVisualProvider({ pageReady, children }: HeroVisualProviderProps) {
  const [state, setState] = useState<HeroVisualState>('idle')
  const [lowReady, setLowReady] = useState(false)
  const [hdReady, setHdReady] = useState(false)

  useEffect(() => {
    let cancelled = false
    setState('loading-low')
    void loadHeroVisualLow().then((ready) => {
      if (cancelled) return
      setLowReady(ready)
      setState(ready ? 'low-ready' : 'error')
    })
    return () => { cancelled = true }
  }, [])

  useEffect(() => {
    if (!pageReady || !lowReady) return
    let cancelled = false
    let idleHandle: number | undefined
    let timeoutHandle: number | undefined

    const startHD = () => {
      if (cancelled) return
      setState('loading-hd')
      void loadHeroVisualHD().then((ready) => {
        if (cancelled) return
        setHdReady(ready)
        setState(ready ? 'hd-ready' : 'error')
      })
    }

    const browserWindow = window as Window & {
      requestIdleCallback?: (callback: IdleRequestCallback, options?: IdleRequestOptions) => number
      cancelIdleCallback?: (handle: number) => void
    }
    if (browserWindow.requestIdleCallback) {
      idleHandle = browserWindow.requestIdleCallback(startHD, { timeout: 1_500 })
    } else {
      timeoutHandle = window.setTimeout(startHD, 120)
    }

    return () => {
      cancelled = true
      if (idleHandle !== undefined) browserWindow.cancelIdleCallback?.(idleHandle)
      if (timeoutHandle !== undefined) window.clearTimeout(timeoutHandle)
    }
  }, [lowReady, pageReady])

  const value = useMemo(() => ({
    ...selectedHeroVisual,
    state,
    lowReady,
    hdReady,
  }), [hdReady, lowReady, state])

  return <HeroVisualContext.Provider value={value}>{children}</HeroVisualContext.Provider>
}

export function useHeroVisual() {
  const context = useContext(HeroVisualContext)
  if (!context) {
    throw new Error('useHeroVisual must be used within HeroVisualProvider')
  }
  return context
}
