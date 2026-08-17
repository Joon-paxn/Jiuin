export type UpdateEntry = {
  date: string
  time: string
  title: string
  description: string
  period: 'morning' | 'afternoon'
}

// Mock history is intentionally shaped like a future GitHub commit response.
export const updatesData: readonly UpdateEntry[] = [
  {
    date: '2026.06.12',
    time: '10:40',
    title: 'chore: 建立霁雪居基础结构',
    description: '完成站点骨架、主题变量与首屏布局，为后续页面留出清晰的扩展边界。',
    period: 'morning',
  },
  {
    date: '2026.07.03',
    time: '15:10',
    title: 'feat: 接入音乐与媒体资源',
    description: '加入播放器、公开媒体路径与本地资源策略，让声音成为空间的一部分。',
    period: 'afternoon',
  },
  {
    date: '2026.08.04',
    time: '09:25',
    title: 'feat: 背景资源单例加载',
    description: '由服务端选择单张 CDN 背景，Loading、主题和背景层共享同一个资源 Promise。',
    period: 'morning',
  },
  {
    date: '2026.08.16',
    time: '18:05',
    title: 'fix: 完善生产媒体链路',
    description: '修复生产环境媒体访问与封面链路，保留音频 Range 请求和可回滚部署。',
    period: 'afternoon',
  },
  {
    date: '2026.08.17',
    time: '16:20',
    title: 'feat: 添加 HTTP 响应状态掩码功能',
    description: '新增配置项和中间件，将指定音乐 API 的成功响应对外伪装为 418，同时保持前端业务数据、媒体和健康检查不变。',
    period: 'afternoon',
  },
] as const
