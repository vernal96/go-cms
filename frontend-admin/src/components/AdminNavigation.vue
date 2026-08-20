<script setup lang="ts">
import { computed } from 'vue'
import { ArrowDown, Menu } from '@element-plus/icons-vue'
import {
  ElButton,
  ElDropdown,
  ElDropdownMenu,
  ElIcon,
} from 'element-plus'
import { RouterLink, useRoute, useRouter } from 'vue-router'

import { adminPluginRegistry } from '../admin-plugins'
import type { ResolvedAdminNavigationItem } from '../composables/use-admin-navigation'
import AdminNavigationDropdownItems from './AdminNavigationDropdownItems.vue'

defineOptions({ name: 'AdminNavigation' })

const props = defineProps<{
  items: readonly ResolvedAdminNavigationItem[]
}>()

const route = useRoute()
const router = useRouter()
const activeRoute = computed(() => findActiveRoute(props.items, route.path))

function findActiveRoute(
  items: readonly ResolvedAdminNavigationItem[],
  currentPath: string,
): string {
  let active = ''
  let activeLength = -1
  const visit = (current: readonly ResolvedAdminNavigationItem[]) => {
    for (const item of current) {
      if (item.route) {
        const definition = adminPluginRegistry.route(item.route)
        const target = definition?.path ?? ''
        if (
          target !== '' &&
          (currentPath === target || currentPath.startsWith(`${target}/`)) &&
          target.length > activeLength
        ) {
          active = item.route
          activeLength = target.length
        }
      }
      visit(item.children)
    }
  }
  visit(items)
  return active
}

function itemActive(item: ResolvedAdminNavigationItem): boolean {
  if (item.route === activeRoute.value) return true
  return item.children.some(itemActive)
}

function icon(code?: string) {
  return code ? (adminPluginRegistry.icon(code) ?? Menu) : Menu
}

function navigate(routeName: string): void {
  if (!adminPluginRegistry.hasRoute(routeName)) {
    console.warn(`[GO CMS admin] Cannot navigate to unavailable semantic route "${routeName}".`)
    return
  }
  void router.push({ name: routeName })
}
</script>

<template>
  <nav class="topbar-menu" aria-label="Главное меню">
    <template v-for="item in items" :key="item.code">
      <router-link
        v-if="item.children.length === 0 && item.route"
        :to="{ name: item.route }"
        class="topbar-link"
        :class="{ 'is-active': itemActive(item) }"
        :data-navigation-code="item.code"
      >
        <el-icon><component :is="icon(item.icon)" /></el-icon>
        <span>{{ item.label }}</span>
      </router-link>
      <el-dropdown
        v-else-if="item.children.length > 0"
        trigger="click"
        :data-navigation-code="item.code"
      >
        <el-button
          text
          class="topbar-link admin-navigation-trigger"
          :class="{ 'is-active': itemActive(item) }"
        >
          <el-icon><component :is="icon(item.icon)" /></el-icon>
          <span>{{ item.label }}</span>
          <el-icon><ArrowDown /></el-icon>
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <admin-navigation-dropdown-items
              :items="item.children"
              :active-route="activeRoute"
              @select="navigate"
            />
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </template>
  </nav>
</template>
