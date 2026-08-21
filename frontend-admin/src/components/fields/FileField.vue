<script setup lang="ts">
import { computed, inject, onBeforeUnmount, ref, watch } from 'vue'
import { ElButton, ElImage, ElInput, ElMessage } from 'element-plus'
import { Delete, FolderOpened } from '@element-plus/icons-vue'
import { adminAccessTokenKey, adminPermissionsKey } from '../../admin-context'
import { adminBlob, adminRequest } from '../../api/admin-api'
import type { FilesystemItem } from '../../types/admin'
import FilePickerDialog from '../files/FilePickerDialog.vue'

const props = withDefaults(defineProps<{ storages?: string[]; mimeTypes?: string[] }>(), {
  storages: () => [], mimeTypes: () => [],
})
const model = defineModel<unknown>()
const accessToken = inject(adminAccessTokenKey)
const permissions = inject(adminPermissionsKey)
const pickerVisible = ref(false)
const selected = ref<FilesystemItem | null>(null)
const previewURL = ref('')
const id = computed(() => typeof model.value === 'number' && model.value > 0 ? model.value : null)

watch(id, async (value) => {
  if (!value || !accessToken?.value || !permissions?.value.has('core.file.read')) {
    selected.value = null; revokePreview(); return
  }
  try {
    selected.value = await adminRequest<FilesystemItem>(`/api/files/${value}`, accessToken.value)
    await loadPreview()
  } catch (error) { selected.value = null; ElMessage.error(error instanceof Error ? error.message : 'Не удалось загрузить файл.') }
}, { immediate: true })
onBeforeUnmount(revokePreview)

async function loadPreview(): Promise<void> {
  revokePreview()
  if (!selected.value?.mime_type?.startsWith('image/') || !accessToken?.value) return
  const blob = await adminBlob(`/api/files/${selected.value.id}/preview`, accessToken.value)
  previewURL.value = URL.createObjectURL(blob)
}
function choose(item: FilesystemItem): void { model.value = item.id }
function clear(): void { model.value = null; selected.value = null; revokePreview() }
function revokePreview(): void { if (previewURL.value) URL.revokeObjectURL(previewURL.value); previewURL.value = '' }
</script>

<template>
  <div class="file-field">
    <el-image v-if="previewURL" :src="previewURL" fit="cover" class="file-field-preview" :preview-src-list="[previewURL]" />
    <el-input :model-value="selected?.name ?? ''" readonly placeholder="Файл не выбран" />
    <el-button :icon="FolderOpened" :disabled="!accessToken || !permissions?.has('core.file.read')" @click="pickerVisible = true">Выбрать</el-button>
    <el-button v-if="id" :icon="Delete" title="Очистить" @click="clear" />
    <file-picker-dialog
      v-if="accessToken && permissions"
      v-model="pickerVisible"
      :access-token="accessToken"
      :permissions="permissions"
      :storages="props.storages"
      :mime-types="props.mimeTypes"
      @select="choose"
    />
  </div>
</template>
