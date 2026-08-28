<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElInputNumber, ElOption, ElSelect, ElSwitch } from 'element-plus'
import type { FormField, FormFieldPayload, FormsFieldOptions, FormsFieldType } from './types'

const props = defineProps<{ modelValue: boolean; field?: FormField | null; fields: FormField[]; availableTypes: FormsFieldType[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; save: [payload: FormFieldPayload] }>()
const typeLabels: Record<string, string> = {
  string: 'Строка', integer: 'Целое число', float: 'Число', checkbox: 'Флаг', radio: 'Один вариант', select: 'Список',
  textarea: 'Многострочный текст', email: 'Email', phone: 'Телефон', json: 'JSON',
  file: 'Файл из библиотеки',
  'forms.captcha': 'CAPTCHA', 'forms.consent': 'Согласие', 'forms.upload': 'Загрузка файлов',
}
const state = reactive({
  code: '', type: 'string' as FormsFieldType, label: '', required: false, rules: '', editor: '',
  result_label: '', show_in_results: false, result_position: 0,
  visible_field: '', visible_value: '', step: undefined as number | undefined, choices: '', multiple: false,
  pattern: '', mime_types: '', max_file_size: undefined as number | undefined, max_files: undefined as number | undefined,
  provider: '', consent_text: '', consent_url: '',
})
const editing = computed(() => Boolean(props.field))
const locked = computed(() => props.field?.code === 'privacy_consent' || props.field?.code === 'captcha')
const controllers = computed(() => props.fields.filter((item) => item.id !== props.field?.id && item.type !== 'forms.captcha' && item.type !== 'forms.upload'))

function stringifyCondition(value: unknown): string {
  if (typeof value === 'string') return value
  try { return JSON.stringify(value) } catch { return '' }
}
function reset(): void {
  const item = props.field
  const options = item?.options ?? {}
  Object.assign(state, {
    code: item?.code ?? '', type: item?.type ?? 'string', label: item?.label ?? '', required: item?.required ?? false,
    rules: item?.rules.join(', ') ?? '', editor: item?.editor ?? '', result_label: item?.result_label ?? '',
    show_in_results: item?.show_in_results ?? false, result_position: item?.result_position ?? props.fields.length,
    visible_field: item?.visible_when?.field ?? '', visible_value: stringifyCondition(item?.visible_when?.value),
    step: options.step, choices: (options.choices ?? []).map((choice) => `${choice.value}|${choice.label}`).join('\n'),
    multiple: options.multiple ?? false, pattern: options.pattern ?? '', mime_types: (options.mime_types ?? []).join(', '),
    max_file_size: options.max_file_size, max_files: options.max_files, provider: options.provider ?? '',
    consent_text: options.text ?? '', consent_url: options.url ?? '',
  })
}
watch(() => [props.modelValue, props.field] as const, ([open]) => { if (open) reset() }, { deep: true })

function conditionValue(): unknown {
  const raw = state.visible_value.trim()
  if (!raw) return ''
  try { return JSON.parse(raw) } catch { return raw }
}
function options(): FormsFieldOptions | undefined {
  switch (state.type) {
    case 'integer': case 'float': return state.step === undefined ? {} : { step: state.step }
    case 'radio': return { choices: state.choices.split('\n').map((row) => row.trim()).filter(Boolean).map((row) => { const [value, label] = row.split('|'); return { value: value ?? '', label: label || value || '' } }) }
    case 'select': return { multiple: state.multiple, choices: state.choices.split('\n').map((row) => row.trim()).filter(Boolean).map((row) => { const [value, label] = row.split('|'); return { value: value ?? '', label: label || value || '' } }) }
    case 'phone': return { pattern: state.pattern.trim() }
    case 'forms.captcha': return { provider: state.provider.trim() }
    case 'forms.consent': return { text: state.consent_text.trim(), url: state.consent_url.trim() }
    case 'forms.upload': return { multiple: state.multiple, mime_types: state.mime_types.split(',').map((item) => item.trim()).filter(Boolean), max_file_size: state.max_file_size ?? 0, max_files: state.max_files ?? 0 }
    default: return undefined
  }
}
function save(): void {
  emit('save', {
    code: state.code.trim(), type: state.type, label: state.label.trim(), required: state.required,
    rules: state.rules.split(',').map((item) => item.trim()).filter(Boolean), options: options(), editor: state.editor.trim(),
    visible_when: state.visible_field ? { field: state.visible_field, value: conditionValue() } : undefined,
    result_label: state.result_label.trim(), show_in_results: state.show_in_results, result_position: state.result_position,
  })
}
</script>

<template>
  <el-dialog :model-value="modelValue" :title="editing ? 'Поле формы' : 'Новое поле'" width="min(760px, 96vw)" @update:model-value="emit('update:modelValue', $event)">
    <el-form label-position="top" class="field-editor" @submit.prevent="save">
      <el-form-item label="Тип" required><el-select v-model="state.type" :disabled="locked"><el-option v-for="type in availableTypes" :key="type" :value="type" :label="typeLabels[type] ?? type" /></el-select></el-form-item>
      <el-form-item label="Код" required><el-input v-model="state.code" :disabled="locked" /></el-form-item>
      <el-form-item label="Подпись" required><el-input v-model="state.label" /></el-form-item>
      <el-form-item label="Обязательное"><el-switch v-model="state.required" :disabled="locked" /></el-form-item>
      <el-form-item label="Правила (через запятую)"><el-input v-model="state.rules" placeholder="min=2, max=100" /></el-form-item>
      <el-form-item label="Редактор"><el-input v-model="state.editor" placeholder="Необязательно" /></el-form-item>
      <template v-if="state.type === 'integer' || state.type === 'float'"><el-form-item label="Шаг"><el-input-number v-model="state.step" /></el-form-item></template>
      <template v-if="state.type === 'radio' || state.type === 'select'">
        <el-form-item label="Варианты (value|Подпись, по одному в строке)"><el-input v-model="state.choices" type="textarea" :rows="4" /></el-form-item>
        <el-form-item v-if="state.type === 'select'" label="Несколько значений"><el-switch v-model="state.multiple" /></el-form-item>
      </template>
      <el-form-item v-if="state.type === 'phone'" label="Шаблон"><el-input v-model="state.pattern" /></el-form-item>
      <el-form-item v-if="state.type === 'forms.captcha'" label="CAPTCHA-провайдер"><el-input v-model="state.provider" placeholder="По умолчанию" /></el-form-item>
      <template v-if="state.type === 'forms.consent'"><el-form-item label="Текст согласия"><el-input v-model="state.consent_text" type="textarea" /></el-form-item><el-form-item label="Ссылка на документ"><el-input v-model="state.consent_url" /></el-form-item></template>
      <template v-if="state.type === 'forms.upload'">
        <el-form-item label="Разрешённые MIME-типы"><el-input v-model="state.mime_types" placeholder="image/*, application/pdf" /></el-form-item>
        <el-form-item label="Несколько файлов"><el-switch v-model="state.multiple" /></el-form-item>
        <el-form-item label="Максимум байт на файл"><el-input-number v-model="state.max_file_size" :min="0" /></el-form-item>
        <el-form-item v-if="state.multiple" label="Максимум файлов"><el-input-number v-model="state.max_files" :min="0" /></el-form-item>
      </template>
      <el-form-item label="Показывать, когда"><el-select v-model="state.visible_field" clearable placeholder="Всегда"><el-option v-for="item in controllers" :key="item.id" :label="`${item.label} (${item.code})`" :value="item.code" /></el-select></el-form-item>
      <el-form-item v-if="state.visible_field" label="Равно значению"><el-input v-model="state.visible_value" placeholder="Значение или JSON: true, 10" /></el-form-item>
      <el-form-item label="Подпись в результатах"><el-input v-model="state.result_label" placeholder="По умолчанию — подпись поля" /></el-form-item>
      <el-form-item label="Колонка в списке результатов"><el-switch v-model="state.show_in_results" /></el-form-item>
      <el-form-item label="Позиция в результатах"><el-input-number v-model="state.result_position" :min="0" /></el-form-item>
    </el-form>
    <template #footer><el-button @click="emit('update:modelValue', false)">Отмена</el-button><el-button type="primary" @click="save">Сохранить</el-button></template>
  </el-dialog>
</template>

<style scoped>.field-editor{display:grid;grid-template-columns:1fr 1fr;gap:0 16px}.field-editor :deep(.el-select),.field-editor :deep(.el-input-number){width:100%}@media(max-width:680px){.field-editor{grid-template-columns:1fr}}</style>
