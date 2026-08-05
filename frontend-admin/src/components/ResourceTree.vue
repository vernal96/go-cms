<script setup lang="ts">
import { computed, ref, watch, type Component } from 'vue'
import { ElButton, ElEmpty, ElIcon, ElMessage, ElTree } from 'element-plus'
import type Node from 'element-plus/es/components/tree/src/model/node'
import type { LoadFunction } from 'element-plus/es/components/tree/src/tree.type'
import { Document, Folder, Link, Plus, Tickets } from '@element-plus/icons-vue'

import { adminRequest } from '../api/admin-api'
import { useSelectedSite } from '../composables/use-selected-site'
import type { ResourceChildrenResponse, ResourceTreeItem } from '../types/admin'
import ResourceCreateDialog from './ResourceCreateDialog.vue'

const props = defineProps<{ accessToken: string; canCreate: boolean }>()
const emit = defineEmits<{ error: [error: unknown] }>()
const selected = useSelectedSite()
const siteId = computed(() => selected.selectedSite.value?.id ?? null)
const treeKey = ref(0)
const treeRef = ref<InstanceType<typeof ElTree> | null>(null)
const dialogRef = ref<{ open(parent: ResourceTreeItem | null): Promise<void> } | null>(null)
const rootError = ref(false)

const treeProps = { label: 'display_title', children: 'children', isLeaf: 'isLeaf' }
type TreeNodeData = ResourceTreeItem & {
  isLeaf: boolean
  loadError?: boolean
  retryParentId?: number
}

const loadNode: LoadFunction = async (node, resolve) => {
  if (siteId.value === null) {
    resolve([])
    return
  }
  const data = node.data as TreeNodeData | undefined
  const parentId = node.level === 0 ? null : data?.id ?? null
  const query = parentId === null ? '' : `?parent_id=${parentId}`
  try {
    const response = await adminRequest<ResourceChildrenResponse>(
      `/api/admin/sites/${siteId.value}/resources${query}`,
      props.accessToken,
    )
    rootError.value = false
    resolve(response.items.map((item) => ({ ...item, isLeaf: !item.has_children })))
  } catch (error) {
    if (node.level === 0) rootError.value = true
    if (node.level === 0 || parentId === null) resolve([])
    else resolve([{
      id: -parentId,
      parent_id: parentId,
      template_code: null,
      icon: 'document',
      title: 'Не удалось загрузить дочерние ресурсы',
      menu_title: '',
      display_title: 'Не удалось загрузить дочерние ресурсы',
      has_children: false,
      can_create_child: false,
      isLeaf: true,
      loadError: true,
      retryParentId: parentId,
    }])
    emit('error', error)
  }
}

function icon(name: string): Component {
  if (name === 'link') return Link
  if (name === 'folder') return Folder
  if (name === 'tickets') return Tickets
  return Document
}

function create(parent: ResourceTreeItem | null): void {
  dialogRef.value?.open(parent)
}

function reloadNode(node: Node): void {
  const data = node.data as TreeNodeData
  data.has_children = true
  data.isLeaf = false
  node.isLeaf = false
  node.loaded = false
  node.childNodes = []
  node.loadData(() => node.expand())
}

function retryNode(parentId: number): void {
  const node = treeRef.value?.getNode(parentId) as Node | undefined
  if (node) reloadNode(node)
}

function handleCreated(_item: ResourceTreeItem, parentId: number | null): void {
  ElMessage.success('Ресурс создан')
  if (parentId === null) {
    treeKey.value += 1
    return
  }
  const node = treeRef.value?.getNode(parentId) as Node | undefined
  if (!node) {
    treeKey.value += 1
    return
  }
  reloadNode(node)
}

function retryRoot(): void {
  rootError.value = false
  treeKey.value += 1
}

watch(siteId, () => {
  rootError.value = false
  treeKey.value += 1
})
</script>

<template>
  <section class="resource-panel">
    <div class="sidebar-heading-row">
      <span class="sidebar-heading">Ресурсы</span>
      <el-button
        v-if="siteId && canCreate"
        text
        :icon="Plus"
        aria-label="Создать корневой ресурс"
        @click="create(null)"
      />
    </div>
    <el-empty v-if="!siteId" :image-size="58" description="Сначала выберите сайт" />
    <div v-else-if="rootError" class="tree-error">
      <span>Не удалось загрузить дерево</span>
      <el-button size="small" @click="retryRoot">Повторить</el-button>
    </div>
    <el-tree
      v-else
      :key="`${siteId}-${treeKey}`"
      ref="treeRef"
      class="resource-tree"
      node-key="id"
      lazy
      :load="loadNode"
      :props="treeProps"
      highlight-current
      :expand-on-click-node="false"
      empty-text="Ресурсов пока нет"
    >
      <template #default="{ data }">
        <span v-if="data.loadError" class="resource-node resource-node-error">
          <span class="resource-node-title">{{ data.display_title }}</span>
          <el-button size="small" text @click.stop="retryNode(data.retryParentId)">Повторить</el-button>
        </span>
        <span v-else class="resource-node">
          <el-icon class="resource-node-icon"><component :is="icon(data.icon)" /></el-icon>
          <span class="resource-node-title">{{ data.display_title }}</span>
          <el-button
            v-if="canCreate && data.can_create_child"
            class="resource-node-add"
            text
            size="small"
            :icon="Plus"
            aria-label="Создать дочерний ресурс"
            @click.stop="create(data)"
          />
        </span>
      </template>
    </el-tree>
    <resource-create-dialog
      v-if="siteId"
      ref="dialogRef"
      :access-token="accessToken"
      :site-id="siteId"
      @created="handleCreated"
      @error="emit('error', $event)"
    />
  </section>
</template>
