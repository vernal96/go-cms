<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { FolderOpened } from '@element-plus/icons-vue'
import { ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElInputNumber, ElOption, ElSelect } from 'element-plus'
import FilePickerDialog from '../../components/files/FilePickerDialog.vue'
import type { FilesystemItem } from '../../types/admin'
import type { ElementType, ElementTypeMetadata, FormElement } from './types'

const props = defineProps<{ modelValue: boolean; element?: FormElement | null; availableTypes: ElementTypeMetadata[]; accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; save: [payload: Pick<FormElement, 'code' | 'type' | 'config'>] }>()
const state = reactive({ code: '', type: 'text' as ElementType, text: '', content: '', level: 2, label: '', file_id: 0, alt: '' })
const picker = ref(false)
const selectedName = ref('')
const editing = computed(() => Boolean(props.element))
const locked = computed(() => props.element?.type === 'submit_button')
function reset(): void {
  const item = props.element
  const config = item?.config ?? {}
  Object.assign(state, {
    code: item?.code ?? '', type: item?.type ?? 'text', text: String(config.text ?? ''), content: String(config.content ?? ''),
    level: Number(config.level ?? 2), label: String(config.label ?? ''), file_id: Number(config.file_id ?? 0), alt: String(config.alt ?? ''),
  })
  selectedName.value = ''
}
watch(() => [props.modelValue, props.element] as const, ([open]) => { if (open) reset() }, { deep: true })
function choose(item: FilesystemItem): void { state.file_id = item.id; selectedName.value = item.name }
function config(): Record<string, unknown> {
  if (state.type === 'text') return { content: state.content }
  if (state.type === 'heading') return { text: state.text, level: state.level }
  if (state.type === 'image') return { file_id: state.file_id, alt: state.alt }
  return { label: state.label }
}
function save(): void { emit('save', { code: state.code.trim(), type: state.type, config: config() }) }
</script>

<template>
  <el-dialog :model-value="modelValue" :title="editing ? 'Элемент формы' : 'Новый элемент'" width="min(620px, 95vw)" @update:model-value="emit('update:modelValue', $event)">
    <el-form label-position="top" @submit.prevent="save">
      <el-form-item label="Тип" required><el-select v-model="state.type" :disabled="locked"><el-option v-for="type in availableTypes" :key="type.code" :value="type.code" :label="type.label" /></el-select></el-form-item>
      <el-form-item label="Код" required><el-input v-model="state.code" :disabled="locked" /></el-form-item>
      <el-form-item v-if="state.type === 'text'" label="Текст" required><el-input v-model="state.content" type="textarea" :rows="5" /></el-form-item>
      <template v-if="state.type === 'heading'"><el-form-item label="Заголовок" required><el-input v-model="state.text" /></el-form-item><el-form-item label="Уровень"><el-input-number v-model="state.level" :min="1" :max="6" /></el-form-item></template>
      <template v-if="state.type === 'image'"><el-form-item label="Публичное изображение" required><el-button :icon="FolderOpened" :disabled="!permissions.has('core.file.read')" @click="picker = true">{{ selectedName || (state.file_id ? `Файл #${state.file_id}` : 'Выбрать файл') }}</el-button></el-form-item><el-form-item label="Alt"><el-input v-model="state.alt" /></el-form-item></template>
      <el-form-item v-if="state.type === 'submit_button'" label="Текст кнопки" required><el-input v-model="state.label" /></el-form-item>
    </el-form>
    <template #footer><el-button @click="emit('update:modelValue', false)">Отмена</el-button><el-button type="primary" @click="save">Сохранить</el-button></template>
    <file-picker-dialog v-model="picker" :access-token="accessToken" :permissions="permissions" :storages="['public']" @select="choose" />
  </el-dialog>
</template>

<style scoped>.el-select,.el-input-number{width:100%}</style>
