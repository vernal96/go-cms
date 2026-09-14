<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElForm, ElFormItem, ElInput, ElInputNumber, ElOption, ElSelect, ElSwitch } from 'element-plus'
import ConfigurationEditor from '../../components/fields/ConfigurationEditor.vue'
import type { FieldTypeMetadata } from '../../types/admin'
import type { FormField, FormFieldPayload, FormsFieldOptions, FormsFieldType } from './types'

const props = defineProps<{ disabled: boolean; initialType: FormsFieldType; field?: FormField | null; fields: FormField[]; availableTypes: FieldTypeMetadata[]; siteId?: number; accessToken?: string }>()
const emit = defineEmits<{ dirty: [value: boolean] }>()
let baseline = ''
const optionValues = ref<FormsFieldOptions>({})
const optionEditor = ref<{ validate(): void }>()
const state = reactive({
  code: '', type: 'string' as FormsFieldType, label: '', required: false, rules: '', editor: '',
  result_label: '', show_in_results: false, show_on_site: false, result_position: 0,
  visible_field: '', visible_value: '',
})
const selectedType = computed(() => props.availableTypes.find(item => item.code === state.type))
watch(() => state.type, (type) => { optionValues.value = {}; state.editor = selectedType.value?.editor ?? ''; if (type === 'forms.captcha' || type === 'forms.upload') state.show_on_site = false }, {flush:'sync'})
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
    code: item?.code ?? '', type: item?.type ?? props.initialType, label: item?.label ?? '', required: item?.required ?? false,
    rules: item?.rules?.join(', ') ?? '', editor: item?.editor ?? '', result_label: item?.result_label ?? '',
    show_in_results: item?.show_in_results ?? false, show_on_site: item?.show_on_site ?? false, result_position: item?.result_position ?? props.fields.length,
    visible_field: item?.visible_when?.field ?? '', visible_value: stringifyCondition(item?.visible_when?.value),

  })
  optionValues.value = JSON.parse(JSON.stringify(options))
}
onMounted(() => { reset(); baseline = JSON.stringify([state,optionValues.value]); emit('dirty', false) })
watch([state,optionValues], () => emit('dirty', JSON.stringify([state,optionValues.value]) !== baseline), { deep: true, flush: 'sync' })

function conditionValue(): unknown {
  const raw = state.visible_value.trim()
  if (!raw) return ''
  try { return JSON.parse(raw) } catch { return raw }
}
function payload(): FormFieldPayload {
  optionEditor.value?.validate()
  return {
    code: state.code.trim(), type: state.type, label: state.label.trim(), required: state.required,
    rules: state.rules.split(',').map((item) => item.trim()).filter(Boolean), options: Object.keys(optionValues.value).length ? JSON.parse(JSON.stringify(optionValues.value)) : undefined, editor: state.editor.trim(),
    visible_when: state.visible_field ? { field: state.visible_field, value: conditionValue() } : undefined,
    result_label: state.result_label.trim(), show_in_results: state.show_in_results, show_on_site: state.show_on_site, result_position: state.result_position,
  }
}
defineExpose({ payload })
</script>

<template>

    <el-form label-position="top" :disabled="disabled" class="field-editor" @submit.prevent>
      <el-form-item v-if="editing" label="Тип" required><el-select v-model="state.type" :disabled="locked"><el-option v-for="type in availableTypes" :key="type.code" :value="type.code" :label="type.label" /></el-select></el-form-item>
      <el-form-item label="Код" required><el-input v-model="state.code" :disabled="locked" /></el-form-item>
      <el-form-item label="Подпись" required><el-input v-model="state.label" /></el-form-item>
      <el-form-item label="Обязательное"><el-switch v-model="state.required" :disabled="locked" /></el-form-item>
      <el-form-item label="Правила (через запятую)"><el-input v-model="state.rules" placeholder="min=2, max=100" /></el-form-item>
      <el-form-item label="Редактор"><el-input v-model="state.editor" placeholder="Необязательно" /></el-form-item>
      <section class="options"><configuration-editor v-if="selectedType" ref="optionEditor" v-model="optionValues" :fields="selectedType.options" :editor="selectedType.options_editor" :site-id="siteId" :access-token="accessToken" /></section>
      <el-form-item label="Показывать, когда"><el-select v-model="state.visible_field" clearable placeholder="Всегда"><el-option v-for="item in controllers" :key="item.id" :label="`${item.label} (${item.code})`" :value="item.code" /></el-select></el-form-item>
      <el-form-item v-if="state.visible_field" label="Равно значению"><el-input v-model="state.visible_value" placeholder="Значение или JSON: true, 10" /></el-form-item>
      <el-form-item label="Подпись в результатах"><el-input v-model="state.result_label" placeholder="По умолчанию — подпись поля" /></el-form-item>
      <el-form-item label="Колонка в списке результатов"><el-switch v-model="state.show_in_results" /></el-form-item>
      <el-form-item label="Показывать на сайте"><el-switch v-model="state.show_on_site" :disabled="state.type === 'forms.captcha' || state.type === 'forms.upload'" /></el-form-item>
      <el-form-item label="Позиция в результатах"><el-input-number v-model="state.result_position" :min="0" /></el-form-item>
    </el-form>


</template>

<style scoped>.field-editor{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,220px),1fr));gap:0 16px}.field-editor :deep(.el-select),.field-editor :deep(.el-input-number){width:100%}@media(max-width:680px){.field-editor{grid-template-columns:1fr}}</style>
