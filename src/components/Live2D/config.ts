import type { Live2DConfig } from './types'

function publicAsset(path: string) {
  const base = import.meta.env.BASE_URL.endsWith('/')
    ? import.meta.env.BASE_URL
    : `${import.meta.env.BASE_URL}/`

  return `${base}${path.replace(/^\//, '')}`
}

export const live2dModels = {
  noir: {
    displayName: 'Noir',
    modelPath: publicAsset('live2d/my_model/noir.model3.json'),
    scale: 1,
  },
} as const

const activeModel = live2dModels.noir

export const live2dConfig: Live2DConfig = {
  enabled: true,
  modelPath: activeModel.modelPath,
  position: 'right-bottom',
  scale: activeModel.scale,
  displayName: activeModel.displayName,
  lazyLoad: true,
  loadDelayMs: 300,
  runtimePaths: {
    cubism2: import.meta.env.VITE_LIVE2D_CUBISM2_CORE_URL
      || 'https://cdn.jsdelivr.net/gh/dylanNew/live2d/webgl/Live2D/lib/live2d.min.js',
    cubism4: import.meta.env.VITE_LIVE2D_CORE_URL
      || publicAsset('live2d/runtime/live2dcubismcore.min.js'),
  },
}
