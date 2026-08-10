export type ApiResponse<T> = {
  code: number
  message: string
  data?: T
}

export type SiteInfo = {
  name: string
  project: string
  domain: string
}

export type CopyrightInfo = {
  year: number
  text: string
}

export type AudioQuality = {
  id: string
  label: string
  sourceUrl: string
  bitrateKbps?: number
}

export type MusicTrack = {
  id: string
  title: string
  artist: string
  artworkUrl?: string
  sourceUrl?: string
  durationSeconds?: number
  qualities?: AudioQuality[]
}
