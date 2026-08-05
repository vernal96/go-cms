<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import {
  ElAside,
  ElAvatar,
  ElButton,
  ElContainer,
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElHeader,
  ElIcon,
  ElMain,
  ElScrollbar,
} from 'element-plus'
import { ArrowDown, Platform, UserFilled } from '@element-plus/icons-vue'
import { RouterLink, RouterView } from 'vue-router'

import { AdminAPIError } from '../api/admin-api'
import { useSelectedSite } from '../composables/use-selected-site'
import ResourceTree from './ResourceTree.vue'
import SiteSelector from './SiteSelector.vue'

const props = defineProps<{
  displayName: string
  accessToken: string
  permissions: ReadonlySet<string>
}>()

const emit = defineEmits<{ logout: [] }>()
const selected = useSelectedSite()

function can(code: string): boolean {
  return props.permissions.has(code)
}

function handleUserCommand(command: string): void {
  if (command === 'logout') emit('logout')
}

function handleAPIError(error: unknown): void {
  if (!(error instanceof AdminAPIError)) return
  if (error.status === 401) emit('logout')
  else if (error.status === 403 || error.status === 404) selected.clearSelected()
}

onMounted(() => {
  void selected.initialize(props.accessToken).catch(handleAPIError)
})
onBeforeUnmount(selected.reset)
</script>

<template>
  <el-container class="admin-shell">
    <el-header class="topbar" height="64px">
      <div class="brand-search">
        <div class="brand-mark" aria-label="Go CMS">
          <el-icon :size="24"><Platform /></el-icon>
        </div>
        <site-selector
          v-if="can('core.site.read')"
          :access-token="accessToken"
          :can-create="can('core.site.create')"
          @error="handleAPIError"
        />
      </div>

      <div class="topbar-main">
        <nav class="topbar-menu" aria-label="Главное меню">
          <router-link v-if="can('core.site.read')" to="/admin/sites" class="topbar-link">
            Сайты
          </router-link>
        </nav>
        <el-dropdown placement="bottom-end" trigger="click" @command="handleUserCommand">
          <el-button class="user-control" text aria-label="Открыть меню пользователя">
            <el-avatar :size="34" :icon="UserFilled" />
            <span class="user-name">{{ displayName }}</span>
            <el-icon class="user-chevron"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item disabled>Профиль</el-dropdown-item>
              <el-dropdown-item command="logout" divided>Выйти</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-header>

    <el-container class="admin-body">
      <el-aside class="resource-sidebar" width="320px">
        <el-scrollbar class="resource-scrollbar">
          <resource-tree
            v-if="can('core.resource.read')"
            :access-token="accessToken"
            :can-create="can('core.resource.create')"
            @error="handleAPIError"
          />
          <div v-else class="sidebar-empty">Нет доступа к ресурсам</div>
        </el-scrollbar>
      </el-aside>

      <el-main class="workspace" aria-label="Рабочая область">
        <router-view v-slot="{ Component }">
          <component
            :is="Component"
            :access-token="accessToken"
            :permissions="permissions"
            @unauthorized="emit('logout')"
          />
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>
