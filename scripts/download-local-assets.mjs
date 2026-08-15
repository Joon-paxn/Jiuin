import { mkdir, writeFile } from 'node:fs/promises'

const publicDirectory = new URL('../public/', import.meta.url)
const backgroundDirectory = new URL('backgrounds/', publicDirectory)
const live2dRuntimeDirectory = new URL('live2d/runtime/', publicDirectory)

const browserHeaders = {
  'user-agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/140 Safari/537.36',
}

const binaryAssets = [
  ...Array.from({ length: 10 }, (_, index) => ({
    source: `https://image.cn-zj1.rains3.com/pc/img${index + 1}.jpg`,
    destination: new URL(`img${index + 1}.jpg`, backgroundDirectory),
  })),
  {
    source: 'https://cdn.jsdelivr.net/gh/dylanNew/live2d/webgl/Live2D/lib/live2d.min.js',
    destination: new URL('live2d.min.js', live2dRuntimeDirectory),
  },
]

async function fetchRequired(url, options = {}) {
  const response = await fetch(url, { headers: browserHeaders, signal: AbortSignal.timeout(30_000), ...options })
  if (!response.ok) {
    throw new Error(`${url} returned ${response.status}`)
  }
  return response
}

async function mapConcurrent(items, concurrency, mapper) {
  const results = new Array(items.length)
  let nextIndex = 0

  await Promise.all(Array.from({ length: Math.min(concurrency, items.length) }, async () => {
    while (nextIndex < items.length) {
      const index = nextIndex++
      results[index] = await mapper(items[index], index)
    }
  }))

  return results
}

await Promise.all([mkdir(backgroundDirectory, { recursive: true }), mkdir(live2dRuntimeDirectory, { recursive: true })])

await mapConcurrent(binaryAssets, 4, async (asset) => {
  await writeFile(asset.destination, Buffer.from(await (await fetchRequired(asset.source)).arrayBuffer()))
})

console.log(`Downloaded ${binaryAssets.length} local assets.`)
