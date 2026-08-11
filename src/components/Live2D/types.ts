import type { Live2DModel } from 'pixi-live2d-display'

export type Live2DStatus = 'idle' | 'loading' | 'ready' | 'error'

export type Live2DModelFormat = 'cubism2' | 'cubism4'

export type Live2DPosition = 'right-bottom'

export type Live2DMenu = 'expressions' | 'models'

export type Live2DModelRegistration = {
  id: string
  displayName: string
  modelPath: string
  scale?: number
}

export type Live2DModelRegistry = Readonly<Record<string, Live2DModelRegistration>>

export type Live2DExpressionOption = {
  id: string
  label: string
  file: string
}

export type Live2DErrorCode =
  | 'WEBGL_UNAVAILABLE'
  | 'MODEL_NOT_FOUND'
  | 'MODEL_MANIFEST_INVALID'
  | 'MODEL_FORMAT_UNSUPPORTED'
  | 'RESOURCE_NOT_FOUND'
  | 'LIVE2D_RUNTIME_NOT_FOUND'
  | 'LIVE2D_RUNTIME_HTTP_ERROR'
  | 'LIVE2D_RUNTIME_CONTENT_TYPE_INVALID'
  | 'LIVE2D_RUNTIME_INVALID_SIZE'
  | 'LIVE2D_RUNTIME_SCRIPT_ERROR'
  | 'LIVE2D_CORE_INIT_FAILED'
  | 'SDK_VERSION_MISMATCH'
  | 'LIVE2D_MODEL_LOAD_FAILED'
  | 'RENDER_SIZE_INVALID'

export type Live2DRuntimePaths = Record<Live2DModelFormat, string>

export type Live2DConfig = {
  enabled: boolean
  modelPath: string
  position: Live2DPosition
  scale: number
  displayName: string
  lazyLoad: boolean
  loadDelayMs: number
  runtimePaths: Live2DRuntimePaths
  /**
   * Models shown by the floating controller. New models only need a registry
   * entry here; the menu is derived from this data at runtime.
   */
  modelRegistry?: Live2DModelRegistry
  initialModelId?: string
}

export type Live2DResource = {
  kind: string
  url: string
}

export type Live2DLoadContext = {
  modelPath?: string
  modelFormat?: Live2DModelFormat
  resourcePath?: string
  runtimePath?: string
  httpStatus?: number
  runtimeContentType?: string
  runtimeBytes?: number
  basePath?: string
  environment?: string
  pageUrl?: string
  coreVersion?: number
  mocVersion?: number
  supportedMocVersion?: number
}

export type Live2DRuntimeCheck = {
  url: string
  httpStatus: number
  contentType: string
  bytes: number
}

export type LoadedLive2DModel = {
  format: Live2DModelFormat
  manifest: Record<string, unknown>
  model: Live2DModel
  pixi: typeof import('pixi.js')
  resources: Live2DResource[]
  runtime: Live2DRuntimeCheck
  mocVersion?: number
  supportedMocVersion?: number
}

export type CubismCoreRuntime = {
  Version?: {
    csmGetVersion?: () => number
    csmGetLatestMocVersion?: () => number
  }
}

declare global {
  interface Window {
    PIXI?: typeof import('pixi.js')
    Live2D?: unknown
    Live2DCubismCore?: CubismCoreRuntime
  }
}
