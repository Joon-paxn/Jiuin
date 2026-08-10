import { get } from './client'
import type { MusicTrack } from './types'

export const mediaApi = {
  listMusic: () => get<MusicTrack[]>('/api/v1/music/list'),
}
