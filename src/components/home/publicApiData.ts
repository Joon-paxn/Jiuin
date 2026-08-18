export type PublicApi = {
  id: string
  name: string
  method: 'GET'
  endpoint: string
  description: string
  status: 'Preview'
}

// Static catalog only. Each endpoint is registered as a public GET route in the backend.
export const publicApis: readonly PublicApi[] = [
  {
    id: 'site-information',
    name: 'Site Information',
    method: 'GET',
    endpoint: '/api/v1/site/info',
    description: '获取公开的网站基础信息。',
    status: 'Preview',
  },
  {
    id: 'shared-site-data',
    name: 'Shared Site Data',
    method: 'GET',
    endpoint: '/api/v1/site',
    description: '获取主站使用的公开共享数据。',
    status: 'Preview',
  },
  {
    id: 'music-library',
    name: 'Music Library',
    method: 'GET',
    endpoint: '/api/v1/music',
    description: '获取公开音乐库的曲目元数据。',
    status: 'Preview',
  },
  {
    id: 'service-status',
    name: 'Service Status',
    method: 'GET',
    endpoint: '/api/v1/status',
    description: '获取公开服务状态信息。',
    status: 'Preview',
  },
] as const
