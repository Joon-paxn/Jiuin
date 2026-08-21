import { access, readFile } from 'node:fs/promises'
import { constants } from 'node:fs'
import { resolve } from 'node:path'

const dist = resolve(import.meta.dirname, '..', 'dist')
const required = [
  'index.html',
  'live2d/runtime/live2dcubismcore.min.js',
  'live2d/my_model/noir.model3.json',
  'live2d/my_model/noir.moc3',
]

for (const relativePath of required) {
  const target = resolve(dist, relativePath)
  try {
    await access(target, constants.R_OK)
  } catch {
    throw new Error(`Required production asset is missing: ${relativePath}`)
  }
}

const document = await readFile(resolve(dist, 'index.html'), 'utf8')
if (!document.includes('<div id="root"></div>')) {
  throw new Error('Production document does not contain the application root.')
}

console.log('Live2D production assets verified.')
