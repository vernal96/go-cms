<script setup lang="ts">
import { computed, ref, type Component } from 'vue'
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
  ElInput,
  ElMain,
  ElScrollbar,
  ElTree,
} from 'element-plus'
import {
  ArrowDown,
  Collection,
  Document,
  Link,
  Platform,
  Search,
  UserFilled,
} from '@element-plus/icons-vue'

import { resourceTree } from '../mocks/resources'
import type { ResourceKind } from '../types/resource'
import { filterResourceTree } from '../utils/filter-resource-tree'

defineProps<{
  displayName: string
}>()

const emit = defineEmits<{
  logout: []
}>()

const searchQuery = ref('')
const filteredResources = computed(() =>
  filterResourceTree(resourceTree, searchQuery.value),
)

const treeProps = {
  children: 'children',
  label: 'title',
}

function resourceIcon(kind: ResourceKind): Component {
  switch (kind) {
    case 'section':
      return Collection
    case 'link':
      return Link
    case 'page':
      return Document
  }
}

function handleUserCommand(command: string): void {
  if (command === 'logout') {
    emit('logout')
  }
}
</script>

<template>
  <el-container class="admin-shell">
    <el-header class="topbar" height="64px">
      <div class="brand-search">
        <div class="brand-mark" aria-label="Go CMS">
          <el-icon :size="24">
            <Platform />
          </el-icon>
        </div>

        <el-input
          v-model="searchQuery"
          class="resource-search"
          :prefix-icon="Search"
          placeholder="Поиск ресурсов"
          clearable
          aria-label="Поиск ресурсов"
        />
      </div>

      <div class="topbar-main">
        <nav class="topbar-menu" aria-label="Главное меню"></nav>

        <el-dropdown
          placement="bottom-end"
          trigger="click"
          @command="handleUserCommand"
        >
          <el-button
            class="user-control"
            text
            aria-label="Открыть меню пользователя"
          >
            <el-avatar :size="34" :icon="UserFilled" />
            <span class="user-name">{{ displayName }}</span>
            <el-icon class="user-chevron">
              <ArrowDown />
            </el-icon>
          </el-button>

          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item disabled>Профиль</el-dropdown-item>
              <el-dropdown-item command="logout" divided>
                Выйти
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-header>

    <el-container class="admin-body">
      <el-aside class="resource-sidebar" width="320px">
        <div class="sidebar-heading">Ресурсы</div>

        <el-scrollbar class="resource-scrollbar">
          <el-tree
            class="resource-tree"
            :data="filteredResources"
            :props="treeProps"
            node-key="id"
            default-expand-all
            highlight-current
            :expand-on-click-node="false"
            empty-text="Ресурсы не найдены"
          >
            <template #default="{ data }">
              <span class="resource-node">
                <el-icon class="resource-node-icon">
                  <component :is="resourceIcon(data.type)" />
                </el-icon>
                <span class="resource-node-title">{{ data.title }}</span>
              </span>
            </template>
          </el-tree>
        </el-scrollbar>
      </el-aside>

      <el-main class="workspace" aria-label="Рабочая область"></el-main>
    </el-container>
  </el-container>
</template>
