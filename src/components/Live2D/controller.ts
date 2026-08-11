import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { RefObject } from 'react'
import type { Live2DModel } from 'pixi-live2d-display'
import { Live2DLoadError, loadLive2DModel, normalizeLive2DError } from './loader'
import type {
  Live2DConfig,
  Live2DExpressionOption,
  Live2DMenu,
  Live2DModelFormat,
  Live2DModelRegistration,
  Live2DStatus,
  LoadedLive2DModel,
} from './types'

type UnknownRecord = Record<string, unknown>

type Live2DFeedback = {
  tone: 'info' | 'error'
  message: string
}

export type Live2DControllerState = {
  status: Live2DStatus
  format?: Live2DModelFormat
  loadError?: Live2DLoadError
  currentModel: Live2DModelRegistration
  currentExpression: string | null
  availableModels: readonly Live2DModelRegistration[]
  availableExpressions: readonly Live2DExpressionOption[]
  isModelLoading: boolean
  openMenu: Live2DMenu | null
  renderedMenu: Live2DMenu | null
  restoreMenuFocus: boolean
  feedback?: Live2DFeedback
}

type Live2DRenderer = {
  model: Live2DModel
  attach: () => void
  dispose: () => void
}

type UseLive2DControllerOptions = {
  config: Live2DConfig
  canLoad: boolean
  containerRef: RefObject<HTMLDivElement | null>
  floatingRef: RefObject<HTMLElement | null>
}

const MENU_TRANSITION_MS = 180
const FEEDBACK_DURATION_MS = 5_000

function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function readString(value: unknown) {
  return typeof value === 'string' && value.length > 0 ? value : undefined
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

function getAvailableModels(config: Live2DConfig): Live2DModelRegistration[] {
  const registered = Object.values(config.modelRegistry ?? {})
  const models = registered.length > 0
    ? registered
    : [{
        id: 'default',
        displayName: config.displayName,
        modelPath: config.modelPath,
        scale: config.scale,
      }]

  const knownIds = new Set<string>()
  return models.flatMap((model) => {
    if (knownIds.has(model.id)) {
      console.warn(`[Live2D] 忽略重复的模型注册 ID：${model.id}`)
      return []
    }
    knownIds.add(model.id)
    return [{ ...model, scale: model.scale ?? config.scale }]
  })
}

function getInitialModel(
  config: Live2DConfig,
  availableModels: readonly Live2DModelRegistration[],
) {
  return availableModels.find((model) => model.id === config.initialModelId)
    ?? availableModels[0]
}

function getConfigSignature(config: Live2DConfig) {
  const models = Object.values(config.modelRegistry ?? {})
    .map((model) => [model.id, model.displayName, model.modelPath, model.scale ?? ''].join('\u001f'))
    .join('\u001e')

  return [
    config.enabled,
    config.modelPath,
    config.position,
    config.scale,
    config.displayName,
    config.lazyLoad,
    config.loadDelayMs,
    config.runtimePaths.cubism2,
    config.runtimePaths.cubism4,
    config.initialModelId ?? '',
    models,
  ].join('\u001d')
}

export function getLive2DExpressions(
  manifest: Record<string, unknown>,
  format: Live2DModelFormat,
): Live2DExpressionOption[] {
  const fileReferences = isRecord(manifest.FileReferences) ? manifest.FileReferences : undefined
  const rawExpressions = format === 'cubism4'
    ? fileReferences?.Expressions
    : manifest.expressions

  if (!Array.isArray(rawExpressions)) {
    return []
  }

  const seenIds = new Set<string>()
  return rawExpressions.flatMap((entry) => {
    const expression = isRecord(entry) ? entry : undefined
    const id = readString(format === 'cubism4' ? expression?.Name : expression?.name)
    const file = readString(format === 'cubism4' ? expression?.File : expression?.file)

    if (!id || !file || seenIds.has(id)) {
      return []
    }

    seenIds.add(id)
    return [{ id, label: id, file }]
  })
}

function createModelConfig(baseConfig: Live2DConfig, model: Live2DModelRegistration): Live2DConfig {
  return {
    ...baseConfig,
    modelPath: model.modelPath,
    scale: model.scale ?? baseConfig.scale,
    displayName: model.displayName,
  }
}

function createRenderer(
  container: HTMLDivElement,
  loaded: LoadedLive2DModel,
  config: Live2DConfig,
  onRandomExpression: () => void,
): Live2DRenderer {
  const { model, pixi } = loaded
  let app: InstanceType<typeof pixi.Application> | undefined
  let observer: ResizeObserver | undefined
  let resize: (() => void) | undefined
  let view: HTMLCanvasElement | undefined
  let disposed = false

  try {
    const bounds = container.getBoundingClientRect()
    app = new pixi.Application({
      width: Math.max(1, bounds.width),
      height: Math.max(1, bounds.height),
      autoDensity: true,
      antialias: true,
      backgroundAlpha: 0,
      resolution: Math.min(window.devicePixelRatio || 1, 2),
    })

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
      app?.renderer.resize(Math.max(1, width), Math.max(1, height))
      const viewport = app?.screen
      if (!viewport) {
        return
      }

      const fitScale = Math.min(
        viewport.width / naturalWidth,
        viewport.height / naturalHeight,
      ) * 0.92 * config.scale

      model.scale.set(fitScale)
      model.position.set(viewport.width / 2, viewport.height)
    }

    app.stage.addChild(model)
    fitModel(bounds.width, bounds.height)
    app.renderer.render(app.stage)

    resize = () => {
      const rect = container.getBoundingClientRect()
      fitModel(rect.width, rect.height)
    }
    observer = typeof ResizeObserver === 'undefined'
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

    view = app.view as HTMLCanvasElement
    const getCanvasPoint = (event: PointerEvent) => {
      const canvasBounds = view!.getBoundingClientRect()
      if (!canvasBounds.width || !canvasBounds.height) {
        return undefined
      }
      return {
        x: (event.clientX - canvasBounds.left) * (app!.screen.width / canvasBounds.width),
        y: (event.clientY - canvasBounds.top) * (app!.screen.height / canvasBounds.height),
      }
    }
    const focus = (event: PointerEvent) => {
      const point = getCanvasPoint(event)
      if (point) {
        model.focus(point.x, point.y)
      }
    }
    const interact = (event: PointerEvent) => {
      const point = getCanvasPoint(event)
      if (!point) {
        return
      }
      model.tap(point.x, point.y)
      void model.expression()
        .then((applied) => {
          if (applied) {
            onRandomExpression()
          }
        })
        .catch((error: unknown) => {
          console.warn('[Live2D] 表情切换失败。', error)
        })
    }
    view.addEventListener('pointermove', focus)
    view.addEventListener('pointerup', interact)

    const dispose = () => {
      if (disposed) {
        return
      }
      disposed = true
      observer?.disconnect()
      if (resize) {
        window.removeEventListener('resize', resize)
      }
      view?.removeEventListener('pointermove', focus)
      view?.removeEventListener('pointerup', interact)
      app?.destroy(true, { children: true, texture: true, baseTexture: true })
    }

    return {
      model,
      attach: () => {
        container.replaceChildren(view!)
      },
      dispose,
    }
  } catch (error) {
    observer?.disconnect()
    if (resize) {
      window.removeEventListener('resize', resize)
    }
    if (app) {
      app.destroy(true, { children: true, texture: true, baseTexture: true })
    } else {
      model.destroy()
    }
    throw error
  }
}

export function useLive2DController({
  config,
  canLoad,
  containerRef,
  floatingRef,
}: UseLive2DControllerOptions) {
  const configRef = useRef(config)
  configRef.current = config
  const configSignature = getConfigSignature(config)
  const availableModels = useMemo(() => getAvailableModels(config), [configSignature])
  const initialModel = useMemo(
    () => getInitialModel(config, availableModels),
    [availableModels, config.initialModelId],
  )
  const [state, setState] = useState<Live2DControllerState>(() => ({
    status: 'idle',
    currentModel: initialModel,
    currentExpression: null,
    availableModels,
    availableExpressions: [],
    isModelLoading: false,
    openMenu: null,
    renderedMenu: null,
    restoreMenuFocus: false,
  }))
  const rendererRef = useRef<Live2DRenderer | undefined>(undefined)
  const requestIdRef = useRef(0)
  const menuDismissTimerRef = useRef<number | undefined>(undefined)
  const feedbackTimerRef = useRef<number | undefined>(undefined)

  const clearMenuDismissTimer = useCallback(() => {
    if (menuDismissTimerRef.current !== undefined) {
      window.clearTimeout(menuDismissTimerRef.current)
      menuDismissTimerRef.current = undefined
    }
  }, [])

  const closeMenu = useCallback((restoreFocus = false) => {
    clearMenuDismissTimer()
    if (!state.openMenu && !state.renderedMenu) {
      return
    }
    setState((previous) => {
      if (!previous.openMenu && !previous.renderedMenu) {
        return previous
      }
      return { ...previous, openMenu: null, restoreMenuFocus: restoreFocus }
    })
    menuDismissTimerRef.current = window.setTimeout(() => {
      setState((previous) => previous.openMenu
        ? previous
        : { ...previous, renderedMenu: null })
      menuDismissTimerRef.current = undefined
    }, MENU_TRANSITION_MS)
  }, [clearMenuDismissTimer, state.openMenu, state.renderedMenu])

  const toggleMenu = useCallback((menu: Live2DMenu) => {
    if (state.openMenu === menu) {
      closeMenu(true)
      return
    }
    clearMenuDismissTimer()
    setState((previous) => {
      return { ...previous, openMenu: menu, renderedMenu: menu, restoreMenuFocus: false }
    })
  }, [clearMenuDismissTimer, closeMenu, state.openMenu])

  const showFeedback = useCallback((feedback: Live2DFeedback) => {
    if (feedbackTimerRef.current !== undefined) {
      window.clearTimeout(feedbackTimerRef.current)
    }
    setState((previous) => ({ ...previous, feedback }))
    feedbackTimerRef.current = window.setTimeout(() => {
      setState((previous) => ({ ...previous, feedback: undefined }))
      feedbackTimerRef.current = undefined
    }, FEEDBACK_DURATION_MS)
  }, [])

  const loadModel = useCallback(async (
    requestedModel: Live2DModelRegistration,
    options: { initial?: boolean } = {},
  ) => {
    const container = containerRef.current
    if (!container) {
      return
    }

    const requestId = ++requestIdRef.current
    const previousRenderer = rendererRef.current
    const modelConfig = createModelConfig(configRef.current, requestedModel)
    let loaded: LoadedLive2DModel | undefined
    let candidate: Live2DRenderer | undefined

    setState((previous) => ({
      ...previous,
      availableModels,
      isModelLoading: true,
      status: previousRenderer ? 'ready' : 'loading',
      loadError: undefined,
      feedback: options.initial ? undefined : previous.feedback,
    }))

    try {
      if (!hasWebGLSupport()) {
        throw new Live2DLoadError(
          'WEBGL_UNAVAILABLE',
          '当前浏览器无法创建 WebGL 上下文。',
          { modelPath: modelConfig.modelPath },
        )
      }

      loaded = await loadLive2DModel(modelConfig)
      const loadedModel = loaded.model
      const loadedManifest = loaded.manifest
      const loadedFormat = loaded.format
      const loadedMocVersion = loaded.mocVersion
      const loadedSupportedMocVersion = loaded.supportedMocVersion
      const loadedRuntime = loaded.runtime
      const loadedResources = loaded.resources
      const loadedCandidate = loaded
      // `createRenderer` takes ownership of the model and releases it if its
      // own setup fails. Do not let the outer error path destroy it twice.
      loaded = undefined
      candidate = createRenderer(container, loadedCandidate, modelConfig, () => {
        if (rendererRef.current?.model === loadedModel) {
          setState((previous) => ({ ...previous, currentExpression: null }))
        }
      })

      if (requestId !== requestIdRef.current) {
        candidate.dispose()
        return
      }

      candidate.attach()
      rendererRef.current = candidate
      candidate = undefined
      try {
        previousRenderer?.dispose()
      } catch (disposeError) {
        console.warn('[Live2D] 清理旧模型时出现异常。', disposeError)
      }

      const expressions = getLive2DExpressions(loadedManifest, loadedFormat)
      console.info('[Live2D] 模型加载完成。', {
        modelPath: modelConfig.modelPath,
        format: loadedFormat,
        mocVersion: loadedMocVersion,
        supportedMocVersion: loadedSupportedMocVersion,
        runtime: loadedRuntime,
        resources: loadedResources.map((resource) => resource.url),
        availableExpressions: expressions.map((expression) => expression.id),
      })
      setState((previous) => ({
        ...previous,
        status: 'ready',
        format: loadedFormat,
        loadError: undefined,
        currentModel: requestedModel,
        currentExpression: null,
        availableModels,
        availableExpressions: expressions,
        isModelLoading: false,
        feedback: undefined,
      }))
      if (!options.initial) {
        showFeedback({ tone: 'info', message: `已切换至 ${requestedModel.displayName}。` })
      }
    } catch (error) {
      try {
        candidate?.dispose()
      } catch (disposeError) {
        console.warn('[Live2D] 清理失败的候选模型时出现异常。', disposeError)
      }
      try {
        loaded?.model.destroy()
      } catch (disposeError) {
        console.warn('[Live2D] 清理未挂载模型时出现异常。', disposeError)
      }

      if (requestId !== requestIdRef.current) {
        return
      }

      const normalized = normalizeLive2DError(error, modelConfig.modelPath)
      reportLoadError(normalized, modelConfig)
      if (previousRenderer) {
        setState((previous) => ({
          ...previous,
          isModelLoading: false,
          status: 'ready',
          feedback: undefined,
        }))
        showFeedback({ tone: 'error', message: '模型加载失败，已恢复当前模型。' })
        return
      }

      setState((previous) => ({
        ...previous,
        isModelLoading: false,
        status: 'error',
        loadError: normalized,
      }))
    }
  }, [availableModels, containerRef, showFeedback])

  useEffect(() => {
    if (!config.enabled || !canLoad) {
      return
    }

    const hasActiveRenderer = Boolean(rendererRef.current)
    setState((previous) => ({
      ...previous,
      currentModel: hasActiveRenderer ? previous.currentModel : initialModel,
      currentExpression: hasActiveRenderer ? previous.currentExpression : null,
      availableModels,
      availableExpressions: hasActiveRenderer ? previous.availableExpressions : [],
    }))
    void loadModel(initialModel, { initial: true })
  }, [availableModels, canLoad, config.enabled, configSignature, initialModel, loadModel])

  useEffect(() => {
    if (config.enabled) {
      return
    }

    requestIdRef.current += 1
    clearMenuDismissTimer()
    try {
      rendererRef.current?.dispose()
    } catch (disposeError) {
      console.warn('[Live2D] 停用模型时出现异常。', disposeError)
    }
    rendererRef.current = undefined
    setState((previous) => ({
      ...previous,
      status: 'idle',
      format: undefined,
      loadError: undefined,
      currentExpression: null,
      availableExpressions: [],
      isModelLoading: false,
      openMenu: null,
      renderedMenu: null,
      restoreMenuFocus: false,
      feedback: undefined,
    }))
  }, [clearMenuDismissTimer, config.enabled])

  useEffect(() => {
    const floating = floatingRef.current
    if (!floating || !state.openMenu) {
      return
    }

    const closeOnOutsidePointer = (event: PointerEvent) => {
      if (event.target instanceof Node && !floating.contains(event.target)) {
        closeMenu()
      }
    }
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        closeMenu(true)
      }
    }

    document.addEventListener('pointerdown', closeOnOutsidePointer)
    document.addEventListener('keydown', closeOnEscape)
    return () => {
      document.removeEventListener('pointerdown', closeOnOutsidePointer)
      document.removeEventListener('keydown', closeOnEscape)
    }
  }, [closeMenu, floatingRef, state.openMenu])

  useEffect(() => () => {
    requestIdRef.current += 1
    clearMenuDismissTimer()
    if (feedbackTimerRef.current !== undefined) {
      window.clearTimeout(feedbackTimerRef.current)
    }
    try {
      rendererRef.current?.dispose()
    } catch (disposeError) {
      console.warn('[Live2D] 卸载模型时出现异常。', disposeError)
    }
    rendererRef.current = undefined
  }, [clearMenuDismissTimer])

  const selectModel = useCallback((model: Live2DModelRegistration) => {
    closeMenu()
    if (state.isModelLoading || model.id === state.currentModel.id) {
      return
    }
    void loadModel(model)
  }, [closeMenu, loadModel, state.currentModel.id, state.isModelLoading])

  const selectExpression = useCallback(async (expression: Live2DExpressionOption) => {
    closeMenu()
    if (state.currentExpression === expression.id) {
      return
    }
    const renderer = rendererRef.current
    if (
      state.isModelLoading
      || !renderer
      || !state.availableExpressions.some((available) => available.id === expression.id)
    ) {
      showFeedback({ tone: 'error', message: '当前模型不支持此表情。' })
      return
    }

    try {
      const applied = await renderer.model.expression(expression.id)
      if (!applied || rendererRef.current !== renderer) {
        showFeedback({ tone: 'error', message: '当前模型不支持此表情。' })
        return
      }
      setState((previous) => ({
        ...previous,
        currentExpression: expression.id,
        feedback: undefined,
      }))
    } catch (error) {
      console.warn('[Live2D] 应用表情失败。', error)
      showFeedback({ tone: 'error', message: '当前模型不支持此表情。' })
    }
  }, [closeMenu, showFeedback, state.availableExpressions, state.isModelLoading])

  const retry = useCallback(() => {
    if (!state.isModelLoading) {
      void loadModel(state.currentModel, { initial: true })
    }
  }, [loadModel, state.currentModel, state.isModelLoading])

  return {
    state,
    isMenuOpen: state.openMenu !== null,
    toggleMenu,
    closeMenu,
    selectModel,
    selectExpression,
    retry,
  }
}
