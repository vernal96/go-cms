import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import manifest from './package.json'

const shared = Object.keys(manifest.peerDependencies)
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist-sdk',
    lib: { entry: 'src/sdk.ts', formats: ['es'], fileName: 'sdk', cssFileName: 'sdk' },
    rollupOptions: {
      // Inline TinyMCE skin strings are assets; runtime libraries remain peers.
      external: id => !id.includes('?inline') && shared.some(name => id === name || id.startsWith(`${name}/`)),
    },
  },
})
