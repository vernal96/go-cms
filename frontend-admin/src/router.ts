import { createRouter, createWebHistory } from 'vue-router'

import SiteCreateView from './views/SiteCreateView.vue'
import SiteEditView from './views/SiteEditView.vue'
import ResourceEditView from './views/ResourceEditView.vue'
import SitesListView from './views/SitesListView.vue'
import GroupsListView from './views/GroupsListView.vue'
import GroupFormView from './views/GroupFormView.vue'
import UsersListView from './views/UsersListView.vue'
import UserFormView from './views/UserFormView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/admin/sites' },
    { path: '/admin', redirect: '/admin/sites' },
    { path: '/admin/sites', component: SitesListView },
    { path: '/admin/sites/create', component: SiteCreateView },
    { path: '/admin/sites/:siteId/edit', component: SiteEditView },
    { path: '/admin/users', component: UsersListView },
    { path: '/admin/users/create', component: UserFormView },
    { path: '/admin/users/:userId/edit', component: UserFormView },
    { path: '/admin/groups', component: GroupsListView },
    { path: '/admin/groups/create', component: GroupFormView },
    { path: '/admin/groups/:groupId/edit', component: GroupFormView },
    {
      path: '/admin/sites/:siteId/resources/:resourceId/edit',
      component: ResourceEditView,
    },
    { path: '/:pathMatch(.*)*', redirect: '/admin/sites' },
  ],
})
