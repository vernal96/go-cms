import { createRouter, createWebHistory } from 'vue-router'

import SiteCreateView from './views/SiteCreateView.vue'
import SiteEditView from './views/SiteEditView.vue'
import ResourceEditView from './views/ResourceEditView.vue'
import SitesListView from './views/SitesListView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/admin/sites' },
    { path: '/admin', redirect: '/admin/sites' },
    { path: '/admin/sites', component: SitesListView },
    { path: '/admin/sites/create', component: SiteCreateView },
    { path: '/admin/sites/:siteId/edit', component: SiteEditView },
    {
      path: '/admin/sites/:siteId/resources/:resourceId/edit',
      component: ResourceEditView,
    },
    { path: '/:pathMatch(.*)*', redirect: '/admin/sites' },
  ],
})
