/** Registers the browser-managed runtime cache only in production builds. */
export function registerResourceCache() {
  if (!import.meta.env.PROD || !('serviceWorker' in navigator)) {
    return
  }

  const basePath = import.meta.env.BASE_URL.endsWith('/')
    ? import.meta.env.BASE_URL
    : `${import.meta.env.BASE_URL}/`

  void navigator.serviceWorker.register(`${basePath}service-worker.js`, { scope: basePath })
    .catch(() => {
      // The cache is an optional enhancement. Do not expose browser/runtime
      // diagnostics in production logs; development remains inspectable.
      if (import.meta.env.DEV) {
        console.warn('Jiuin resource cache registration failed.')
      }
    })
}
