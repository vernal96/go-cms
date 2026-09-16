<script setup lang="ts">
import { computed } from 'vue'
import { ElFormItem, ElOption, ElRadioButton, ElRadioGroup, ElSelect } from 'element-plus'
import DynamicFieldsForm from '../fields/DynamicFieldsForm.vue'
import { createFieldValues, type DynamicFieldErrors } from '../fields/model'
import type { FieldDefinition, WidgetParamBinding, WidgetValueShape, WidgetValueSource } from '../../types/admin'

const props = defineProps<{
  fields: FieldDefinition[]
  modelValue: Record<string, unknown>
  bindings: Record<string, WidgetParamBinding>
  sources: WidgetValueSource[]
  paramTypes: Record<string, WidgetValueShape>
  errors: DynamicFieldErrors
  siteId: number
  accessToken: string
}>()
const emit = defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
  'update:bindings': [value: Record<string, WidgetParamBinding>]
}>()
const visibleFields = computed(() => props.fields.filter((field) => !field.visible_when ||
  Object.hasOwn(props.bindings, field.visible_when.field) || props.modelValue[field.visible_when.field] === field.visible_when.value))
function sourcesFor(key: string): WidgetValueSource[] {
  const shape = props.paramTypes[key]
  return props.sources.filter((source) => source.type === shape?.type && source.multiple === shape?.multiple)
}
function sourceId(source: WidgetParamBinding): string { return `${source.kind}:${source.key}` }
function setMode(field: FieldDefinition, mode: unknown): void {
  const bindings = { ...props.bindings }
  const values = { ...props.modelValue }
  if (mode === 'source') {
    bindings[field.key] = { kind: 'resource_field', key: '' }
    delete values[field.key]
  } else {
    delete bindings[field.key]
    Object.assign(values, createFieldValues([field]))
  }
  emit('update:modelValue', values)
  emit('update:bindings', bindings)
}
function setSource(key: string, id: unknown): void {
  const source = sourcesFor(key).find((item) => sourceId(item) === id)
  if (source) emit('update:bindings', { ...props.bindings, [key]: { kind: source.kind, key: source.key } })
}
</script>

<template>
  <div v-for="field in visibleFields" :key="field.key" class="widget-param-field">
    <el-radio-group :model-value="bindings[field.key] ? 'source' : 'literal'" :aria-label="`Источник: ${field.label}`" size="small" @update:model-value="setMode(field, $event)">
      <el-radio-button value="literal">Значение</el-radio-button>
      <el-radio-button value="source" :disabled="!bindings[field.key] && !sourcesFor(field.key).length">Поле ресурса</el-radio-button>
    </el-radio-group>
    <el-form-item v-if="bindings[field.key]" :label="field.label" required :error="errors[field.key]">
      <el-select :model-value="bindings[field.key]?.key ? sourceId(bindings[field.key]!) : undefined" filterable placeholder="Выберите поле ресурса" @update:model-value="setSource(field.key, $event)">
        <el-option v-for="source in sourcesFor(field.key)" :key="sourceId(source)" :value="sourceId(source)" :label="`${source.label} (${source.kind === 'resource_field' ? 'Поле шаблона' : 'Свойство ресурса'}: ${source.key})`" />
      </el-select>
    </el-form-item>
    <dynamic-fields-form v-else :model-value="modelValue" :fields="[{ ...field, visible_when: undefined }]" :errors="errors" :site-id="siteId" :access-token="accessToken" @update:model-value="emit('update:modelValue', $event)" />
  </div>
</template>

<style scoped>
.widget-param-field > .el-radio-group { margin-bottom: 8px; }
.widget-param-field .el-select { width: 100%; }
</style>
