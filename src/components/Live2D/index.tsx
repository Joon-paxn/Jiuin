import { useEffect, useRef, useState } from 'react'
import type { RefObject } from 'react'
import { classNames } from '../../utils/classNames'
import { live2dConfig } from './config'
import { Live2DLoadError, loadLive2DModel, normalizeLive2DError } from './loader'
import type { Live2DConfig, Live2DModelFormat, Live2DStatus } from './types'

type Live2DProps = {
  config?: Live2DConfig
}

type IdleWindow = Window & {
  requestIdleCallback?: (callback: () => void, options?: { timeout: number }) => number
  cancelIdleCallback?: (handle: number) => void
}

function hasWebGLSupport() {
  try {
    const canvas = document.createElement('canvas')
    return Boolean(
      canvas.getContext('webgl2')
      || canvas.getContext('webgl')
      || canvas.getContext('experimental-webgl'),
    )
  } catch {
    return false
  }
}

function useDeferredLoad(enabled: boolean, lazyLoad: boolean, delayMs: number) {
  const [canLoad, setCanLoad] = useState(enabled && !lazyLoad)

  useEffect(() => {
    if (!enabled) {
      setCanLoad(false)
      return
    }

    if (!lazyLoad) {
      setCanLoad(true)
      return
    }

    const idleWindow = window as IdleWindow
    let idleHandle: number | undefined
    let timerHandle: number | undefined

    const schedule = () => {
      timerHandle = window.setTimeout(() => {
        if (idleWindow.requestIdleCallback) {
          idleHandle = idleWindow.requestIdleCallback(() => setCanLoad(true), { timeout: 2_000 })
        } else {
          setCanLoad(true)
        }
      }, delayMs)
    }

    if (document.readyState === 'complete') {
      schedule()
    } else {
      window.addEventListener('load', schedule, { once: true })
    }

    return () => {
      window.removeEventListener('load', schedule)
      if (timerHandle !== undefined) {
        window.clearTimeout(timerHandle)
      }
      if (idleHandle !== undefined) {
        idleWindow.cancelIdleCallback?.(idleHandle)
      }
    }
  }, [delayMs, enabled, lazyLoad])

  return canLoad
}

function useFooterAvoidance(floatingRef: RefObject<HTMLElement | null>) {
  useEffect(() => {
    const floating = floatingRef.current
    const footer = document.querySelector<HTMLElement>('.site-footer')
    if (!floating || !footer) {
      return
    }

    let frameHandle = 0
    const updateOffset = () => {
      frameHandle = 0
      const floatingRect = floating.getBoundingClientRect()
      const footerRect = footer.getBoundingClientRect()
      const currentOffset = Number.parseFloat(
        floating.style.getPropertyValue('--live2d-footer-offset'),
      ) || 0
      const gap = 16

      if (footerRect.top >= window.innerHeight || footerRect.bottom <= 0) {
        floating.style.setProperty('--live2d-footer-offset', '0px')
        return
      }

      const baseTop = floatingRect.top + currentOffset
      const baseBottom = floatingRect.bottom + currentOffset
      const targetBottom = Math.max(gap, footerRect.top - gap)
      const requiredOffset = Math.max(0, baseBottom - targetBottom)
      const maximumOffset = Math.max(0, baseTop - gap)
      floating.style.setProperty(
        '--live2d-footer-offset',
        `${Math.min(requiredOffset, maximumOffset)}px`,
      )
    }
    const requestUpdate = () => {
      if (!frameHandle) {
        frameHandle = window.requestAnimationFrame(updateOffset)
      }
    }

    const observer = typeof ResizeObserver === 'undefined'
      ? undefined
      : new ResizeObserver(requestUpdate)
    observer?.observe(footer)
    observer?.observe(floating)
    window.addEventListener('scroll', requestUpdate, { passive: true })
    window.addEventListener('resize', requestUpdate)
    requestUpdate()

    return () => {
      observer?.disconnect()
      window.removeEventListener('scroll', requestUpdate)
      window.removeEventListener('resize', requestUpdate)
      if (frameHandle) {
        window.cancelAnimationFrame(frameHandle)
      }
    }
  }, [floatingRef])
}

function reportLoadError(error: Live2DLoadError, config: Live2DConfig) {
  console.groupCollapsed(`[Live2D] 加载失败：${error.code}`)
  console.error(error.message, error.cause ?? error)
  console.info('模型路径：', config.modelPath)
  console.info('错误上下文：', error.context)
  console.info('当前环境：', {
    mode: import.meta.env.MODE,
    basePath: import.meta.env.BASE_URL,
    pageUrl: window.location.href,
  })
  console.info('建议检查：HTTP 状态、manifest 引用、MOC 版本、Core 版本及浏览器 WebGL 支持。')
  console.groupEnd()
}

export function Live2D({ config = live2dConfig }: Live2DProps) {
  const floatingRef = useRef<HTMLElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const [status, setStatus] = useState<Live2DStatus>('idle')
  const [format, setFormat] = useState<Live2DModelFormat>()
  const [loadError, setLoadError] = useState<Live2DLoadError>()
  const [retryNonce, setRetryNonce] = useState(0)
  const canLoad = useDeferredLoad(config.enabled, config.lazyLoad, config.loadDelayMs)
  useFooterAvoidance(floatingRef)

  useEffect(() => {
    const container = containerRef.current
    if (!config.enabled || !canLoad || !container) {
      return
    }

    let disposed = false
    let disposeRenderer: (() => void) | undefined
    let destroyUnattachedModel: (() => void) | undefined
    setLoadError(undefined)
    setStatus('loading')

    const load = async () => {
      try {
        if (!hasWebGLSupport()) {
          throw new Live2DLoadError(
            'WEBGL_UNAVAILABLE',
            '当前浏览器无法创建 WebGL 上下文。',
            { modelPath: config.modelPath },
          )
        }

        const loaded = await loadLive2DModel(config)
        destroyUnattachedModel = () => loaded.model.destroy()
        if (disposed) {
          destroyUnattachedModel()
          destroyUnattachedModel = undefined
          return
        }

        const { model, pixi } = loaded
        const bounds = container.getBoundingClientRect()
        const app = new pixi.Application({
          width: Math.max(1, bounds.width),
          height: Math.max(1, bounds.height),
          autoDensity: true,
          antialias: true,
          backgroundAlpha: 0,
          resolution: Math.min(window.devicePixelRatio || 1, 2),
        })
        let modelAttached = false
        let appDestroyed = false
        const destroyApp = () => {
          if (appDestroyed) {
            return
          }
          appDestroyed = true
          if (!modelAttached) {
            destroyUnattachedModel?.()
          }
          destroyUnattachedModel = undefined
          app.destroy(true, { children: true, texture: true, baseTexture: true })
        }
        disposeRenderer = destroyApp

        container.replaceChildren(app.view)
        app.stage.addChild(model)
        modelAttached = true

        model.scale.set(1)
        model.anchor.set(0.5, 1)
        const naturalWidth = model.width
        const naturalHeight = model.height
        if (
          !Number.isFinite(naturalWidth)
          || !Number.isFinite(naturalHeight)
          || naturalWidth <= 0
          || naturalHeight <= 0
        ) {
          throw new Live2DLoadError(
            'RENDER_SIZE_INVALID',
            'Live2D 模型返回了无效的渲染尺寸。',
            { modelPath: config.modelPath, modelFormat: loaded.format },
          )
        }

        const fitModel = (width: number, height: number) => {
          app.renderer.resize(Math.max(1, width), Math.max(1, height))
          const viewport = app.screen
          const fitScale = Math.min(
            viewport.width / naturalWidth,
            viewport.height / naturalHeight,
          ) * 0.92 * config.scale

          model.scale.set(fitScale)
          model.position.set(viewport.width / 2, viewport.height)
        }

        fitModel(bounds.width, bounds.height)
        const resize = () => {
          const rect = container.getBoundingClientRect()
          fitModel(rect.width, rect.height)
        }
        const observer = typeof ResizeObserver === 'undefined'
          ? undefined
          : new ResizeObserver(([entry]) => {
              if (entry) {
                fitModel(entry.contentRect.width, entry.contentRect.height)
              }
            })
        observer?.observe(container)
        if (!observer) {
          window.addEventListener('resize', resize)
        }

        const view = app.view as HTMLCanvasElement
        const getCanvasPoint = (event: PointerEvent) => {
          const canvasBounds = view.getBoundingClientRect()
          return {
            x: (event.clientX - canvasBounds.left) * (app.screen.width / canvasBounds.width),
            y: (event.clientY - canvasBounds.top) * (app.screen.height / canvasBounds.height),
          }
        }
        const focus = (event: PointerEvent) => {
          const point = getCanvasPoint(event)
          model.focus(point.x, point.y)
        }
        const interact = (event: PointerEvent) => {
          const point = getCanvasPoint(event)
          model.tap(point.x, point.y)
          void model.expression().catch((error: unknown) => {
            console.warn('[Live2D] 表情切换失败。', error)
          })
        }

        view.addEventListener('pointermove', focus)
        view.addEventListener('pointerup', interact)
        disposeRenderer = () => {
          observer?.disconnect()
          window.removeEventListener('resize', resize)
          view.removeEventListener('pointermove', focus)
          view.removeEventListener('pointerup', interact)
          destroyApp()
        }

        console.info('[Live2D] 模型加载完成。', {
          modelPath: config.modelPath,
          format: loaded.format,
          mocVersion: loaded.mocVersion,
          supportedMocVersion: loaded.supportedMocVersion,
          runtime: loaded.runtime,
          resources: loaded.resources.map((resource) => resource.url),
        })
        setFormat(loaded.format)
        setStatus('ready')
      } catch (error) {
        try {
          disposeRenderer?.()
        } catch (disposeError) {
          console.warn('[Live2D] 清理失败的渲染器时出现异常。', disposeError)
        }
        disposeRenderer = undefined
        try {
          destroyUnattachedModel?.()
        } catch (disposeError) {
          console.warn('[Live2D] 清理失败的模型时出现异常。', disposeError)
        }
        destroyUnattachedModel = undefined
        if (!disposed) {
          const normalized = normalizeLive2DError(error, config.modelPath)
          reportLoadError(normalized, config)
          setLoadError(normalized)
          setStatus('error')
        }
      }
    }

    void load()

    return () => {
      disposed = true
      disposeRenderer?.()
    }
  }, [canLoad, config, retryNonce])

  if (!config.enabled) {
    return null
  }

  return (
    <aside
      ref={floatingRef}
      className={classNames(
        'live2d-floating',
        `live2d-floating--${config.position}`,
        `is-${status}`,
      )}
      aria-label={`${config.displayName} Live2D 角色`}
      data-live2d-format={format}
      data-live2d-status={status}
    >
      <div className="live2d-floating__frame">
        <div ref={containerRef} className="live2d-canvas" />
        {status === 'loading' && (
          <span className="live2d-floating__status" role="status">正在唤醒角色…</span>
        )}
        {status === 'error' && (
          <div className="live2d-floating__error" role="alert">
            <strong>{loadError?.code ?? 'LIVE2D_MODEL_LOAD_FAILED'}</strong>
            <span>{loadError?.message ?? 'Live2D 初始化失败。'}</span>
            <button type="button" onClick={() => setRetryNonce((value) => value + 1)}>
              重试
            </button>
          </div>
        )}
      </div>
    </aside>
  )
}

export const Live2DFloating = Live2D
export { live2dConfig, live2dModels } from './config'
export { detectModelFormat, loadLive2DModel } from './loader'
export type {
  Live2DConfig,
  Live2DErrorCode,
  Live2DModelFormat,
  Live2DPosition,
  Live2DStatus,
} from './types'
