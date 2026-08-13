import { createRouter, createWebHistory } from 'vue-router'

import SiteCreateView from './views/SiteCreateView.vue'
import SiteEditView from './views/SiteEditView.vue'
import ResourceEditView from './views/ResourceEditView.vue'
import SitesListView from './views/SitesListView.vue'
import GroupsListView from './views/GroupsListView.vue'
import GroupFormView from './views/GroupFormView.vue'
import DashboardView from './views/DashboardView.vue'
import UsersListView from './views/UsersListView.vue'
import UserFormView from './views/UserFormView.vue'
import FilesystemView from './views/FilesystemView.vue'
import ProfileView from './views/ProfileView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/admin/dashboard' },
    { path: '/admin', redirect: '/admin/dashboard' },
    { path: '/admin/dashboard', component: DashboardView },
    { path: '/admin/files', component: FilesystemView },
    { path: '/admin/profile', component: ProfileView },
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
    { path: '/:pathMatch(.*)*', redirect: '/admin/dashboard' },
  ],
})
