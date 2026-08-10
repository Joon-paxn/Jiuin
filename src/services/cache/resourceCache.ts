/** Registers the browser-managed runtime cache only in production builds. */
export function registerResourceCache() {
  if (!import.meta.env.PROD || !('serviceWorker' in navigator)) {
    return
  }

  void navigator.serviceWorker.register('/service-worker.js', { scope: '/' })
    .catch((error: unknown) => console.warn('Jiuin resource cache registration failed.', error))
}
