import { defineConfig } from 'vite'
import { fileURLToPath } from 'node:url'
export default defineConfig({root:fileURLToPath(new URL('.',import.meta.url)),resolve:{dedupe:['vue','vue-router','element-plus']},define:{__VUE_OPTIONS_API__:true,__VUE_PROD_DEVTOOLS__:false,__VUE_PROD_HYDRATION_MISMATCH_DETAILS__:false},server:{host:'127.0.0.1',port:18082,strictPort:true,proxy:{'/api':'http://127.0.0.1:18081'}}})
