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

export type SharedSiteConfiguration = {
  site: SiteInfo
  copyright: CopyrightInfo
}

export type ServiceStatus = {
  name: string
  status: 'online' | 'degraded' | 'offline' | 'unknown'
}

export type EcosystemStatus = {
  site: ServiceStatus['status']
  api: ServiceStatus['status']
  services: ServiceStatus[]
  checkedAt: string
}

export type ExternalLink = {
  name: string
  url: string
  description: string
}

export type ResourceDescriptor = {
  name: string
  url: string
  priority: 1 | 2 | 3 | 4
  cachePolicy: 'static' | 'config' | 'media'
}

export type PageStatistics = {
  path: string
  views: number
  lastVisitedAt: string
}

export type SiteStatistics = {
  totalViews: number
  pages: PageStatistics[]
}
