export type Live2DConfiguration = {
  enabled: boolean
  modelPath: string
  coreScriptUrl: string
  displayName: string
}

/**
 * Live2D runtime and model locations are centralized here so a future model
 * switch does not affect page or component code.
 */
export const live2dConfig: Live2DConfiguration = {
  enabled: true,
  modelPath: '/models/noir/noir.web.model3.json',
  coreScriptUrl: import.meta.env.VITE_LIVE2D_CORE_URL
    ?? 'https://cubism.live2d.com/sdk-web/cubismcore/live2dcubismcore.min.js',
  displayName: 'Noir',
}
