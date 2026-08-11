import type { Live2DModel } from 'pixi-live2d-display'

export type Live2DStatus = 'idle' | 'loading' | 'ready' | 'error'

export type Live2DModelFormat = 'cubism2' | 'cubism4'

export type Live2DPosition = 'right-bottom'

export type Live2DErrorCode =
  | 'WEBGL_UNAVAILABLE'
  | 'MODEL_NOT_FOUND'
  | 'MODEL_MANIFEST_INVALID'
  | 'MODEL_FORMAT_UNSUPPORTED'
  | 'RESOURCE_NOT_FOUND'
  | 'RUNTIME_LOAD_FAILED'
  | 'SDK_VERSION_MISMATCH'
  | 'MODEL_LOAD_FAILED'
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
  mocVersion?: number
  supportedMocVersion?: number
}

export type LoadedLive2DModel = {
  format: Live2DModelFormat
  manifest: Record<string, unknown>
  model: Live2DModel
  pixi: typeof import('pixi.js')
  resources: Live2DResource[]
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
