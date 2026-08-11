import { apiConfig } from './config'
import type { ApiResponse } from './types'

export class ApiClientError extends Error {
  constructor(message: string, public readonly code?: number) {
    super(message)
    this.name = 'ApiClientError'
  }
}

function getApiUrl(path: string) {
  return `${apiConfig.baseUrl}${path}`
}

export async function get<T>(path: string): Promise<T> {
  const response = await fetch(getApiUrl(path), {
    headers: { Accept: 'application/json' },
  })
  const body = await response.json() as ApiResponse<T>

  if (!response.ok || body.code !== 200 || body.data === undefined) {
    throw new ApiClientError(body.message || 'API request failed', body.code)
  }

  return body.data
}
