<script setup lang="ts">
import { ref } from 'vue'
import { Delete, FolderOpened, Plus } from '@element-plus/icons-vue'
import { ElButton, ElInput, ElOption, ElSelect } from 'element-plus'
import FilePickerDialog from '../../components/files/FilePickerDialog.vue'
import type { FilesystemItem, FieldDefinition } from '../../types/admin'
import type { MailAttachmentTemplate, MailSiteVariable } from './types'

const props = withDefaults(defineProps<{
  modelValue: MailAttachmentTemplate[]
  variables: FieldDefinition[]
  siteVariables?: MailSiteVariable[]
  accessToken: string
  permissions: ReadonlySet<string>
  uploadStorage?: string
  uploadPath?: string
}>(), { siteVariables: () => [], uploadStorage: '', uploadPath: '' })
const emit = defineEmits<{ 'update:modelValue': [value: MailAttachmentTemplate[]] }>()
const pickerVisible = ref(false)
const pickerIndex = ref<number | null>(null)
const selectedNames = ref<Record<number, string>>({})

function add(): void {
  emit('update:modelValue', [...props.modelValue, { source: 'static', filename_template: '' }])
}
function remove(index: number): void {
  emit('update:modelValue', props.modelValue.filter((_, current) => current !== index))
}
function update(index: number, patch: Partial<MailAttachmentTemplate>): void {
  const result = [...props.modelValue]
  result[index] = { ...result[index], ...patch }
  emit('update:modelValue', result)
}
function setSource(index: number, source: 'static' | 'variable' | 'site'): void {
  update(index, source === 'static'
    ? { source, variable: undefined }
    : { source, file_id: undefined, variable: undefined })
}
function openPicker(index: number): void {
  if (!props.permissions.has('core.file.read')) return
  pickerIndex.value = index
  pickerVisible.value = true
}
function selectFile(item: FilesystemItem): void {
  if (pickerIndex.value === null) return
  update(pickerIndex.value, { file_id: item.id })
  selectedNames.value = { ...selectedNames.value, [pickerIndex.value]: item.name }
}
</script>

<template>
  <div class="mail-attachments-editor">
    <div v-for="(attachment, index) in modelValue" :key="index" class="mail-attachment-row">
      <el-select :model-value="attachment.source" @update:model-value="setSource(index, $event)">
        <el-option value="static" label="Статический файл" />
        <el-option value="variable" label="Файловая переменная" />
        <el-option value="site" label="Файл из настроек сайта" />
      </el-select>
      <el-button v-if="attachment.source === 'static'" :icon="FolderOpened" :disabled="!permissions.has('core.file.read')" @click="openPicker(index)">
        {{ selectedNames[index] ?? (attachment.file_id ? `Файл #${attachment.file_id}` : 'Выбрать файл') }}
      </el-button>
      <el-select
        v-else
        :model-value="attachment.variable"
        placeholder="Выберите файловую переменную"
        @update:model-value="update(index, { variable: $event })"
      >
        <template v-if="attachment.source === 'variable'">
          <el-option
            v-for="variable in variables.filter((item) => item.type === 'file')"
            :key="variable.key"
            :label="`${variable.label} — {{data.${variable.key}}}`"
            :value="`data.${variable.key}`"
          />
        </template>
        <template v-else>
          <el-option
            v-for="variable in siteVariables.filter((item) => item.type === 'file')"
            :key="variable.variable"
            :label="`${variable.label} — {{${variable.variable}}}`"
            :value="variable.variable"
          />
        </template>
      </el-select>
      <el-input
        :model-value="attachment.filename_template"
        placeholder="Имя файла (необязательно, поддерживает переменные)"
        @update:model-value="update(index, { filename_template: $event })"
      />
      <el-button :icon="Delete" circle plain aria-label="Удалить вложение" @click="remove(index)" />
    </div>
    <el-button plain :icon="Plus" @click="add">Добавить вложение</el-button>
    <file-picker-dialog
      v-model="pickerVisible"
      :access-token="accessToken"
      :permissions="permissions"
      :initial-storage="uploadStorage"
      :initial-path="uploadPath"
      @select="selectFile"
    />
  </div>
</template>

<style scoped>
.mail-attachments-editor { display: grid; gap: 10px; }
.mail-attachment-row { display: grid; grid-template-columns: 190px minmax(220px, 1fr) minmax(260px, 1.2fr) auto; gap: 8px; align-items: center; }
@media (max-width: 980px) { .mail-attachment-row { grid-template-columns: 1fr; } }
</style>
