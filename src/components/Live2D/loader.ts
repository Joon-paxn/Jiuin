import type {
  CubismCoreRuntime,
  Live2DConfig,
  Live2DErrorCode,
  Live2DLoadContext,
  Live2DModelFormat,
  Live2DResource,
  Live2DRuntimeCheck,
  LoadedLive2DModel,
} from './types'

type UnknownRecord = Record<string, unknown>

const runtimePromises = new Map<string, Promise<Live2DRuntimeCheck>>()
const RUNTIME_TIMEOUT_MS = 15_000
const MIN_RUNTIME_BYTES = 1_024

export class Live2DLoadError extends Error {
  readonly code: Live2DErrorCode
  readonly context: Live2DLoadContext

  constructor(
    code: Live2DErrorCode,
    message: string,
    context: Live2DLoadContext = {},
    cause?: unknown,
  ) {
    super(message, { cause })
    this.name = 'Live2DLoadError'
    this.code = code
    this.context = context
  }
}

function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function readRecord(value: unknown) {
  return isRecord(value) ? value : undefined
}

function readString(value: unknown) {
  return typeof value === 'string' && value.length > 0 ? value : undefined
}

async function fetchManifest(modelPath: string) {
  let response: Response

  try {
    response = await fetch(modelPath, {
      cache: 'no-cache',
      headers: { Accept: 'application/json' },
    })
  } catch (cause) {
    throw new Live2DLoadError(
      'MODEL_NOT_FOUND',
      '无法请求 Live2D 模型配置文件。',
      { modelPath },
      cause,
    )
  }

  if (!response.ok) {
    throw new Live2DLoadError(
      'MODEL_NOT_FOUND',
      `Live2D 模型配置请求失败（HTTP ${response.status}）。`,
      { modelPath, httpStatus: response.status },
    )
  }

  try {
    const manifest = await response.json() as unknown
    if (!isRecord(manifest)) {
      throw new TypeError('The model manifest must be a JSON object.')
    }
    return manifest
  } catch (cause) {
    throw new Live2DLoadError(
      'MODEL_MANIFEST_INVALID',
      'Live2D 模型配置不是有效 JSON。',
      { modelPath },
      cause,
    )
  }
}

function detectModelFormatFromPath(modelPath: string): Live2DModelFormat {
  const pathname = new URL(modelPath, window.location.href).pathname.toLowerCase()
  if (pathname.endsWith('.model3.json')) {
    return 'cubism4'
  }
  if (pathname.endsWith('.model.json')) {
    return 'cubism2'
  }

  throw new Live2DLoadError(
    'MODEL_FORMAT_UNSUPPORTED',
    '模型入口必须使用 .model.json 或 .model3.json 后缀。',
    { modelPath },
  )
}

export function detectModelFormat(modelPath: string, manifest: UnknownRecord): Live2DModelFormat {
  const filenameFormat = detectModelFormatFromPath(modelPath)

  const fileReferences = readRecord(manifest.FileReferences)
  const structureFormat = readString(fileReferences?.Moc) && Array.isArray(fileReferences?.Textures)
    ? 'cubism4'
    : readString(manifest.model) && Array.isArray(manifest.textures)
      ? 'cubism2'
      : undefined

  if (!structureFormat) {
    throw new Live2DLoadError(
      'MODEL_MANIFEST_INVALID',
      '模型配置缺少 Cubism 所需的 moc 或 textures 字段。',
      { modelPath },
    )
  }

  if (filenameFormat && filenameFormat !== structureFormat) {
    throw new Live2DLoadError(
      'MODEL_FORMAT_UNSUPPORTED',
      '模型文件名与配置结构指向不同的 Cubism 版本。',
      { modelPath, modelFormat: structureFormat },
    )
  }

  return structureFormat
}

function collectResources(
  modelPath: string,
  manifest: UnknownRecord,
  format: Live2DModelFormat,
) {
  const manifestUrl = new URL(modelPath, window.location.href)
  const resources: Live2DResource[] = []

  const add = (kind: string, reference: unknown) => {
    const path = readString(reference)
    if (path) {
      resources.push({ kind, url: new URL(path, manifestUrl).toString() })
    }
  }

  const addArray = (kind: string, references: unknown) => {
    if (Array.isArray(references)) {
      references.forEach((reference) => add(kind, reference))
    }
  }

  if (format === 'cubism4') {
    const references = readRecord(manifest.FileReferences)
    add('moc', references?.Moc)
    addArray('texture', references?.Textures)
    add('physics', references?.Physics)
    add('pose', references?.Pose)
    add('display-info', references?.DisplayInfo)
    add('user-data', references?.UserData)

    if (Array.isArray(references?.Expressions)) {
      references.Expressions.forEach((expression) => add('expression', readRecord(expression)?.File))
    }

    const motions = readRecord(references?.Motions)
    Object.values(motions ?? {}).forEach((group) => {
      if (Array.isArray(group)) {
        group.forEach((motion) => {
          const entry = readRecord(motion)
          add('motion', entry?.File)
          add('sound', entry?.Sound)
        })
      }
    })
  } else {
    add('moc', manifest.model)
    addArray('texture', manifest.textures)
    add('physics', manifest.physics)
    add('pose', manifest.pose)

    if (Array.isArray(manifest.expressions)) {
      manifest.expressions.forEach((expression) => add('expression', readRecord(expression)?.file))
    }

    const motions = readRecord(manifest.motions)
    Object.values(motions ?? {}).forEach((group) => {
      if (Array.isArray(group)) {
        group.forEach((motion) => {
          const entry = readRecord(motion)
          add('motion', entry?.file)
          add('sound', entry?.sound)
        })
      }
    })
  }

  const hasMoc = resources.some((resource) => resource.kind === 'moc')
  const hasTexture = resources.some((resource) => resource.kind === 'texture')
  if (!hasMoc || !hasTexture) {
    throw new Live2DLoadError(
      'MODEL_MANIFEST_INVALID',
      '模型配置必须至少引用一个 moc 文件和一个纹理文件。',
      { modelPath, modelFormat: format },
    )
  }

  return Array.from(new Map(resources.map((resource) => [resource.url, resource])).values())
}

async function verifyResource(resource: Live2DResource, modelPath: string) {
  let response: Response

  try {
    response = await fetch(resource.url, { method: 'HEAD', cache: 'no-cache' })
  } catch (cause) {
    throw new Live2DLoadError(
      'RESOURCE_NOT_FOUND',
      `无法请求模型资源：${resource.kind}。`,
      { modelPath, resourcePath: resource.url },
      cause,
    )
  }

  if (response.status === 405 || response.status === 501) {
    return
  }

  const contentType = response.headers.get('content-type') ?? ''
  if (!response.ok || contentType.includes('text/html')) {
    throw new Live2DLoadError(
      'RESOURCE_NOT_FOUND',
      `模型资源不存在或返回了错误内容：${resource.kind}。`,
      {
        modelPath,
        resourcePath: resource.url,
        httpStatus: response.status,
      },
    )
  }
}

function toAbsoluteUrl(path: string) {
  return new URL(path, window.location.href).toString()
}

function createRuntimeContext(
  format: Live2DModelFormat,
  runtimeUrl: string,
  modelPath: string,
  runtime?: Live2DRuntimeCheck,
): Live2DLoadContext {
  return {
    modelPath,
    modelFormat: format,
    runtimePath: runtimeUrl,
    httpStatus: runtime?.httpStatus,
    runtimeContentType: runtime?.contentType,
    runtimeBytes: runtime?.bytes,
    basePath: import.meta.env.BASE_URL,
    environment: import.meta.env.MODE,
    pageUrl: window.location.href,
  }
}

function isJavaScriptContentType(contentType: string) {
  return /^(?:application|text)\/(?:javascript|ecmascript)(?:;|$)|^application\/x-javascript(?:;|$)/i
    .test(contentType.trim())
}

async function inspectRuntimeRequest(
  format: Live2DModelFormat,
  runtimePath: string,
  modelPath: string,
  signal?: AbortSignal,
): Promise<Live2DRuntimeCheck> {
  const runtimeUrl = toAbsoluteUrl(runtimePath)
  let response: Response

  try {
    response = await fetch(runtimeUrl, {
      cache: 'no-store',
      headers: { Accept: 'application/javascript, text/javascript;q=0.9, */*;q=0.1' },
      signal,
    })
  } catch (cause) {
    throw new Live2DLoadError(
      'LIVE2D_RUNTIME_HTTP_ERROR',
      '无法请求 Live2D Runtime 文件。',
      createRuntimeContext(format, runtimeUrl, modelPath),
      cause,
    )
  }

  const contentType = response.headers.get('content-type') ?? ''
  let bytes: number
  try {
    bytes = (await response.arrayBuffer()).byteLength
  } catch (cause) {
    throw new Live2DLoadError(
      'LIVE2D_RUNTIME_HTTP_ERROR',
      'Live2D Runtime 响应体无法读取。',
      createRuntimeContext(format, runtimeUrl, modelPath, {
        url: runtimeUrl,
        httpStatus: response.status,
        contentType,
        bytes: 0,
      }),
      cause,
    )
  }

  const runtime = { url: runtimeUrl, httpStatus: response.status, contentType, bytes }
  const checkedContext = createRuntimeContext(format, runtimeUrl, modelPath, runtime)

  if (response.status === 404) {
    throw new Live2DLoadError(
      'LIVE2D_RUNTIME_NOT_FOUND',
      'Live2D Runtime 文件不存在（HTTP 404）。',
      checkedContext,
    )
  }

  if (!response.ok) {
    throw new Live2DLoadError(
      'LIVE2D_RUNTIME_HTTP_ERROR',
      `Live2D Runtime 请求失败（HTTP ${response.status}）。`,
      checkedContext,
    )
  }

  if (!isJavaScriptContentType(contentType)) {
    throw new Live2DLoadError(
      'LIVE2D_RUNTIME_CONTENT_TYPE_INVALID',
      `Live2D Runtime 返回了无效 MIME：${contentType || '(missing)'}。`,
      checkedContext,
    )
  }

  if (bytes < MIN_RUNTIME_BYTES) {
    throw new Live2DLoadError(
      'LIVE2D_RUNTIME_INVALID_SIZE',
      `Live2D Runtime 文件过小（${bytes} B），可能是错误页面或不完整部署。`,
      checkedContext,
    )
  }

  return runtime
}

async function inspectRuntime(
  format: Live2DModelFormat,
  runtimePath: string,
  modelPath: string,
): Promise<Live2DRuntimeCheck> {
  const runtimeUrl = toAbsoluteUrl(runtimePath)
  const controller = typeof AbortController === 'undefined' ? undefined : new AbortController()
  let timedOut = false
  let timeoutId = 0

  const deadline = new Promise<never>((_, reject) => {
    timeoutId = window.setTimeout(() => {
      timedOut = true
      controller?.abort()
      reject(new Error('Live2D Runtime preflight timed out.'))
    }, RUNTIME_TIMEOUT_MS)
  })

  try {
    return await Promise.race([
      inspectRuntimeRequest(format, runtimeUrl, modelPath, controller?.signal),
      deadline,
    ])
  } catch (cause) {
    if (!timedOut) {
      throw cause
    }

    const context = cause instanceof Live2DLoadError
      ? cause.context
      : createRuntimeContext(format, runtimeUrl, modelPath)
    throw new Live2DLoadError(
      'LIVE2D_RUNTIME_HTTP_ERROR',
      `Live2D Runtime 请求或读取超时（${RUNTIME_TIMEOUT_MS / 1_000} 秒）。`,
      context,
      cause,
    )
  } finally {
    window.clearTimeout(timeoutId)
  }
}

function validateRuntimeGlobal(format: Live2DModelFormat) {
  if (format === 'cubism2') {
    if (!window.Live2D) {
      throw new Error('Live2D 全局对象不存在。')
    }
    return
  }

  const version = window.Live2DCubismCore?.Version
  if (!version?.csmGetVersion || !version.csmGetLatestMocVersion) {
    throw new Error('Live2DCubismCore 或其 Version API 不存在。')
  }

  const coreVersion = version.csmGetVersion()
  const supportedMocVersion = version.csmGetLatestMocVersion()
  if (!Number.isFinite(coreVersion) || !Number.isFinite(supportedMocVersion) || supportedMocVersion <= 0) {
    throw new Error('Live2DCubismCore 未完成初始化。')
  }
}

function runtimeIsReady(format: Live2DModelFormat) {
  try {
    validateRuntimeGlobal(format)
    return true
  } catch {
    return false
  }
}

function findManagedRuntimeScript(format: Live2DModelFormat, runtimeUrl: string) {
  return Array.from(document.querySelectorAll<HTMLScriptElement>('script[data-live2d-runtime]'))
    .find((script) => (
      script.dataset.live2dRuntime === format
      && script.dataset.live2dRuntimeUrl === runtimeUrl
    ))
}

function loadRuntimeScript(
  format: Live2DModelFormat,
  runtime: Live2DRuntimeCheck,
  modelPath: string,
) {
  return new Promise<void>((resolve, reject) => {
    const existing = findManagedRuntimeScript(format, runtime.url)
    const script = existing ?? document.createElement('script')
    const createdByLoader = !existing
    let settled = false
    let timeoutId = 0

    const cleanup = () => {
      window.clearTimeout(timeoutId)
      script.removeEventListener('load', handleLoad)
      script.removeEventListener('error', handleError)
      window.removeEventListener('error', handleExecutionError)
    }
    const fail = (code: Live2DErrorCode, message: string, cause?: unknown) => {
      if (settled) {
        return
      }
      settled = true
      cleanup()
      if (createdByLoader) {
        script.remove()
      }
      reject(new Live2DLoadError(
        code,
        message,
        createRuntimeContext(format, runtime.url, modelPath, runtime),
        cause,
      ))
    }
    const succeed = () => {
      if (settled) {
        return
      }
      settled = true
      script.dataset.live2dRuntimeState = 'loaded'
      cleanup()
      resolve()
    }
    const handleLoad = () => {
      try {
        validateRuntimeGlobal(format)
        succeed()
      } catch (cause) {
        fail(
          'LIVE2D_CORE_INIT_FAILED',
          'Live2D Runtime 脚本已加载，但 Core 初始化未完成。',
          cause,
        )
      }
    }
    const handleError = (event: Event) => fail(
      'LIVE2D_RUNTIME_SCRIPT_ERROR',
      '浏览器无法执行 Live2D Runtime 脚本。',
      event,
    )
    const handleExecutionError = (event: ErrorEvent) => {
      if (event.filename && toAbsoluteUrl(event.filename) === runtime.url) {
        fail('LIVE2D_CORE_INIT_FAILED', 'Live2D Core 初始化时抛出了异常。', event.error ?? event)
      }
    }

    script.addEventListener('load', handleLoad, { once: true })
    script.addEventListener('error', handleError, { once: true })
    window.addEventListener('error', handleExecutionError)
    timeoutId = window.setTimeout(
      () => fail('LIVE2D_RUNTIME_SCRIPT_ERROR', 'Live2D Runtime 脚本加载超时。'),
      RUNTIME_TIMEOUT_MS,
    )

    if (!existing) {
      script.async = true
      script.src = runtime.url
      script.dataset.live2dRuntime = format
      script.dataset.live2dRuntimeUrl = runtime.url
      script.dataset.live2dRuntimeState = 'loading'
      try {
        document.head.append(script)
      } catch (cause) {
        fail('LIVE2D_RUNTIME_SCRIPT_ERROR', '无法将 Live2D Runtime 脚本插入页面。', cause)
      }
    } else if (existing.dataset.live2dRuntimeState === 'loaded') {
      window.queueMicrotask(handleLoad)
    }
  })
}

function ensureRuntime(format: Live2DModelFormat, runtimePath: string, modelPath: string) {
  const runtimeUrl = toAbsoluteUrl(runtimePath)
  const key = `${format}:${runtimeUrl}`
  const activePromise = runtimePromises.get(key)
  if (activePromise) {
    return activePromise
  }

  const promise = (async () => {
    const runtime = await inspectRuntime(format, runtimeUrl, modelPath)

    if (!runtimeIsReady(format)) {
      await loadRuntimeScript(format, runtime, modelPath)
    }

    try {
      validateRuntimeGlobal(format)
    } catch (cause) {
      throw new Live2DLoadError(
        'LIVE2D_CORE_INIT_FAILED',
        'Live2D Core 未暴露可用的初始化 API。',
        createRuntimeContext(format, runtime.url, modelPath, runtime),
        cause,
      )
    }

    return runtime
  })().catch((error: unknown) => {
    runtimePromises.delete(key)
    throw error
  })

  runtimePromises.set(key, promise)
  return promise
}

async function inspectMoc(
  resource: Live2DResource,
  format: Live2DModelFormat,
  modelPath: string,
) {
  let response: Response

  try {
    response = await fetch(resource.url, {
      // A cached 206 range response can be reused as the full moc response
      // by some static servers, which leaves Cubism with a truncated binary.
      cache: 'no-store',
      headers: { Range: 'bytes=0-7' },
    })
  } catch (cause) {
    throw new Live2DLoadError(
      'RESOURCE_NOT_FOUND',
      '无法读取 moc 文件头。',
      { modelPath, modelFormat: format, resourcePath: resource.url },
      cause,
    )
  }

  if (!response.ok) {
    throw new Live2DLoadError(
      'RESOURCE_NOT_FOUND',
      `moc 文件请求失败（HTTP ${response.status}）。`,
      {
        modelPath,
        modelFormat: format,
        resourcePath: resource.url,
        httpStatus: response.status,
      },
    )
  }

  const bytes = new Uint8Array(await response.arrayBuffer()).subarray(0, 8)
  const signature = String.fromCharCode(...bytes.subarray(0, format === 'cubism4' ? 4 : 3))
  const expectedSignature = format === 'cubism4' ? 'MOC3' : 'moc'
  if (signature !== expectedSignature) {
    throw new Live2DLoadError(
      'MODEL_FORMAT_UNSUPPORTED',
      `moc 文件签名为 ${JSON.stringify(signature)}，预期 ${expectedSignature}。`,
      { modelPath, modelFormat: format, resourcePath: resource.url },
    )
  }

  return format === 'cubism4' ? bytes[4] : undefined
}

function getSupportedMocVersion(runtime: CubismCoreRuntime | undefined) {
  try {
    return runtime?.Version?.csmGetLatestMocVersion?.()
  } catch {
    return undefined
  }
}

export async function loadLive2DModel(config: Live2DConfig): Promise<LoadedLive2DModel> {
  const format = detectModelFormatFromPath(config.modelPath)
  const runtimePath = config.runtimePaths[format]

  // Runtime availability and Core initialization must succeed before Pixi or
  // model loading begins. This keeps deployment failures attributable to the
  // actual Runtime URL instead of surfacing later as a generic model error.
  const runtime = await ensureRuntime(format, runtimePath, config.modelPath)
  const pixi = await import('pixi.js')
  window.PIXI = pixi

  const manifest = await fetchManifest(config.modelPath)
  detectModelFormat(config.modelPath, manifest)
  const resources = collectResources(config.modelPath, manifest, format)
  await Promise.all(resources.map((resource) => verifyResource(resource, config.modelPath)))

  const mocResource = resources.find((resource) => resource.kind === 'moc')!
  const mocVersion = await inspectMoc(mocResource, format, config.modelPath)
  const supportedMocVersion = format === 'cubism4'
    ? getSupportedMocVersion(window.Live2DCubismCore)
    : undefined

  if (
    mocVersion !== undefined
    && supportedMocVersion !== undefined
    && mocVersion > supportedMocVersion
  ) {
    throw new Live2DLoadError(
      'SDK_VERSION_MISMATCH',
      `模型需要 MOC3 v${mocVersion}，当前 SDK 最高支持 v${supportedMocVersion}。`,
      {
        modelPath: config.modelPath,
        modelFormat: format,
        runtimePath: runtime.url,
        httpStatus: runtime.httpStatus,
        runtimeContentType: runtime.contentType,
        runtimeBytes: runtime.bytes,
        mocVersion,
        supportedMocVersion,
      },
    )
  }

  try {
    const adapter = format === 'cubism4'
      ? await import('pixi-live2d-display/cubism4')
      : await import('pixi-live2d-display/cubism2')

    adapter.Live2DModel.registerTicker(pixi.Ticker)
    const model = await adapter.Live2DModel.from(config.modelPath, { autoInteract: false })

    return {
      format,
      manifest,
      model,
      pixi,
      resources,
      runtime,
      mocVersion,
      supportedMocVersion,
    }
  } catch (cause) {
    if (cause instanceof Live2DLoadError) {
      throw cause
    }

    throw new Live2DLoadError(
      'LIVE2D_MODEL_LOAD_FAILED',
      'Live2D 适配器无法创建模型，请检查模型格式、纹理和 SDK 版本。',
      {
        modelPath: config.modelPath,
        modelFormat: format,
        runtimePath: runtime.url,
        httpStatus: runtime.httpStatus,
        runtimeContentType: runtime.contentType,
        runtimeBytes: runtime.bytes,
        mocVersion,
        supportedMocVersion,
      },
      cause,
    )
  }
}

export function normalizeLive2DError(error: unknown, modelPath: string) {
  if (error instanceof Live2DLoadError) {
    return error
  }

  return new Live2DLoadError(
    'LIVE2D_MODEL_LOAD_FAILED',
    error instanceof Error ? error.message : 'Live2D 模型加载失败。',
    { modelPath },
    error,
  )
}
