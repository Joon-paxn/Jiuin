import type {
  CubismCoreRuntime,
  Live2DConfig,
  Live2DErrorCode,
  Live2DLoadContext,
  Live2DModelFormat,
  Live2DResource,
  LoadedLive2DModel,
} from './types'

type UnknownRecord = Record<string, unknown>

const runtimePromises = new Map<Live2DModelFormat, Promise<void>>()

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

export function detectModelFormat(modelPath: string, manifest: UnknownRecord): Live2DModelFormat {
  const pathname = new URL(modelPath, window.location.href).pathname.toLowerCase()
  const filenameFormat = pathname.endsWith('.model3.json')
    ? 'cubism4'
    : pathname.endsWith('.model.json')
      ? 'cubism2'
      : undefined

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

  if (!filenameFormat) {
    throw new Live2DLoadError(
      'MODEL_FORMAT_UNSUPPORTED',
      '模型入口必须使用 .model.json 或 .model3.json 后缀。',
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

function runtimeIsReady(format: Live2DModelFormat) {
  return format === 'cubism4'
    ? Boolean(window.Live2DCubismCore)
    : Boolean(window.Live2D)
}

function ensureRuntime(format: Live2DModelFormat, runtimePath: string, modelPath: string) {
  if (runtimeIsReady(format)) {
    return Promise.resolve()
  }

  const activePromise = runtimePromises.get(format)
  if (activePromise) {
    return activePromise
  }

  const promise = new Promise<void>((resolve, reject) => {
    const selector = `script[data-live2d-runtime="${format}"]`
    const existing = document.querySelector<HTMLScriptElement>(selector)
    const script = existing ?? document.createElement('script')
    let timeoutId = 0

    const cleanup = () => {
      window.clearTimeout(timeoutId)
      script.removeEventListener('load', handleLoad)
      script.removeEventListener('error', handleError)
    }
    const fail = (message: string, cause?: unknown) => {
      cleanup()
      reject(new Live2DLoadError(
        'RUNTIME_LOAD_FAILED',
        message,
        { modelPath, modelFormat: format, runtimePath },
        cause,
      ))
    }
    const handleLoad = () => {
      if (runtimeIsReady(format)) {
        cleanup()
        resolve()
      } else {
        fail('Live2D 运行时脚本已加载，但没有暴露预期的 SDK 全局对象。')
      }
    }
    const handleError = (event: Event) => fail('Live2D 运行时脚本加载失败。', event)

    script.addEventListener('load', handleLoad, { once: true })
    script.addEventListener('error', handleError, { once: true })
    timeoutId = window.setTimeout(
      () => fail('Live2D 运行时脚本加载超时。'),
      15_000,
    )

    if (!existing) {
      script.async = true
      script.src = runtimePath
      script.dataset.live2dRuntime = format
      document.head.append(script)
    }
  }).catch((error: unknown) => {
    runtimePromises.delete(format)
    throw error
  })

  runtimePromises.set(format, promise)
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
  const manifest = await fetchManifest(config.modelPath)
  const format = detectModelFormat(config.modelPath, manifest)
  const resources = collectResources(config.modelPath, manifest, format)
  const runtimePath = config.runtimePaths[format]

  await Promise.all(resources.map((resource) => verifyResource(resource, config.modelPath)))

  const [pixi] = await Promise.all([
    import('pixi.js'),
    ensureRuntime(format, runtimePath, config.modelPath),
  ])
  window.PIXI = pixi

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
        runtimePath,
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
      mocVersion,
      supportedMocVersion,
    }
  } catch (cause) {
    if (cause instanceof Live2DLoadError) {
      throw cause
    }

    throw new Live2DLoadError(
      'MODEL_LOAD_FAILED',
      'Live2D 适配器无法创建模型，请检查模型格式、纹理和 SDK 版本。',
      {
        modelPath: config.modelPath,
        modelFormat: format,
        runtimePath,
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
    'MODEL_LOAD_FAILED',
    error instanceof Error ? error.message : 'Live2D 模型加载失败。',
    { modelPath },
    error,
  )
}
