import { get } from './client'
import { apiConfig } from './config'
import type { MusicTrack } from './types'

function resolveMediaUrl(sourceUrl: string | undefined) {
  if (!sourceUrl || !apiConfig.baseUrl) {
    return sourceUrl
  }

  return new URL(sourceUrl, `${apiConfig.baseUrl}/`).toString()
}

function resolveTrackSources(track: MusicTrack): MusicTrack {
  return {
    ...track,
    sourceUrl: resolveMediaUrl(track.sourceUrl),
    qualities: track.qualities?.map((quality) => ({
      ...quality,
      sourceUrl: resolveMediaUrl(quality.sourceUrl) ?? quality.sourceUrl,
    })),
  }
}

export const mediaApi = {
  listMusic: async () => (await get<MusicTrack[]>('/api/v1/music/list')).map(resolveTrackSources),
}
