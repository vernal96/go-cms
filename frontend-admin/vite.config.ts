import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: { alias: [{ find: /^@go-cms\/admin\/sdk$/, replacement: fileURLToPath(new URL('./src/sdk.ts', import.meta.url)) }], dedupe: ['vue', 'vue-router', 'element-plus'] },
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
