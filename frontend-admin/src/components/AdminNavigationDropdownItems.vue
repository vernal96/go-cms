<script setup lang="ts">
import { ElDropdownItem, ElIcon } from 'element-plus'
import { Menu } from '@element-plus/icons-vue'

import { adminPluginRegistry } from '../admin-plugins'
import type { ResolvedAdminNavigationItem } from '../composables/use-admin-navigation'

defineOptions({ name: 'AdminNavigationDropdownItems' })

defineProps<{
  items: readonly ResolvedAdminNavigationItem[]
  activeRoute: string
}>()

const emit = defineEmits<{ select: [route: string] }>()

function icon(code?: string) {
  return code ? (adminPluginRegistry.icon(code) ?? Menu) : Menu
}
</script>

<template>
  <template v-for="item in items" :key="item.code">
    <div
      v-if="item.children.length > 0"
      class="admin-navigation-nested-group"
      :data-navigation-code="item.code"
    >
      <div class="admin-navigation-nested-title">
        <el-icon><component :is="icon(item.icon)" /></el-icon>
        <span>{{ item.label }}</span>
      </div>
      <div class="admin-navigation-nested-items">
        <admin-navigation-dropdown-items
          :items="item.children"
          :active-route="activeRoute"
          @select="emit('select', $event)"
        />
      </div>
    </div>
    <el-dropdown-item
      v-else-if="item.route"
      :command="item.route"
      :class="{ 'is-active': activeRoute === item.route }"
      :data-navigation-code="item.code"
      @click="emit('select', item.route)"
    >
      <el-icon><component :is="icon(item.icon)" /></el-icon>
      <span>{{ item.label }}</span>
    </el-dropdown-item>
  </template>
</template>
