import { useEffect, useRef, useState } from 'react'
import type { Live2DModelConfig, Live2DStatus } from './live2d.types'

declare global {
  interface Window {
    PIXI?: typeof import('pixi.js')
    Live2DCubismCore?: unknown
  }
}

type Live2DCanvasProps = {
  config: Live2DModelConfig
  onStatusChange?: (status: Live2DStatus) => void
}

let coreLoadPromise: Promise<void> | undefined

function ensureCubismCore(source: string) {
  if (window.Live2DCubismCore) {
    return Promise.resolve()
  }

  if (!coreLoadPromise) {
    coreLoadPromise = new Promise<void>((resolve, reject) => {
      const existing = document.querySelector<HTMLScriptElement>('script[data-jiuin-live2d-core]')
      const script = existing ?? document.createElement('script')

      const complete = () => window.Live2DCubismCore
        ? resolve()
        : reject(new Error('Cubism Core did not expose its browser runtime.'))

      script.addEventListener('load', complete, { once: true })
      script.addEventListener('error', () => reject(new Error('Unable to load the Live2D runtime.')), { once: true })

      if (!existing) {
        script.async = true
        script.src = source
        script.dataset.jiuinLive2dCore = 'true'
        document.head.append(script)
      }
    }).catch((error: unknown) => {
      coreLoadPromise = undefined
      throw error
    })
  }

  return coreLoadPromise
}

function hasWebGLSupport() {
  const canvas = document.createElement('canvas')
  return Boolean(canvas.getContext('webgl') || canvas.getContext('experimental-webgl'))
}

/** Lazy, self-contained Cubism canvas. It creates no renderer until mounted. */
export function Live2DCanvas({ config, onStatusChange }: Live2DCanvasProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [status, setStatus] = useState<Live2DStatus>('idle')

  useEffect(() => {
    if (!config.enabled || !containerRef.current) {
      return
    }

    if (!hasWebGLSupport()) {
      setStatus('error')
      onStatusChange?.('error')
      return
    }

    let disposed = false
    let disposeRenderer: (() => void) | undefined
    setStatus('loading')
    onStatusChange?.('loading')

    const load = async () => {
      try {
        await ensureCubismCore(config.coreScriptUrl)
        const [PIXI, { Live2DModel }] = await Promise.all([
          import('pixi.js'),
          import('pixi-live2d-display/cubism4'),
        ])

        if (disposed || !containerRef.current) {
          return
        }

        window.PIXI = PIXI
        const app = new PIXI.Application({
          resizeTo: containerRef.current,
          autoDensity: true,
          antialias: true,
          backgroundAlpha: 0,
          resolution: Math.min(window.devicePixelRatio || 1, 2),
        })
        containerRef.current.replaceChildren(app.view)

        const model = await Live2DModel.from(config.modelPath, { autoInteract: false })
        if (disposed || !containerRef.current) {
          model.destroy()
          app.destroy(true, { children: true, texture: true, baseTexture: true })
          return
        }

        app.stage.addChild(model)

        const fitModel = () => {
          const { width, height } = app.renderer
          const scale = Math.min(width / model.width, height / model.height) * 0.92
          model.scale.set(scale)
          model.x = width / 2
          model.y = height
        }

        fitModel()
        const observer = new ResizeObserver(fitModel)
        observer.observe(containerRef.current)

        const view = app.view as HTMLCanvasElement
        const getCanvasPoint = (event: PointerEvent) => {
          const bounds = view.getBoundingClientRect()
          return {
            x: (event.clientX - bounds.left) * (app.renderer.width / bounds.width),
            y: (event.clientY - bounds.top) * (app.renderer.height / bounds.height),
          }
        }
        const focus = (event: PointerEvent) => {
          const point = getCanvasPoint(event)
          model.focus(point.x, point.y)
        }
        const interact = (event: PointerEvent) => {
          const point = getCanvasPoint(event)
          model.tap(point.x, point.y)
          model.expression()
        }

        view.addEventListener('pointermove', focus)
        view.addEventListener('pointerup', interact)
        disposeRenderer = () => {
          observer.disconnect()
          view.removeEventListener('pointermove', focus)
          view.removeEventListener('pointerup', interact)
          app.destroy(true, { children: true, texture: true, baseTexture: true })
        }

        setStatus('ready')
        onStatusChange?.('ready')
      } catch (error) {
        if (!disposed) {
          console.error('Live2D model failed to load.', error)
          setStatus('error')
          onStatusChange?.('error')
        }
      }
    }

    void load()

    return () => {
      disposed = true
      disposeRenderer?.()
    }
  }, [config, onStatusChange])

  return <div ref={containerRef} className={`live2d-canvas is-${status}`} aria-hidden={status !== 'error'} />
}
