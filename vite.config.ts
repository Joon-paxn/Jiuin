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
  },
})
