const configuredBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

/**
 * 本地开发默认连接同机 Go 服务，避免播放器因遗漏 .env 而静默为空。
 * 生产环境未配置时保留空基础地址，以便通过同源 /api 与 /media 反向代理访问后端。
 */
export const apiConfig = {
  baseUrl: configuredBaseUrl || (import.meta.env.DEV ? 'http://127.0.0.1:8080' : ''),
} as const
