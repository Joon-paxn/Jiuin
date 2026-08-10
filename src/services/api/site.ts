import { get } from './client'
import type { CopyrightInfo, SiteInfo } from './types'

export const siteApi = {
  getInfo: () => get<SiteInfo>('/api/v1/site/info'),
  getCopyright: () => get<CopyrightInfo>('/api/v1/site/copyright'),
}
