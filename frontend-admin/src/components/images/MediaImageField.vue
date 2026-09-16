<script setup lang="ts">
import { computed, inject, onBeforeUnmount, ref, watch } from 'vue'
import { ElButton, ElMessage } from 'element-plus'
import { adminPermissionsKey } from '../../admin-context'
import { adminBlob, adminRequest } from '../../api/admin-api'
import type { FilesystemItem } from '../../types/admin'
import FilePickerDialog from '../files/FilePickerDialog.vue'
import ImageEditor from './ImageEditor.vue'
import MediaSettingsDialog from './MediaSettingsDialog.vue'
import { imageState, type ImageState } from './image-api'
const props = defineProps<{ accessToken: string; disabled?: boolean; siteId?: number; settingsCode?: string; resourceTemplates?: Array<{ code: string; label: string }> }>()
const model = defineModel<number | null>({ required: true })
const permissions = inject(adminPermissionsKey)
const picker = ref(false), editor = ref(false), settings = ref(false), preview = ref('')
const canConfigure = computed(() => permissions?.value.has('core.media.read') && permissions?.value.has('core.media.update'))
watch(() => [model.value, props.siteId, props.settingsCode], () => { settings.value = false })
const canChoose = computed(() => permissions?.value.has('core.media.create') && permissions?.value.has('core.file.read'))
const canEdit = computed(() => permissions?.value.has('core.media.update') && permissions?.value.has('core.file.create'))
const editable = ref(false)
let generation = 0
function clearURL() { if (preview.value) URL.revokeObjectURL(preview.value); preview.value = '' }
onBeforeUnmount(() => { generation++; clearURL() })
async function load(state?: ImageState) {
  const version = ++generation; clearURL(); if (!model.value) return
  try {
    const s = state ?? await imageState(`/api/media/${model.value}/image`, props.accessToken)
    editable.value = s.editable
    const blob = await adminBlob(`/api/files/${s.current_file.id}/thumbnail?width=128&height=128&fit=contain`, props.accessToken)
    if (version === generation) preview.value = URL.createObjectURL(blob)
  } catch (e) { ElMessage.error(e instanceof Error ? e.message : 'Не удалось загрузить изображение.') }
}
watch(() => [model.value, props.accessToken], () => void load(), { immediate: true })
async function choose(item: FilesystemItem) {
  try { const m = await adminRequest<{ id: number }>('/api/media', props.accessToken, { method: 'POST', body: JSON.stringify({ file_id: item.id }) }); model.value = m.id }
  catch (e) { ElMessage.error(e instanceof Error ? e.message : 'Не удалось выбрать изображение.') }
}
</script>
<template>
  <div class="media-image-field">
    <img v-if="preview" :src="preview" alt="Изображение" width="96" height="96" style="object-fit:contain" />
    <el-button :disabled="disabled || !canChoose" @click="picker = true">Выбрать изображение</el-button>
    <el-button v-if="model && editable" :disabled="disabled || !canEdit" @click="editor = true">Редактировать</el-button>
    <el-button v-if="model && settingsCode" :disabled="disabled || !siteId || !canConfigure" @click="settings = true">Настройки</el-button>
    <el-button v-if="model" :disabled="disabled" @click="model = null">Очистить</el-button>
    <file-picker-dialog v-model="picker" :access-token="accessToken" :permissions="permissions ?? new Set()" :mime-types="['image/jpeg','image/png']" @select="choose" />
    <media-settings-dialog v-if="settings && model && siteId && settingsCode" v-model="settings" :media-id="model" :site-id="siteId" :settings-code="settingsCode" :access-token="accessToken" :resource-templates="resourceTemplates" />
    <image-editor v-if="model" v-model="editor" :access-token="accessToken" :base-url="`/api/media/${model}/image`" @saved="load" />
  </div>
</template>

<style scoped>
.media-image-field { display:flex; flex-wrap:wrap; align-items:center; gap:8px; }
.media-image-field .el-button { margin:0; }
</style>
