export type Live2DStatus = 'idle' | 'loading' | 'ready' | 'error'

export type Live2DModelConfig = {
  enabled: boolean
  modelPath: string
  coreScriptUrl: string
  displayName: string
}
