import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
export default defineConfig({plugins:[vue()],build:{outDir:'dist',lib:{entry:'src/index.ts',formats:['es'],fileName:'notice'},rollupOptions:{external:['vue','@go-cms/admin/sdk']}}})
