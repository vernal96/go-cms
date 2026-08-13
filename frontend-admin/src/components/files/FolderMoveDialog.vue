<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElAlert, ElButton, ElDialog, ElTree } from 'element-plus'
import type Node from 'element-plus/es/components/tree/src/model/node'
import type { LoadFunction } from 'element-plus/es/components/tree/src/tree.type'

import { adminRequest } from '../../api/admin-api'
import type { FilesystemItem, FilesystemListingResponse } from '../../types/admin'

interface FolderNode {
  key: string
  id: number | null
  name: string
  disabled: boolean
  blocked: boolean
  isLeaf?: boolean
}

const props = defineProps<{
  accessToken: string
  disk: string
  items: FilesystemItem[]
}>()
const visible = defineModel<boolean>({ required: true })
const emit = defineEmits<{ confirm: [folderID: number | null] }>()
const destination = ref<FolderNode | null>(null)
const error = ref('')
const revision = ref(0)
const root = computed<FolderNode>(() => ({
  key: `root:${revision.value}`,
  id: null,
  name: 'Корень',
  blocked: false,
  disabled: alreadyThere(null),
}))
const treeProps = { label: 'name', disabled: 'disabled', isLeaf: 'isLeaf' }

watch(visible, (opened) => {
  if (!opened) return
  destination.value = null
  error.value = ''
  revision.value++
})

const loadNode: LoadFunction = async (node: Node, resolve) => {
  if (node.level === 0) {
    resolve([root.value])
    return
  }
  const data = node.data as FolderNode | undefined
  if (!data) {
    resolve([])
    return
  }
  try {
    const query = new URLSearchParams({ disk: props.disk })
    if (data.id !== null) query.set('folder_id', String(data.id))
    const response = await adminRequest<FilesystemListingResponse>(
      `/api/admin/filesystem/items?${query}`,
      props.accessToken,
    )
    resolve(response.items.filter((item) => item.kind === 'folder').map((item) => {
      const source = props.items.some((selected) => selected.kind === 'folder' && selected.id === item.id)
      const blocked = data.blocked || source
      return {
        key: `folder:${item.id}`,
        id: item.id,
        name: item.name,
        blocked,
        disabled: blocked || alreadyThere(item.id),
      } satisfies FolderNode
    }))
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : 'Не удалось загрузить папки.'
    resolve([])
  }
}

function choose(data: FolderNode): void {
  destination.value = data.disabled ? null : data
}

function confirm(): void {
  if (destination.value && !destination.value.disabled) emit('confirm', destination.value.id)
}

function alreadyThere(folderID: number | null): boolean {
  return props.items.length > 0 && props.items.every((item) => item.parent_id === folderID)
}
</script>

<template>
  <el-dialog v-model="visible" title="Переместить" width="560px" destroy-on-close>
    <p class="move-dialog-copy">Выберите папку на диске <strong>{{ disk }}</strong>.</p>
    <el-alert v-if="error" type="error" :closable="false" :title="error" />
    <el-tree
      :key="revision"
      class="folder-move-tree"
      node-key="key"
      lazy
      highlight-current
      :props="treeProps"
      :load="loadNode"
      :default-expanded-keys="[root.key]"
      @current-change="choose"
    />
    <template #footer>
      <el-button @click="visible = false">Отмена</el-button>
      <el-button type="primary" :disabled="!destination" @click="confirm">Переместить</el-button>
    </template>
  </el-dialog>
</template>
