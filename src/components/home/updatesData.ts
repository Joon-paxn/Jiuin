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

// Snapshot of the local history from the requested root commit through the latest commit.
const commitHistory = [
  { id: "c81c8a0def3fe185d9d19f10556d9d118609abcb", timestamp: "2026-08-10T14:13:23+08:00", title: "chore: initialize Jiuin frontend foundation" },
  { id: "f8432af909886d5b46b7307281828026b14ca1f0", timestamp: "2026-08-10T14:51:16+08:00", title: "refactor: 重构全站基础组件与样式体系，统一 UI 规范" },
  { id: "bf2a5e0e1f10e3c99d59c2527191740be6a6cc6b", timestamp: "2026-08-10T15:09:40+08:00", title: "临时添加: 过滤部分开发文件" },
  { id: "b20bb03e7ece95d86286046d24f2fb7667b672c8", timestamp: "2026-08-10T19:20:23+08:00", title: "feat: 完成项目初始搭建与基础功能开发" },
  { id: "af3b5b1b827b5c76da54559991adbfb098cbc110", timestamp: "2026-08-10T19:27:13+08:00", title: "feat: add configurable background component and progress bar theme token" },
  { id: "c4df8bad0e1e43288b397a5c5b150de42aae6f57", timestamp: "2026-08-10T19:49:08+08:00", title: "feat: 新增Live2D看板娘与音乐播放器功能，完善后端媒体接口" },
  { id: "2a88314f9324fc402c565b266e9fa637c5bbc5b0", timestamp: "2026-08-10T20:15:14+08:00", title: "feat: 新增站點生態系統相關功能與後端API" },
  { id: "d39ef91b40e2eeac5936206c78f5e1c5499767a2", timestamp: "2026-08-11T06:36:08+08:00", title: "feat(experiments): add temporary opening animation experiment" },
  { id: "cd88803f79745cffdc48ca960172bfbbed68a748", timestamp: "2026-08-11T06:43:46+08:00", title: "临时添加: 过滤部分开发文件" },
  { id: "6c0acf139f15caa2e9e1d3731f5a2eec7edfa25a", timestamp: "2026-08-11T13:55:30+08:00", title: "refactor(live2d): 重构Live2D组件并完善模型配置" },
  { id: "69c9f7c015b99f0e2e1d84ecb5a4f8e56482c0e2", timestamp: "2026-08-11T15:18:46+08:00", title: "refactor: 重构全站布局与组件样式，优化交互体验" },
  { id: "ab10426f98b312fe792e56390dcf4c5f89e18b60", timestamp: "2026-08-11T15:31:57+08:00", title: "build(vite): configure dev server host port and strict port" },
  { id: "16a27b849c322b4e0fc3a3f7c40743d44cc346c4", timestamp: "2026-08-11T15:39:39+08:00", title: "build(vite): add allowed hosts for jiuin.cn and its www subdomain" },
  { id: "494b030ef6640eba512de2b9f581f11d0f60b970", timestamp: "2026-08-11T17:49:36+08:00", title: "refactor(live2d): 重构并完善Live2D加载与错误处理流程" },
  { id: "cd08ff4a6e8fa92c558e56da3e693abb0cde05bd", timestamp: "2026-08-11T19:21:16+08:00", title: "feat(live2d): 实现模型切换与表情菜单，重构控制器逻辑" },
  { id: "46b114bc4c8b49deb27e87a15db0eaa52fca9100", timestamp: "2026-08-11T20:51:46+08:00", title: "refactor(music-player): 重构播放器状态与UI布局" },
  { id: "57dc5280f2d516f46fed105b945a5d227606f161", timestamp: "2026-08-12T07:11:05+08:00", title: "refactor(background): 重构背景系统，拆分出BackgroundSystem组件" },
  { id: "4b2c58d8ea699b9a2327cd8c4a06ec35c24574dc", timestamp: "2026-08-12T07:53:33+08:00", title: "feat: 实现本地音乐文件播放功能，支持目录扫描与流式传输" },
  { id: "656d196a1c5b9e71e3f8f30e278f71c5a8921593", timestamp: "2026-08-12T09:46:36+08:00", title: "style(layout,footer,live2d): 重构布局与页脚结构，优化悬浮元素适配" },
  { id: "9404861bc001282a7888bb9be731fd7cf62de2a1", timestamp: "2026-08-12T12:32:22+08:00", title: "feat: 完成v1.0.0正式版本的功能迭代与优化" },
  { id: "f1ca51a155b9ab03f89c2dec99c747911364919c", timestamp: "2026-08-13T08:14:56+08:00", title: "feat: 完成音乐播放后端与前端完整重构" },
  { id: "46d7141a2aad387bda5652318ba81200c4763fb5", timestamp: "2026-08-13T09:01:59+08:00", title: "修复go后端 启动慢 无法获取music list bug" },
  { id: "d1668b7769fe8419c22d5eb4258b58f5f2c25dd4", timestamp: "2026-08-13T11:38:21+08:00", title: "chore: 添加本地代理技能和锁文件的gitignore规则" },
  { id: "e7d51915f3c2f539a783429686c5c1efe283aa06", timestamp: "2026-08-14T06:55:21+08:00", title: "feat: 新增平滑滚动、加载动画与视觉优化" },
  { id: "bdcb2485fc5fecbaab60a6780258723dfa6239ce", timestamp: "2026-08-14T07:29:49+08:00", title: "refactor(opening-animation): 重构开启动画逻辑，优化蒙版与资源加载流程" },
  { id: "019b4c08b9f9bee3d0e324bdfce3672d6bfee799", timestamp: "2026-08-14T07:42:48+08:00", title: "feat(temporary-animation): 新增加载错误重试与资源管理逻辑" },
  { id: "95e980722ea7468388946d18d64454091bc14f56", timestamp: "2026-08-14T08:02:23+08:00", title: "chore: 清理无用的临时文件和旧加载图资源" },
  { id: "091222d2dd040ee3a4abb557bf7f244d31d54c6c", timestamp: "2026-08-14T08:04:01+08:00", title: "chore: 修复.gitignore中.vscode目录的忽略规则" },
  { id: "b418e6bc2e760fa227543d7d31e583ce62b71d54", timestamp: "2026-08-14T11:44:07+08:00", title: "refactor: 重构头部导航，新增动画交互与响应式布局" },
  { id: "f1d30ed23fe8188645bc143c82d974f4f32d0eb6", timestamp: "2026-08-14T13:25:12+08:00", title: "feat(live2d): 添加触摸交互对话与滚动进度按钮布局优化" },
  { id: "fd8770e85de06f0b21cb9829ac38d3a2543d73de", timestamp: "2026-08-14T18:26:44+08:00", title: "feat(backend): add legacy MP3 embedded cover extraction and serving support" },
  { id: "9f6b0e522331f898f831d7ed3872ce602ad37068", timestamp: "2026-08-14T19:27:09+08:00", title: "feat: 支持音频分块加载播放并完善响应头" },
  { id: "9e5942e3d3606d5216259343fedc295b4b94eaf6", timestamp: "2026-08-15T11:23:55+08:00", title: "feat: 实现完整后端生命周期与部署工具链" },
  { id: "feead3b961ed4b9ef51cf857f9e698035959992e", timestamp: "2026-08-15T11:44:33+08:00", title: "build(backend): 新增Linux一键启动脚本并优化环境配置文件生成" },
  { id: "48b63b691abf83d3a333a81a6431e65942bb697c", timestamp: "2026-08-15T16:14:31+08:00", title: "chore: 切换为本地托管字体和静态资源，移除外部CDN依赖" },
  { id: "df0f5e159e6dc9f7eed56191558eb2ea093b8b05", timestamp: "2026-08-15T17:07:19+08:00", title: "refactor(homepage): 重构首页布局与样式，新增字体依赖" },
  { id: "95969ffb000e04524c560ac6844f9193c14e6387", timestamp: "2026-08-15T17:24:46+08:00", title: "refactor: 迁移背景资源到CDN并优化加载逻辑" },
  { id: "e60b1bc74f172bc6c0b04291e2bf59afc90ab018", timestamp: "2026-08-15T18:46:03+08:00", title: "refactor(loading): optimize artwork loading logic" },
  { id: "be2ec5979d07e56bbcbce73c898020696de4e5c1", timestamp: "2026-08-15T19:38:27+08:00", title: "feat(background): add parallax hover effect for background images" },
  { id: "2ae77611420c223eec8005c0f61a69bcefbd7714", timestamp: "2026-08-16T11:29:01+08:00", title: "feat: 实现首页英雄视觉区域组件并替换原有占位" },
  { id: "d60ed26bf5c0c4b609c8af859b11c2ecb74c3fb0", timestamp: "2026-08-16T12:07:42+08:00", title: "feat: 新增随机背景接口与前端渲染逻辑" },
  { id: "27d96f0975b2ca3bdb3a438dd36f6b36699a3673", timestamp: "2026-08-16T12:43:25+08:00", title: "feat: 添加HTTP响应状态掩码功能" },
  { id: "4c635af1171ba4f3253d747966a7da051494a4ae", timestamp: "2026-08-17T15:45:52+08:00", title: "refactor: 重构首页布局，拆分并整合首页模块" },
  { id: "34490ffa5eff61fc351e7103f244b43de9f13152", timestamp: "2026-08-17T16:04:38+08:00", title: "refactor: 重构首页布局与路由规划" },
  { id: "bef785871487c57204e48a2ec6d30c94a600565e", timestamp: "2026-08-18T08:50:04+08:00", title: "refactor: 重构首页布局，替换内容预览区块为更新日志区块" },
] as const

const commitDescriptions: Partial<Record<string, string>> = {
  f8432af909886d5b46b7307281828026b14ca1f0:
    '完成通用工具、可复用 UI、全站样式令牌、站点骨架、Header/Footer、动画规范与响应式支持的整体重构。',
}

export const mockUpdates: readonly UpdateDraft[] = commitHistory.map((commit) => ({
  ...commit,
  description: commitDescriptions[commit.id] ?? `提交 ID：${commit.id.slice(0, 7)} · 已收录于 Jiuin 本地开发历史。`,
}))

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
