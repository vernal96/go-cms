import { projectName } from './project'
import { createApp } from 'vue'
import { ElLoading } from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'

import { adminPluginRegistry } from './admin-plugins'
import { adminPluginRegistryKey } from './admin-plugins/context'

import App from './App.vue'
import { router } from './router'
import './styles.css'

document.title = `${projectName} — Администрирование`
document.querySelector('meta[name="description"]')?.setAttribute('content', `Панель управления сайтом ${projectName}`)

createApp(App).provide(adminPluginRegistryKey, adminPluginRegistry).use(router).use(ElLoading).mount('#app')
