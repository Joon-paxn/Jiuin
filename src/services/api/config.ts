const configuredBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

/**
 * 本地开发默认连接同机 Go 服务，避免播放器因遗漏 .env 而静默为空。
 * 生产构建固定使用同源 /api 与 /media 反向代理，使 CSP 与媒体信任边界保持一致。
 */
export const apiConfig = {
  baseUrl: import.meta.env.DEV ? configuredBaseUrl || 'http://127.0.0.1:8080' : '',
} as const
