import { createRouter, createWebHistory } from 'vue-router'

import { adminPluginRegistry } from './admin-plugins'
import { shellRoutes } from './shell-routes'
import DashboardView from './views/DashboardView.vue'
import ProfileView from './views/ProfileView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/admin/dashboard' },
    { ...shellRoutes.root, redirect: shellRoutes.dashboard.path },
    { ...shellRoutes.dashboard, component: DashboardView },
    { ...shellRoutes.profile, component: ProfileView },
    ...adminPluginRegistry.routeRecords(),
    { path: '/:pathMatch(.*)*', redirect: '/admin/dashboard' },
  ],
})
