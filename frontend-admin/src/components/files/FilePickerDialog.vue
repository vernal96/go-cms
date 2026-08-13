<script setup lang="ts">
import { ElDialog } from 'element-plus'
import type { FilesystemItem } from '../../types/admin'
import FileExplorer from './FileExplorer.vue'

defineProps<{
  accessToken: string
  permissions: ReadonlySet<string>
  storages?: string[]
  mimeTypes?: string[]
}>()
const visible = defineModel<boolean>({ required: true })
const emit = defineEmits<{ select: [item: FilesystemItem] }>()

function select(item: FilesystemItem): void {
  emit('select', item)
  visible.value = false
}
</script>

<template>
  <el-dialog v-model="visible" title="Выбор файла" width="min(1180px, 96vw)" class="file-picker-dialog" destroy-on-close>
    <file-explorer
      :access-token="accessToken"
      :permissions="permissions"
      :allowed-storages="storages"
      :allowed-m-i-m-e-types="mimeTypes"
      picker
      @select="select"
    />
  </el-dialog>
</template>
