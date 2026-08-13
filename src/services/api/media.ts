import { get } from './client'
import { apiConfig } from './config'
import { resolveMusicMediaUrl } from './mediaUrl'
import type { AudioQuality, MusicTrack, PublicMusicTrack } from './types'

function resolveMediaUrl(sourceUrl: string | undefined) {
  return resolveMusicMediaUrl(sourceUrl, apiConfig.baseUrl, window.location.protocol)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function optionalText(value: unknown) {
  return typeof value === 'string' && value.trim() !== '' ? value.trim() : undefined
}

function optionalDuration(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0 ? value : undefined
}

function optionalSize(value: unknown) {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0 ? value : undefined
}

function parsePublicMusicTrack(value: unknown): PublicMusicTrack | undefined {
  if (!isRecord(value) || !isRecord(value.audio)) {
    return undefined
  }

  const id = optionalText(value.id)
  const title = optionalText(value.title)
  const artist = optionalText(value.artist)
  if (!id || !title || !artist) {
    return undefined
  }

  return {
    id,
    title,
    artist,
    album: optionalText(value.album),
    albumArtist: optionalText(value.albumArtist),
    genre: optionalText(value.genre),
    year: optionalText(value.year),
    cover: optionalText(value.cover),
    durationSeconds: optionalDuration(value.durationSeconds),
    fullSize: optionalSize(value.fullSize),
    liteSize: optionalSize(value.liteSize),
    createdAt: optionalText(value.createdAt),
    audio: {
      full: optionalText(value.audio.full),
      lite: optionalText(value.audio.lite),
    },
  }
}

function toAudioQuality(id: 'full' | 'lite', label: string, sourceUrl: string | undefined): AudioQuality | undefined {
  const resolvedSourceUrl = resolveMediaUrl(sourceUrl)
  if (!resolvedSourceUrl) {
    return undefined
  }

  return { id, label, sourceUrl: resolvedSourceUrl }
}

function toMusicTrack(track: PublicMusicTrack): MusicTrack | undefined {
  const qualities = [
    toAudioQuality('full', '完整版', track.audio?.full),
    toAudioQuality('lite', '省流版', track.audio?.lite),
  ].filter((quality): quality is AudioQuality => quality !== undefined)

  if (qualities.length === 0) {
    return undefined
  }

  return {
    id: track.id,
    title: track.title,
    artist: track.artist,
    artworkUrl: resolveMediaUrl(track.cover),
    durationSeconds: track.durationSeconds,
    sourceUrl: qualities[0]?.sourceUrl,
    qualities,
  }
}

export const mediaApi = {
  listMusic: async () => {
    const payload = await get<unknown>('/api/v1/music')
    if (!Array.isArray(payload)) {
      throw new Error('Music list response data is invalid')
    }

    return payload
      .map(parsePublicMusicTrack)
      .flatMap((track) => track ? [toMusicTrack(track)] : [])
      .filter((track): track is MusicTrack => track !== undefined)
  },
}
