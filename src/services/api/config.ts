/**
 * Go 服务接入预留：在部署环境中由构建变量注入，当前阶段不发起业务请求。
 */
export const apiConfig = {
  baseUrl: (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, ''),
} as const
