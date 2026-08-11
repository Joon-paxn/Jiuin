const [originArgument, ...flags] = process.argv.slice(2)
const allowTextJavaScript = flags.includes('--allow-text-javascript')

if (!originArgument) {
  throw new Error(
    'Usage: npm run verify:live2d:http -- https://example.com[/base/] [--allow-text-javascript]',
  )
}

const baseUrl = new URL(originArgument.endsWith('/') ? originArgument : `${originArgument}/`)
const runtimeUrl = new URL('live2d/runtime/live2dcubismcore.min.js', baseUrl)
const manifestUrl = new URL('live2d/my_model/noir.model3.json', baseUrl)

function fail(message) {
  throw new Error(`[verify:live2d:http] ${message}`)
}

function isJavaScriptContentType(contentType) {
  if (allowTextJavaScript) {
    return /^(?:application|text)\/(?:javascript|ecmascript)(?:;|$)|^application\/x-javascript(?:;|$)/i
      .test(contentType)
  }
  return /^application\/javascript(?:;|$)/i.test(contentType)
}

async function get(url, label) {
  let response
  try {
    response = await fetch(url, { cache: 'no-store' })
  } catch (error) {
    fail(`${label} request failed for ${url}: ${error instanceof Error ? error.message : String(error)}`)
  }
  if (!response.ok) {
    fail(`${label} returned HTTP ${response.status} for ${url}`)
  }
  return response
}

function collectModelReferences(manifest) {
  const references = manifest?.FileReferences
  if (!references || typeof references !== 'object') {
    fail('Noir model manifest is missing FileReferences')
  }

  const resources = []
  const add = (label, reference) => {
    if (typeof reference === 'string' && reference.length > 0) {
      resources.push({ label, reference })
    }
  }

  add('MOC', references.Moc)
  add('Physics', references.Physics)
  add('Pose', references.Pose)
  add('DisplayInfo', references.DisplayInfo)
  add('UserData', references.UserData)

  const textures = Array.isArray(references.Textures)
    ? references.Textures.filter((texture) => typeof texture === 'string' && texture.length > 0)
    : []
  if (textures.length === 0) {
    fail('Noir model manifest must reference at least one texture')
  }
  for (const texture of textures) add('Texture', texture)

  for (const expression of Array.isArray(references.Expressions) ? references.Expressions : []) {
    add('Expression', expression?.File)
  }
  for (const group of Object.values(references.Motions ?? {})) {
    for (const motion of Array.isArray(group) ? group : []) {
      add('Motion', motion?.File)
      add('Sound', motion?.Sound)
    }
  }

  if (!resources.some((resource) => resource.label === 'MOC')) {
    fail('Noir model manifest has no FileReferences.Moc value')
  }
  return resources
}

const runtimeResponse = await get(runtimeUrl, 'Cubism Core runtime')
const runtimeContentType = runtimeResponse.headers.get('content-type') ?? ''
const runtimeBytes = (await runtimeResponse.arrayBuffer()).byteLength
if (!isJavaScriptContentType(runtimeContentType)) {
  fail(`Cubism Core MIME must be application/javascript${allowTextJavaScript ? ' or text/javascript' : ''}; received ${runtimeContentType || '(missing)'}`)
}
if (runtimeBytes < 1024) {
  fail(`Cubism Core response is unexpectedly small (${runtimeBytes} B)`)
}

const manifestResponse = await get(manifestUrl, 'Noir model manifest')
const manifestContentType = manifestResponse.headers.get('content-type') ?? ''
if (!/^application\/json(?:;|$)/i.test(manifestContentType)) {
  fail(`Noir model manifest must be JSON; received ${manifestContentType || '(missing)'}`)
}
const manifest = await manifestResponse.json()
const resources = collectModelReferences(manifest)
const loadedResources = await Promise.all(resources.map(async ({ label, reference }) => {
  const url = new URL(reference, manifestUrl)
  const response = await get(url, `Noir ${label}`)
  const contentType = response.headers.get('content-type') ?? ''
  const bytes = new Uint8Array(await response.arrayBuffer())
  if (contentType.toLowerCase().includes('text/html') || bytes.byteLength === 0) {
    fail(`Noir ${label} returned invalid content at ${url}`)
  }
  return { label, url, bytes }
}))

const moc = loadedResources.find((resource) => resource.label === 'MOC')
if (!moc) {
  fail('Noir MOC could not be loaded')
}
const mocHeader = moc.bytes.subarray(0, 5)
if (mocHeader.length < 5 || String.fromCharCode(...mocHeader.subarray(0, 4)) !== 'MOC3') {
  fail(`Noir MOC has an invalid header at ${moc.url}`)
}

console.info('[verify:live2d:http] Live2D HTTP deployment checks passed.')
console.info(`  Runtime: ${runtimeUrl} (${runtimeContentType}, ${runtimeBytes} B)`)
console.info(`  Model: ${manifestUrl}`)
console.info(`  Resources: ${loadedResources.length}`)
console.info(`  MOC: ${moc.url} (MOC3 v${mocHeader[4]})`)
