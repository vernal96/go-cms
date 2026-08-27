import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  test: {
    setupFiles: ['./src/test/setup.ts'],
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    strictPort: true,
    proxy: {
      '/api': {
        target:
          process.env.ADMIN_API_TARGET ??
          'http://host.docker.internal:8080',
        changeOrigin: false,
      },
    },
  },
})
