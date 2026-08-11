import { readFile, stat } from 'node:fs/promises'
import { dirname, relative, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const projectRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const distRoot = resolve(projectRoot, 'dist')
const live2dRoot = resolve(distRoot, 'live2d')
const modelPath = resolve(live2dRoot, 'my_model', 'noir.model3.json')
const runtimePath = resolve(live2dRoot, 'runtime', 'live2dcubismcore.min.js')

function fail(message) {
  throw new Error(`[verify:live2d:dist] ${message}`)
}

function displayPath(path) {
  return relative(projectRoot, path).split(sep).join('/')
}

async function requireFile(path, label) {
  try {
    const details = await stat(path)
    if (!details.isFile() || details.size === 0) {
      fail(`${label} is missing or empty: ${displayPath(path)}`)
    }
    return details
  } catch (error) {
    if (error && typeof error === 'object' && 'code' in error && error.code === 'ENOENT') {
      fail(`${label} is missing: ${displayPath(path)}`)
    }
    throw error
  }
}

function modelResourcePath(reference) {
  if (typeof reference !== 'string' || reference.length === 0) {
    return undefined
  }

  const path = resolve(dirname(modelPath), reference)
  if (!path.startsWith(`${live2dRoot}${sep}`)) {
    fail(`model reference escapes dist/live2d: ${reference}`)
  }
  return path
}

function collectModelResources(manifest) {
  const references = manifest.FileReferences
  if (!references || typeof references !== 'object') {
    fail('noir.model3.json is missing FileReferences')
  }

  const textures = Array.isArray(references.Textures)
    ? references.Textures.filter((texture) => typeof texture === 'string' && texture.length > 0)
    : []
  if (textures.length === 0) {
    fail('noir.model3.json must reference at least one texture')
  }

  const resources = [
    ['Moc', references.Moc],
    ['Physics', references.Physics],
    ['Pose', references.Pose],
    ['DisplayInfo', references.DisplayInfo],
    ['UserData', references.UserData],
  ]

  for (const texture of textures) {
    resources.push(['Texture', texture])
  }
  for (const expression of references.Expressions ?? []) {
    resources.push(['Expression', expression?.File])
  }
  for (const group of Object.values(references.Motions ?? {})) {
    for (const motion of Array.isArray(group) ? group : []) {
      resources.push(['Motion', motion?.File])
      resources.push(['Sound', motion?.Sound])
    }
  }

  return resources
    .map(([label, reference]) => [label, modelResourcePath(reference)])
    .filter(([, path]) => path)
}

const runtime = await requireFile(runtimePath, 'Cubism Core runtime')
if (runtime.size < 1024) {
  fail(`Cubism Core runtime is unexpectedly small (${runtime.size} B)`)
}

await requireFile(resolve(distRoot, 'service-worker.js'), 'Service worker')
await requireFile(modelPath, 'Noir model manifest')

let manifest
try {
  manifest = JSON.parse(await readFile(modelPath, 'utf8'))
} catch (error) {
  fail(`cannot parse noir.model3.json: ${error instanceof Error ? error.message : String(error)}`)
}

for (const [label, path] of collectModelResources(manifest)) {
  await requireFile(path, `Noir ${label}`)
}

const mocPath = modelResourcePath(manifest.FileReferences?.Moc)
if (!mocPath) {
  fail('noir.model3.json has no MOC reference')
}
const mocHeader = await readFile(mocPath)
if (mocHeader.length < 5 || mocHeader.subarray(0, 4).toString('ascii') !== 'MOC3') {
  fail(`Noir MOC has an invalid header: ${displayPath(mocPath)}`)
}

console.info('[verify:live2d:dist] Live2D deployment assets are complete.')
console.info(`  Runtime: ${displayPath(runtimePath)} (${runtime.size} B)`)
console.info(`  MOC: ${displayPath(mocPath)} (MOC3 v${mocHeader[4]})`)
