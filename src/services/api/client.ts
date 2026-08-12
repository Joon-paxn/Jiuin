import { apiConfig } from './config'
import type { ApiResponse } from './types'

const genericRequestError = '暂时无法连接服务，请稍后再试。'
const genericResponseError = '服务响应异常，请稍后再试。'

export class ApiClientError extends Error {
  constructor(message: string, public readonly code?: number) {
    super(message)
    this.name = 'ApiClientError'
  }
}

function getApiUrl(path: string) {
  return `${apiConfig.baseUrl}${path}`
}

function isApiResponse(value: unknown): value is ApiResponse<unknown> {
  return typeof value === 'object'
    && value !== null
    && typeof (value as { code?: unknown }).code === 'number'
    && typeof (value as { message?: unknown }).message === 'string'
}

async function readApiResponse(response: Response): Promise<ApiResponse<unknown> | undefined> {
  const contentType = response.headers.get('content-type') ?? ''
  if (!contentType.toLowerCase().includes('application/json')) {
    return undefined
  }

  try {
    const body = await response.json() as unknown
    return isApiResponse(body) ? body : undefined
  } catch {
    return undefined
  }
}

export async function get<T>(path: string): Promise<T> {
  let response: Response
  try {
    response = await fetch(getApiUrl(path), {
      headers: { Accept: 'application/json' },
    })
  } catch {
    throw new ApiClientError(genericRequestError)
  }

  const body = await readApiResponse(response)

  if (!response.ok || !body || body.code !== 200 || body.data === undefined) {
    // API error text is intentionally not reflected to the UI. It may come
    // from a reverse proxy or a future service and is not user-facing data.
    throw new ApiClientError(genericResponseError, body?.code ?? response.status)
  }

  return body.data as T
}
