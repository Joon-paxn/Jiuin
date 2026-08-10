import { get } from './client'
import type { EcosystemStatus, ExternalLink, ResourceDescriptor, SiteStatistics } from './types'

/** Public, read-only APIs shared by the main site and future ecosystem sites. */
export const ecosystemApi = {
  getStatistics: () => get<SiteStatistics>('/api/v1/statistics'),
  getStatus: () => get<EcosystemStatus>('/api/v1/status'),
  listLinks: () => get<ExternalLink[]>('/api/v1/links'),
  listResources: () => get<ResourceDescriptor[]>('/api/v1/resources'),
}
