const maximumExternalUrlLength = 2_048

/**
 * Returns a normalized public HTTPS URL or nothing.  The API validates its
 * configured links too, but this keeps an unexpected response from becoming
 * an executable browser navigation.
 */
export function toSafeExternalUrl(value: unknown) {
  if (typeof value !== 'string') {
    return undefined
  }

  const candidate = value.trim()
  if (!candidate || candidate.length > maximumExternalUrlLength) {
    return undefined
  }

  try {
    const url = new URL(candidate)
    if (
      url.protocol !== 'https:'
      || !url.hostname
      || url.username
      || url.password
    ) {
      return undefined
    }

    return url.toString()
  } catch {
    return undefined
  }
}
