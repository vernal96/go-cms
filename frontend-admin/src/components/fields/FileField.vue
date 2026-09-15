<script setup lang="ts">
import { computed, inject, onBeforeUnmount, ref, watch } from 'vue'
import { ElButton, ElImage, ElImageViewer, ElInput, ElMessage } from 'element-plus'
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
const fullURL = ref('')
let generation = 0
const id = computed(() => typeof model.value === 'number' && model.value > 0 ? model.value : null)

watch(id, async (value) => {
  const revision = ++generation
  if (!value || !accessToken?.value || !permissions?.value.has('core.file.read')) {
    selected.value = null; revokePreview(); return
  }
  try {
    const loaded = await adminRequest<FilesystemItem>(`/api/files/${value}`, accessToken.value)
    if (revision !== generation) return
    selected.value = loaded
    await loadPreview(revision)
  } catch (error) { if (revision !== generation) return; selected.value = null; ElMessage.error(error instanceof Error ? error.message : 'Не удалось загрузить файл.') }
}, { immediate: true })
onBeforeUnmount(() => { generation++; revokePreview(); closeFullPreview() })

async function loadPreview(revision: number): Promise<void> {
  revokePreview()
  if (!selected.value || !['image/jpeg', 'image/png'].includes(selected.value?.mime_type ?? '') || !accessToken?.value) return
  const blob = await adminBlob(`/api/files/${selected.value.id}/thumbnail?width=128&height=128&fit=contain`, accessToken.value)
  if (revision === generation) previewURL.value = URL.createObjectURL(blob)
}
async function fullPreview(): Promise<void> {
  if (!selected.value || !accessToken?.value) return
  const revision = generation
  try {
    const blob = await adminBlob(`/api/files/${selected.value.id}/preview`, accessToken.value)
    if (revision === generation) { closeFullPreview(); fullURL.value = URL.createObjectURL(blob) }
  } catch (error) { ElMessage.error(error instanceof Error ? error.message : 'Не удалось открыть файл.') }
}
function closeFullPreview(): void { if (fullURL.value) URL.revokeObjectURL(fullURL.value); fullURL.value = '' }
function choose(item: FilesystemItem): void { model.value = item.id }
function clear(): void { model.value = null; selected.value = null; revokePreview() }
function revokePreview(): void { if (previewURL.value) URL.revokeObjectURL(previewURL.value); previewURL.value = '' }
</script>

<template>
  <div class="file-field">
    <el-image v-if="previewURL" :src="previewURL" fit="cover" class="file-field-preview" @click="fullPreview" />
    <el-image-viewer v-if="fullURL" :url-list="[fullURL]" @close="closeFullPreview" />
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
