<script setup lang="ts">
import { computed } from 'vue'
import { ElAlert, ElFormItem } from 'element-plus'
import type { FieldDefinition } from '../../types/admin'
import DynamicField from './DynamicField.vue'
import type { DynamicFieldErrors, DynamicValues } from './model'
import { unsupportedFieldTypes } from './model'

const props = withDefaults(
  defineProps<{
    fields: FieldDefinition[]
    modelValue: DynamicValues
    errors?: DynamicFieldErrors
		siteId?: number
		accessToken?: string
		resourceTemplates?: Array<{ code: string; label: string }>
  }>(),
  { errors: () => ({}) },
)
const emit = defineEmits<{ 'update:modelValue': [value: DynamicValues] }>()
const unsupported = computed(() => unsupportedFieldTypes(props.fields))
const visibleFields = computed(() => props.fields.filter((field) => !field.visible_when || props.modelValue[field.visible_when.field] === field.visible_when.value))

function update(key: string, value: unknown): void {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}
</script>

<template>
  <el-alert
    v-if="unsupported.length"
    class="form-alert"
    type="error"
    :closable="false"
    :title="`Невозможно отправить форму: неизвестные типы полей — ${unsupported.join(', ')}.`"
  />
  <el-form-item
    v-for="field in visibleFields"
    :key="field.key"
    :label="field.type === 'checkbox' ? undefined : field.label"
    :required="field.required"
    :error="errors[field.key]"
  >
    <dynamic-field
      :field="field"
      :model-value="modelValue[field.key]"
			:site-id="siteId ?? 0"
			:access-token="accessToken ?? ''"
			:resource-templates="resourceTemplates ?? []"
      @update:model-value="update(field.key, $event)"
    />
  </el-form-item>
</template>
