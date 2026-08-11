import { useEffect, useId, useRef, useState } from 'react'
import type { RefObject } from 'react'
import { classNames } from '../../utils/classNames'
import { live2dConfig } from './config'
import { useLive2DController } from './controller'
import type { Live2DConfig } from './types'

type Live2DProps = {
  config?: Live2DConfig
}

type IdleWindow = Window & {
  requestIdleCallback?: (callback: () => void, options?: { timeout: number }) => number
  cancelIdleCallback?: (handle: number) => void
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

export function Live2D({ config = live2dConfig }: Live2DProps) {
  const floatingRef = useRef<HTMLElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const expressionTriggerRef = useRef<HTMLButtonElement>(null)
  const modelTriggerRef = useRef<HTMLButtonElement>(null)
  const previousOpenMenuRef = useRef<string | null>(null)
  const menuId = useId()
  const canLoad = useDeferredLoad(config.enabled, config.lazyLoad, config.loadDelayMs)
  const controller = useLive2DController({
    config,
    canLoad,
    containerRef,
    floatingRef,
  })
  const { state } = controller
  useFooterAvoidance(floatingRef)

  useEffect(() => {
    const previouslyOpenMenu = previousOpenMenuRef.current
    previousOpenMenuRef.current = state.openMenu
    if (!previouslyOpenMenu || state.openMenu || !state.restoreMenuFocus) {
      return
    }

    const activeElement = document.activeElement
    if (!(activeElement instanceof HTMLElement) || !activeElement.closest('.live2d-floating__menu')) {
      return
    }

    const trigger = previouslyOpenMenu === 'expressions'
      ? expressionTriggerRef.current
      : modelTriggerRef.current
    const focusTimer = window.setTimeout(() => trigger?.focus(), 180)
    return () => window.clearTimeout(focusTimer)
  }, [state.openMenu])

  if (!config.enabled) {
    return null
  }

  const renderedMenu = state.renderedMenu
  const menuIsOpen = state.openMenu === renderedMenu

  return (
    <aside
      ref={floatingRef}
      className={classNames(
        'live2d-floating',
        `live2d-floating--${config.position}`,
        `is-${state.status}`,
      )}
      aria-label={`${state.currentModel.displayName} Live2D 角色`}
      data-live2d-format={state.format}
      data-live2d-status={state.status}
      data-live2d-model={state.currentModel.id}
      data-live2d-model-loading={state.isModelLoading}
    >
      <div className="live2d-floating__frame">
        <div ref={containerRef} className="live2d-canvas" />
        {state.status === 'loading' && (
          <span className="live2d-floating__status" role="status">正在唤醒角色…</span>
        )}
        {state.isModelLoading && state.status === 'ready' && (
          <span className="live2d-floating__loading" role="status">模型加载中…</span>
        )}
        {state.status === 'error' && (
          <div className="live2d-floating__error" role="alert">
            <strong>{state.loadError?.code ?? 'LIVE2D_MODEL_LOAD_FAILED'}</strong>
            <span>{state.loadError?.message ?? 'Live2D 初始化失败。'}</span>
            <button type="button" onClick={controller.retry} disabled={state.isModelLoading}>
              重试
            </button>
          </div>
        )}
      </div>

      <div
        className="live2d-floating__controls"
        data-menu={state.openMenu ?? ''}
        data-menu-open={controller.isMenuOpen}
        role="group"
        aria-label="Live2D 控制"
      >
        <button
          ref={expressionTriggerRef}
          className="live2d-floating__control"
          type="button"
          aria-controls={renderedMenu ? menuId : undefined}
          aria-expanded={state.openMenu === 'expressions'}
          onClick={() => controller.toggleMenu('expressions')}
          disabled={state.isModelLoading}
        >
          表情
        </button>
        <button
          ref={modelTriggerRef}
          className="live2d-floating__control"
          type="button"
          aria-controls={renderedMenu ? menuId : undefined}
          aria-expanded={state.openMenu === 'models'}
          onClick={() => controller.toggleMenu('models')}
          disabled={state.isModelLoading}
        >
          模型
        </button>

        {renderedMenu && (
          <section
            id={menuId}
            className="live2d-floating__menu"
            data-open={menuIsOpen}
            aria-hidden={!menuIsOpen}
            aria-label={renderedMenu === 'expressions' ? '可用表情' : '可用模型'}
          >
            <strong className="live2d-floating__menu-title">
              {renderedMenu === 'expressions' ? '表情' : '模型'}
            </strong>
            {renderedMenu === 'expressions' ? (
              state.availableExpressions.length > 0 ? (
                <div className="live2d-floating__menu-list">
                  {state.availableExpressions.map((expression) => (
                    <button
                      key={expression.id}
                      type="button"
                      className="live2d-floating__menu-item"
                      data-selected={state.currentExpression === expression.id}
                      onClick={() => void controller.selectExpression(expression)}
                      tabIndex={menuIsOpen ? 0 : -1}
                    >
                      {expression.label}
                    </button>
                  ))}
                </div>
              ) : (
                <p className="live2d-floating__menu-empty">当前模型没有可用表情</p>
              )
            ) : (
              <div className="live2d-floating__menu-list">
                {state.availableModels.map((model) => (
                  <button
                    key={model.id}
                    type="button"
                    className="live2d-floating__menu-item"
                    data-selected={state.currentModel.id === model.id}
                    onClick={() => controller.selectModel(model)}
                    disabled={state.isModelLoading}
                    tabIndex={menuIsOpen ? 0 : -1}
                  >
                    {model.displayName}
                  </button>
                ))}
              </div>
            )}
          </section>
        )}
      </div>

      {state.feedback && (
        <span
          className="live2d-floating__feedback"
          data-tone={state.feedback.tone}
          role="status"
        >
          {state.feedback.message}
        </span>
      )}
    </aside>
  )
}

export const Live2DFloating = Live2D
export { live2dConfig, live2dModels } from './config'
export { detectModelFormat, loadLive2DModel } from './loader'
export type {
  Live2DConfig,
  Live2DErrorCode,
  Live2DExpressionOption,
  Live2DMenu,
  Live2DModelFormat,
  Live2DModelRegistration,
  Live2DPosition,
  Live2DStatus,
} from './types'
