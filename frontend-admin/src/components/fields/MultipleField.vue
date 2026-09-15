<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElButton, ElFormItem } from 'element-plus'
import type { FieldDefinition } from '../../types/admin'
import DynamicField from './DynamicField.vue'
import { createFieldValues, singleValueField, type DynamicFieldErrors } from './model'

const props = defineProps<{
  field: FieldDefinition
  modelValue?: unknown
  errors?: DynamicFieldErrors
  fieldPath?: string
  siteId?: number
  accessToken?: string
  resourceTemplates?: Array<{ code: string; label: string }>
}>()
const emit = defineEmits<{ 'update:modelValue': [value: unknown[]] }>()
const items = computed<unknown[]>(() => Array.isArray(props.modelValue) ? props.modelValue : [])
const itemField = computed(() => singleValueField(props.field))
const path = computed(() => props.fieldPath ?? props.field.key)
const maximum = computed(() => props.field.options?.max_items ?? 0)
const minimum = computed(() => Math.max(props.field.required ? 1 : 0, props.field.options?.min_items ?? 0))
let nextID = 0
const ids = ref<number[]>([])
watch(() => items.value.length, length => {
  while (ids.value.length < length) ids.value.push(nextID++)
  ids.value.length = length
}, { immediate: true, flush: 'sync' })

function add() {
  if (maximum.value && items.value.length >= maximum.value) return
  ids.value.push(nextID++)
  emit('update:modelValue', [...items.value, createFieldValues([itemField.value])[props.field.key]])
}
function remove(index: number) {
  if (items.value.length <= minimum.value) return
  ids.value.splice(index, 1)
  emit('update:modelValue', items.value.filter((_, i) => i !== index))
}
function move(index: number, offset: number) {
  const target = index + offset
  if (target < 0 || target >= items.value.length) return
  const updated = [...items.value]
  ;[updated[index], updated[target]] = [updated[target], updated[index]]
  ;[ids.value[index], ids.value[target]] = [ids.value[target]!, ids.value[index]!]
  emit('update:modelValue', updated)
}
function update(index: number, value: unknown) {
  emit('update:modelValue', items.value.map((item, i) => i === index ? value : item))
}
</script>

<template>
  <div class="multiple-field">
    <section v-for="(item, index) in items" :key="ids[index]" class="multiple-item">
      <div class="multiple-actions">
        <span>Значение {{ index + 1 }}</span>
        <el-button :disabled="index === 0" :aria-label="`Переместить значение ${index + 1} вверх`" @click="move(index, -1)">Вверх</el-button>
        <el-button :disabled="index === items.length - 1" :aria-label="`Переместить значение ${index + 1} вниз`" @click="move(index, 1)">Вниз</el-button>
        <el-button type="danger" plain :disabled="items.length <= minimum" :aria-label="`Удалить значение ${index + 1}`" @click="remove(index)">Удалить</el-button>
      </div>
      <el-form-item :error="errors?.[`${path}[${index}]`]">
        <dynamic-field :field="itemField" :model-value="item" :errors="errors" :field-path="`${path}[${index}]`"
          :site-id="siteId" :access-token="accessToken" :resource-templates="resourceTemplates"
          @update:model-value="update(index, $event)" />
      </el-form-item>
    </section>
    <div class="multiple-footer">
      <el-button :disabled="!!maximum && items.length >= maximum" @click="add">Добавить</el-button>
      <span v-if="minimum || maximum" class="multiple-limits">Значений: {{ items.length }}. Минимум: {{ minimum }}<template v-if="maximum">, максимум: {{ maximum }}</template>.</span>
    </div>
  </div>
</template>

<style scoped>
.multiple-field { width: 100%; min-width: 0; display: grid; gap: 12px; }
.multiple-item { min-width: 0; padding: 12px; border: 1px solid var(--el-border-color); border-radius: 6px; }
.multiple-actions, .multiple-footer { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.multiple-actions { margin-bottom: 12px; }
.multiple-actions span { margin-right: auto; }
.multiple-actions .el-button { margin: 0; }
.multiple-item :deep(.el-form-item) { margin-bottom: 18px; }
.multiple-limits { color: var(--el-text-color-secondary); font-size: 12px; }
</style>
