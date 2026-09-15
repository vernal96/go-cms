<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElButton, ElFormItem } from 'element-plus'
import type { FieldDefinition } from '../../types/admin'
import DynamicField from './DynamicField.vue'
import { createFieldValues, type DynamicFieldErrors, type DynamicValues } from './model'

const props = defineProps<{
  field: FieldDefinition
  modelValue?: unknown
  siteId?: number
  accessToken?: string
  resourceTemplates?: Array<{ code: string; label: string }>
  errors?: DynamicFieldErrors
  fieldPath?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: DynamicValues[]] }>()
const fields = computed(() => props.field.options?.fields ?? [])
const rows = computed<DynamicValues[]>(() => Array.isArray(props.modelValue) ? props.modelValue : [])
const minimum = computed(() => props.field.options?.min_items ?? 0)
const maximum = computed(() => props.field.options?.max_items ?? 0)
const canAdd = computed(() => !maximum.value || rows.value.length < maximum.value)
const path = computed(() => props.fieldPath ?? props.field.key)
// UI identity follows a row across edits and moves; it never enters saved data.
let nextID = 0
const rowIDs = ref<number[]>([])
watch(() => rows.value.length, (length) => {
  while (rowIDs.value.length < length) rowIDs.value.push(nextID++)
  rowIDs.value.length = length
}, { immediate: true })

function add(): void {
  if (!canAdd.value) return
  emit('update:modelValue', [...rows.value, createFieldValues(fields.value)])
}
function remove(index: number): void {
  if (rows.value.length <= minimum.value) return
  rowIDs.value.splice(index, 1)
  emit('update:modelValue', rows.value.filter((_, i) => i !== index))
}
function move(index: number, offset: number): void {
  const target = index + offset
  if (target < 0 || target >= rows.value.length) return
  const updated = [...rows.value]
  ;[updated[index], updated[target]] = [updated[target]!, updated[index]!]
  ;[rowIDs.value[index], rowIDs.value[target]] = [rowIDs.value[target]!, rowIDs.value[index]!]
  emit('update:modelValue', updated)
}
function update(index: number, key: string, value: unknown): void {
  emit('update:modelValue', rows.value.map((row, i) => i === index ? { ...row, [key]: value } : row))
}
</script>

<template>
  <div class="repeater-field">
    <section v-for="(row, index) in rows" :key="rowIDs[index]" class="repeater-row">
      <header class="repeater-row-header">
        <strong>Строка {{ index + 1 }}</strong>
        <div class="repeater-row-actions">
          <el-button :disabled="index === 0" :aria-label="`Переместить строку ${index + 1} вверх`" @click="move(index, -1)">Вверх</el-button>
          <el-button :disabled="index === rows.length - 1" :aria-label="`Переместить строку ${index + 1} вниз`" @click="move(index, 1)">Вниз</el-button>
          <el-button type="danger" plain :disabled="rows.length <= minimum" :aria-label="`Удалить строку ${index + 1}`" @click="remove(index)">Удалить</el-button>
        </div>
      </header>
      <div v-if="errors?.[`${path}[${index}]`]" class="el-form-item__error row-error">{{ errors[`${path}[${index}]`] }}</div>
      <el-form-item
        v-for="nested in fields" :key="nested.key"
        :label="nested.type === 'checkbox' ? undefined : nested.label"
        :required="nested.required"
        :error="errors?.[`${path}[${index}].${nested.key}`]"
      >
        <dynamic-field
          :field="nested" :model-value="row[nested.key]"
          :site-id="siteId" :access-token="accessToken" :resource-templates="resourceTemplates"
          :errors="errors" :field-path="`${path}[${index}].${nested.key}`"
          @update:model-value="update(index, nested.key, $event)"
        />
      </el-form-item>
    </section>
    <el-button :disabled="!canAdd" @click="add">Добавить</el-button>
  </div>
</template>

<style scoped>
.repeater-field { width: 100%; min-width: 0; }
.repeater-row { margin-bottom: 16px; padding: 16px; border: 1px solid var(--el-border-color); border-radius: 6px; }
.repeater-row-header { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 12px; margin-bottom: 18px; }
.repeater-row-actions { display: flex; flex-wrap: wrap; gap: 8px; }
.repeater-row-actions .el-button + .el-button { margin-left: 0; }
.repeater-row :deep(.el-form-item) { margin-bottom: 22px; }
.row-error { position: static; margin-bottom: 12px; }
</style>
