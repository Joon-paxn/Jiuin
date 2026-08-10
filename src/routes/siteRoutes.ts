export type SiteRoute = {
  label: string
  path: string
  status: 'available' | 'planned'
}

/**
 * Phase 1 路由规划。未来接入路由库时可直接以此配置生成页面与导航。
 */
export const siteRoutes: readonly SiteRoute[] = [
  { label: '首页', path: '/', status: 'available' },
  { label: '关于', path: '/about', status: 'planned' },
  { label: '项目', path: '/projects', status: 'planned' },
  { label: '资源', path: '/resources', status: 'planned' },
]
