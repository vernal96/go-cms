import { createRouter, createWebHistory } from 'vue-router'

import { adminPluginRegistry } from './admin-plugins'
import DashboardView from './views/DashboardView.vue'
import ProfileView from './views/ProfileView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/admin/dashboard' },
    { path: '/admin', redirect: '/admin/dashboard' },
    { name: 'shell.dashboard', path: '/admin/dashboard', component: DashboardView },
    { name: 'shell.profile', path: '/admin/profile', component: ProfileView },
    ...adminPluginRegistry.routeRecords(),
    { path: '/:pathMatch(.*)*', redirect: '/admin/dashboard' },
  ],
})
