const fallbackMediaOrigin = 'https://jiuin-media.invalid'
const musicMediaPathPrefix = '/media/music/'

function parseTrustedApiBase(apiBaseUrl: string) {
  if (!apiBaseUrl) return undefined

  try {
    const base = new URL(apiBaseUrl)
    if ((base.protocol !== 'http:' && base.protocol !== 'https:') || base.username || base.password) {
      return undefined
    }
    return base
  } catch {
    return undefined
  }
}

/** Resolve only public music assets served by this API deployment. */
export function resolveMusicMediaUrl(sourceUrl: string | undefined, apiBaseUrl: string, pageProtocol: string) {
  const source = sourceUrl?.trim()
  if (!source || source.startsWith('//')) return undefined

  const isAbsoluteHttp = /^https?:\/\//i.test(source)
  if (!isAbsoluteHttp && !source.startsWith('/')) return undefined

  const apiBase = parseTrustedApiBase(apiBaseUrl)
  if ((apiBaseUrl && !apiBase) || (isAbsoluteHttp && !apiBase)) return undefined

  try {
    const resolved = new URL(source, apiBase ?? fallbackMediaOrigin)
    if (
      (resolved.protocol !== 'http:' && resolved.protocol !== 'https:')
      || resolved.username
      || resolved.password
      || resolved.search
      || resolved.hash
      || !resolved.pathname.startsWith(musicMediaPathPrefix)
    ) return undefined

    if (apiBase && resolved.origin !== apiBase.origin) return undefined
    if (pageProtocol === 'https:' && resolved.protocol !== 'https:') return undefined

    return apiBase ? resolved.toString() : resolved.pathname
  } catch {
    return undefined
  }
}
