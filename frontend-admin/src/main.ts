import { createApp } from 'vue'
import { ElLoading } from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'

import { adminPluginRegistry } from './admin-plugins'
import { adminPluginRegistryKey } from './admin-plugins/context'

import App from './App.vue'
import { router } from './router'
import './styles.css'

createApp(App).provide(adminPluginRegistryKey, adminPluginRegistry).use(router).use(ElLoading).mount('#app')
