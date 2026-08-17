export type SiteRoute = {
  label: string
  path: string
  status: 'available' | 'planned'
}

/**
 * Phase 1 路由规划。未来接入路由库时可直接以此配置生成页面与导航。
 */
export const siteRoutes: readonly SiteRoute[] = [
  { label: '首页', path: '#hero', status: 'available' },
  { label: '简介', path: '#introduction', status: 'available' },
  { label: '更新', path: '#updates', status: 'available' },
  { label: 'Blog', path: '#blog', status: 'available' },
  { label: 'Image', path: '#image', status: 'available' },
  { label: 'API', path: '#api', status: 'available' },
  { label: '关于', path: '#about', status: 'available' },
]
