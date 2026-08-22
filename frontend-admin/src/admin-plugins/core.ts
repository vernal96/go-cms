import { FolderOpened, OfficeBuilding, UserFilled } from '@element-plus/icons-vue'

import FilesystemView from '../views/FilesystemView.vue'
import GroupFormView from '../views/GroupFormView.vue'
import GroupsListView from '../views/GroupsListView.vue'
import ResourceEditView from '../views/ResourceEditView.vue'
import LibraryItemEditView from '../views/LibraryItemEditView.vue'
import SiteCreateView from '../views/SiteCreateView.vue'
import SiteEditView from '../views/SiteEditView.vue'
import SitesListView from '../views/SitesListView.vue'
import UserFormView from '../views/UserFormView.vue'
import UsersListView from '../views/UsersListView.vue'
import type { AdminPlugin } from './plugin'

export const coreAdminPlugin: AdminPlugin = {
  code: 'core',
  icons: {
    sites: OfficeBuilding,
    files: FolderOpened,
    users: UserFilled,
  },
  routes: [
    { name: 'core.files', path: '/admin/files', component: FilesystemView },
    { name: 'core.sites', path: '/admin/sites', component: SitesListView },
    { name: 'core.sites.create', path: '/admin/sites/create', component: SiteCreateView },
    { name: 'core.sites.edit', path: '/admin/sites/:siteId/edit', component: SiteEditView },
    { name: 'core.resources.edit', path: '/admin/sites/:siteId/resources/:resourceId/edit', component: ResourceEditView },
    { name: 'core.library-items.create', path: '/admin/sites/:siteId/resources/:resourceId/items/new', component: LibraryItemEditView },
    { name: 'core.library-items.edit', path: '/admin/sites/:siteId/resources/:resourceId/items/:itemId/edit', component: LibraryItemEditView },
    { name: 'core.users', path: '/admin/users', component: UsersListView },
    { name: 'core.users.create', path: '/admin/users/create', component: UserFormView },
    { name: 'core.users.edit', path: '/admin/users/:userId/edit', component: UserFormView },
    { name: 'core.groups', path: '/admin/groups', component: GroupsListView },
    { name: 'core.groups.create', path: '/admin/groups/create', component: GroupFormView },
    { name: 'core.groups.edit', path: '/admin/groups/:groupId/edit', component: GroupFormView },
  ],
}
