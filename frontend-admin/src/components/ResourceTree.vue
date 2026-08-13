<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch, type Component } from 'vue'
import { ElButton, ElEmpty, ElIcon, ElMessage, ElMessageBox, ElTree } from 'element-plus'
import type Node from 'element-plus/es/components/tree/src/model/node'
import type { LoadFunction } from 'element-plus/es/components/tree/src/tree.type'
import { Document, Folder, Link, Plus, Tickets } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'

import { adminRequest, adminRequestVoid } from '../api/admin-api'
import { useSelectedSite } from '../composables/use-selected-site'
import type { ResourceChildrenResponse, ResourceTreeItem } from '../types/admin'
import ResourceCreateDialog from './ResourceCreateDialog.vue'

const props = withDefaults(defineProps<{
  accessToken: string
  canCreate: boolean
  canUpdate?: boolean
  canDelete?: boolean
}>(), { canUpdate: false, canDelete: false })
const emit = defineEmits<{ error: [error: unknown] }>()
const selected = useSelectedSite()
const router = useRouter()
const siteId = computed(() => selected.selectedSite.value?.id ?? null)
const treeKey = ref(0)
const treeRef = ref<InstanceType<typeof ElTree> | null>(null)
const rootError = ref(false)
const moving = ref(false)
const expandedIDs = new Set<number>()
const context = ref<{ item: TreeNodeData; x: number; y: number } | null>(null)
const dialogRef = ref<{ open(parent: ResourceTreeItem | null): Promise<void> } | null>(null)
let removeRouteListener: (() => void) | null = null

const treeProps = { label: 'display_title', children: 'children', isLeaf: 'isLeaf' }
type TreeNodeData = ResourceTreeItem & {
  isLeaf: boolean
  loadError?: boolean
  retryParentId?: number
}

const loadNode: LoadFunction = async (node, resolve) => {
  if (siteId.value === null) { resolve([]); return }
  const data = node.data as TreeNodeData | undefined
  const parentId = node.level === 0 ? null : (data?.id ?? null)
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
    else resolve([errorNode(parentId)])
    emit('error', error)
  }
}

function errorNode(parentId: number): TreeNodeData {
  return {
    id: -parentId, parent_id: parentId, template_code: null, icon: 'document',
    title: 'Не удалось загрузить дочерние ресурсы', menu_title: '',
    display_title: 'Не удалось загрузить дочерние ресурсы', sort: 0,
    deleted: false, deleted_at: null, has_children: false, can_create_child: false,
    isLeaf: true, loadError: true, retryParentId: parentId,
  }
}

function icon(name: string): Component {
  if (name === 'link') return Link
  if (name === 'folder') return Folder
  if (name === 'tickets') return Tickets
  return Document
}

function create(parent: ResourceTreeItem | null): void { dialogRef.value?.open(parent) }

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

function handleCreated(item: ResourceTreeItem, parentId: number | null): void {
  ElMessage.success('Ресурс создан')
  if (parentId === null) reloadTree()
  else {
    const node = treeRef.value?.getNode(parentId) as Node | undefined
    if (node) reloadNode(node)
    else reloadTree()
  }
  void router.push(`/admin/sites/${siteId.value}/resources/${item.id}/edit`)
}

function openEditor(item: TreeNodeData): void {
  if (item.loadError || siteId.value === null) return
  void router.push(`/admin/sites/${siteId.value}/resources/${item.id}/edit`)
}

function openContextMenu(event: MouseEvent, item: TreeNodeData): void {
  if (item.loadError) return
  event.preventDefault()
  context.value = {
    item,
    x: Math.min(event.clientX, window.innerWidth - 230),
    y: Math.min(event.clientY, window.innerHeight - 250),
  }
}

function closeContextMenu(): void { context.value = null }

async function softDelete(item: TreeNodeData): Promise<void> {
  closeContextMenu()
  try {
    await ElMessageBox.confirm(
      'Ресурс и всё его поддерево останутся в дереве, но будут сняты с публикации.',
      `Удалить «${item.display_title}»?`,
      { type: 'warning', confirmButtonText: 'Удалить', cancelButtonText: 'Отмена' },
    )
    await adminRequestVoid(`/api/admin/sites/${siteId.value}/resources/${item.id}`, props.accessToken, { method: 'DELETE' })
    ElMessage.success('Ресурс удалён')
    notifyChanged()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    emit('error', error)
    ElMessage.error(error instanceof Error ? error.message : 'Не удалось удалить ресурс')
  }
}

async function restore(item: TreeNodeData, withDescendants: boolean): Promise<void> {
  closeContextMenu()
  try {
    await adminRequestVoid(`/api/admin/sites/${siteId.value}/resources/${item.id}/restore`, props.accessToken, {
      method: 'POST', body: JSON.stringify({ with_descendants: withDescendants }),
    })
    ElMessage.success(withDescendants ? 'Поддерево восстановлено' : 'Ресурс восстановлен')
    notifyChanged()
  } catch (error) {
    emit('error', error)
    ElMessage.error(error instanceof Error ? error.message : 'Не удалось восстановить ресурс')
  }
}

async function permanentDelete(item: TreeNodeData): Promise<void> {
  closeContextMenu()
  try {
    await ElMessageBox.confirm(
      'Ресурс и всё его поддерево будут удалены без возможности восстановления.',
      `Удалить «${item.display_title}» окончательно?`,
      { type: 'error', confirmButtonText: 'Удалить окончательно', cancelButtonText: 'Отмена' },
    )
    await adminRequestVoid(`/api/admin/sites/${siteId.value}/resources/${item.id}/permanent`, props.accessToken, { method: 'DELETE' })
    if (Number(router.currentRoute?.value.params.resourceId) === item.id) void router.push('/admin/dashboard')
    ElMessage.success('Ресурс удалён окончательно')
    notifyChanged()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    emit('error', error)
    ElMessage.error(error instanceof Error ? error.message : 'Не удалось удалить ресурс')
  }
}

function allowDrag(node: Node): boolean {
  const item = node.data as TreeNodeData
  return props.canUpdate && !item.deleted && !item.loadError && !moving.value
}

function allowDrop(dragging: Node, drop: Node, type: string): boolean {
  const source = dragging.data as TreeNodeData
  const target = drop.data as TreeNodeData
  if (!props.canUpdate || source.deleted || target.deleted || target.loadError) return false
  return type !== 'inner' || source.id !== target.id
}

async function handleDrop(dragging: Node, drop: Node, type: 'before' | 'after' | 'inner'): Promise<void> {
  if (!siteId.value) return
  const source = dragging.data as TreeNodeData
  const target = drop.data as TreeNodeData
  let parentId: number | null
  let position: number
  if (type === 'inner') {
    parentId = target.id
    position = 2_147_483_647
  } else {
		const parent = drop.parent
		if (!parent) return
		parentId = drop.level === 1 ? null : ((parent.data as TreeNodeData | undefined)?.id ?? null)
		const siblings = parent.childNodes
      .map((node) => node.data as TreeNodeData)
      .filter((item) => item.id !== source.id)
    const targetIndex = siblings.findIndex((item) => item.id === target.id)
    position = Math.max(0, targetIndex + (type === 'after' ? 1 : 0))
  }
  moving.value = true
  try {
    await adminRequest(`/api/admin/sites/${siteId.value}/resources/${source.id}/move`, props.accessToken, {
      method: 'POST', body: JSON.stringify({ parent_id: parentId, position }),
    })
    ElMessage.success('Ресурс перемещён')
    notifyChanged()
  } catch (error) {
    emit('error', error)
    ElMessage.error(error instanceof Error ? error.message : 'Не удалось переместить ресурс')
    reloadTree()
  } finally {
    moving.value = false
  }
}

function rememberExpanded(item: TreeNodeData): void { expandedIDs.add(item.id) }
function forgetExpanded(item: TreeNodeData): void { expandedIDs.delete(item.id) }

function reloadTree(): void {
  rootError.value = false
  closeContextMenu()
  treeKey.value += 1
}

function notifyChanged(): void {
  window.dispatchEvent(new CustomEvent('admin:resource-tree-changed', { detail: { siteId: siteId.value } }))
}

function handleTreeChanged(event: Event): void {
  const changedSite = (event as CustomEvent<{ siteId?: number }>).detail?.siteId
  if (changedSite === undefined || changedSite === siteId.value) reloadTree()
}

function handleEscape(event: KeyboardEvent): void { if (event.key === 'Escape') closeContextMenu() }

onMounted(() => {
  document.addEventListener('pointerdown', closeContextMenu)
  document.addEventListener('keydown', handleEscape)
  document.addEventListener('scroll', closeContextMenu, true)
  window.addEventListener('admin:resource-tree-changed', handleTreeChanged)
  removeRouteListener = router.afterEach?.(() => closeContextMenu()) ?? null
})
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', closeContextMenu)
  document.removeEventListener('keydown', handleEscape)
  document.removeEventListener('scroll', closeContextMenu, true)
  window.removeEventListener('admin:resource-tree-changed', handleTreeChanged)
  removeRouteListener?.()
})
watch(siteId, reloadTree)
</script>

<template>
  <section class="resource-panel">
    <div class="sidebar-heading-row">
      <span class="sidebar-heading">Ресурсы</span>
      <el-button v-if="siteId && canCreate" text :icon="Plus" aria-label="Создать корневой ресурс" @click="create(null)" />
    </div>
    <el-empty v-if="!siteId" :image-size="58" description="Сначала выберите сайт" />
    <div v-else-if="rootError" class="tree-error">
      <span>Не удалось загрузить дерево</span>
      <el-button size="small" @click="reloadTree">Повторить</el-button>
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
      :default-expanded-keys="Array.from(expandedIDs)"
      :draggable="canUpdate"
      :allow-drag="allowDrag"
      :allow-drop="allowDrop"
      highlight-current
      :expand-on-click-node="false"
      empty-text="Ресурсов пока нет"
      @node-click="openEditor"
      @node-contextmenu="openContextMenu"
      @node-expand="rememberExpanded"
      @node-collapse="forgetExpanded"
      @node-drop="handleDrop"
    >
      <template #default="{ data }">
        <span v-if="data.loadError" class="resource-node resource-node-error">
          <span class="resource-node-title">{{ data.display_title }}</span>
          <el-button size="small" text @click.stop="retryNode(data.retryParentId)">Повторить</el-button>
        </span>
        <span v-else class="resource-node" :class="{ 'is-deleted': data.deleted }">
          <el-icon class="resource-node-icon"><component :is="icon(data.icon)" /></el-icon>
          <span class="resource-node-title">{{ data.display_title }} ({{ data.id }})</span>
          <el-button
            v-if="canCreate && data.can_create_child && !data.deleted"
            class="resource-node-add"
            text size="small" :icon="Plus"
            aria-label="Создать дочерний ресурс"
            @click.stop="create(data)"
          />
        </span>
      </template>
    </el-tree>

    <div
      v-if="context"
      class="resource-context-menu"
      :style="{ left: `${context.x}px`, top: `${context.y}px` }"
      role="menu"
      @pointerdown.stop
    >
      <button type="button" role="menuitem" @click="openEditor(context.item); closeContextMenu()">Редактировать</button>
      <template v-if="canDelete && !context.item.deleted">
        <button type="button" role="menuitem" @click="softDelete(context.item)">Удалить</button>
      </template>
      <template v-if="canDelete && context.item.deleted">
        <button type="button" role="menuitem" @click="restore(context.item, false)">Восстановить</button>
        <button v-if="context.item.has_children" type="button" role="menuitem" @click="restore(context.item, true)">Восстановить с потомками</button>
      </template>
      <button v-if="canDelete" type="button" class="is-danger" role="menuitem" @click="permanentDelete(context.item)">Удалить окончательно</button>
    </div>

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
