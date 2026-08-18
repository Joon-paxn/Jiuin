export type UpdateDraft = {
  id: string
  timestamp: string
  title: string
  description: string
}

export type TimelineUpdate = UpdateDraft & {
  dateLabel: string
  timeLabel: string
  period: 'morning' | 'afternoon'
}

const displayTimeZone = 'Asia/Shanghai'
const dateFormatter = new Intl.DateTimeFormat('zh-CN', {
  timeZone: displayTimeZone,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
})
const timeFormatter = new Intl.DateTimeFormat('en-GB', {
  timeZone: displayTimeZone,
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
  hour12: false,
})
const hourFormatter = new Intl.DateTimeFormat('en-GB', {
  timeZone: displayTimeZone,
  hour: 'numeric',
  hourCycle: 'h23',
  hour12: false,
})

export const mockUpdates: readonly UpdateDraft[] = [
  {
    id: 'commit-001',
    timestamp: '2026-08-10T09:20:00+08:00',
    title: 'feat: 重构 Jiuin 主站',
    description: '完成主站基础结构和页面系统，为背景、主题与固定组件建立清晰的协作边界。',
  },
  {
    id: 'commit-002',
    timestamp: '2026-08-11T14:32:00+08:00',
    title: 'fix: 修复 Live2D 加载问题',
    description: '修复生产环境 Live2D Runtime 加载问题，并补充本地静态资源校验流程。',
  },
  {
    id: 'commit-003',
    timestamp: '2026-08-14T10:15:00+08:00',
    title: 'feat: 增加随机背景系统',
    description: '增加随机背景、模糊效果以及动态主题支持，让整站保持统一的空间氛围。',
  },
  {
    id: 'commit-004',
    timestamp: '2026-08-16T18:05:00+08:00',
    title: 'fix: 完善生产媒体链路',
    description: '修复生产环境媒体访问与专辑封面链路，同时保留音频 Range 请求和可回滚部署。',
  },
  {
    id: 'commit-005',
    timestamp: '2026-08-17T16:20:00+08:00',
    title: 'feat: 添加 HTTP 响应状态掩码功能',
    description: '新增配置项和中间件，将指定音乐 API 的成功响应对外伪装为 418，同时保持前端业务数据、媒体和健康检查不变。',
  },
] as const

function normalizeUpdate(update: UpdateDraft): TimelineUpdate {
  const date = new Date(update.timestamp)
  if (Number.isNaN(date.getTime())) {
    throw new Error(`Invalid update timestamp: ${update.id}`)
  }

  const dateLabel = dateFormatter.format(date).replaceAll('/', '.')
  const hour = Number(hourFormatter.format(date))

  return {
    ...update,
    dateLabel,
    timeLabel: timeFormatter.format(date),
    period: hour < 12 ? 'morning' : 'afternoon',
  }
}

export function normalizeUpdates(updates: readonly UpdateDraft[]): TimelineUpdate[] {
  return updates
    .map(normalizeUpdate)
    .sort((first, second) => Date.parse(first.timestamp) - Date.parse(second.timestamp))
}

export const updatesData = normalizeUpdates(mockUpdates)
