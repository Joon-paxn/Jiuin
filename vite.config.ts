import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    // Production source maps are useful to a trusted error-monitoring system,
    // but this public static deployment does not upload them anywhere.
    sourcemap: false,
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    allowedHosts: ['jiuin.cn', 'www.jiuin.cn'],
    // Development-only reverse proxy. This config is not bundled into browser
    // code, so API, media, and WebSocket calls stay origin-relative.
    proxy: {
      '/api': { target: 'http://127.0.0.1:8080', changeOrigin: false },
      '/media': { target: 'http://127.0.0.1:8080', changeOrigin: false },
      '/ws': { target: 'ws://127.0.0.1:8080', ws: true, changeOrigin: false },
      '/health': { target: 'http://127.0.0.1:8080', changeOrigin: false },
      '/ready': { target: 'http://127.0.0.1:8080', changeOrigin: false },
    },
  },
})
