// Browser requests always remain on the public origin. Development routing is
// performed by Vite's server-side proxy, while production uses Nginx.
export const apiConfig = {
  baseUrl: '',
} as const
