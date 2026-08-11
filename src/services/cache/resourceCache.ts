/** Registers the browser-managed runtime cache only in production builds. */
export function registerResourceCache() {
  if (!import.meta.env.PROD || !('serviceWorker' in navigator)) {
    return
  }

  const basePath = import.meta.env.BASE_URL.endsWith('/')
    ? import.meta.env.BASE_URL
    : `${import.meta.env.BASE_URL}/`

  void navigator.serviceWorker.register(`${basePath}service-worker.js`, { scope: basePath })
    .catch((error: unknown) => console.warn('Jiuin resource cache registration failed.', error))
}
