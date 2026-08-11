import type { Live2DConfig, Live2DModelRegistry } from './types'

function publicAsset(path: string) {
  const base = import.meta.env.BASE_URL.endsWith('/')
    ? import.meta.env.BASE_URL
    : `${import.meta.env.BASE_URL}/`

  return `${base}${path.replace(/^\/+/, '')}`
}

function runtimeAsset(path: string | undefined, fallback: string) {
  if (!path) {
    return fallback
  }

  // An absolute URL deliberately opts out of this app's Vite base path. Local
  // overrides remain base-aware, including values that accidentally start '/'.
  if (/^(?:[a-z][a-z\d+.-]*:)?\/\//i.test(path)) {
    return path
  }

  return publicAsset(path)
}

export const live2dModels = {
  noir: {
    id: 'noir',
    displayName: 'Noir',
    modelPath: publicAsset('live2d/my_model/noir.model3.json'),
    scale: 1,
  },
} as const satisfies Live2DModelRegistry

const activeModel = live2dModels.noir

export const live2dConfig: Live2DConfig = {
  enabled: true,
  modelPath: activeModel.modelPath,
  position: 'right-bottom',
  scale: activeModel.scale,
  displayName: activeModel.displayName,
  lazyLoad: true,
  loadDelayMs: 300,
  modelRegistry: live2dModels,
  initialModelId: activeModel.id,
  runtimePaths: {
    cubism2: runtimeAsset(
      import.meta.env.VITE_LIVE2D_CUBISM2_CORE_URL,
      'https://cdn.jsdelivr.net/gh/dylanNew/live2d/webgl/Live2D/lib/live2d.min.js',
    ),
    cubism4: runtimeAsset(
      import.meta.env.VITE_LIVE2D_CORE_URL,
      publicAsset('live2d/runtime/live2dcubismcore.min.js'),
    ),
  },
}
