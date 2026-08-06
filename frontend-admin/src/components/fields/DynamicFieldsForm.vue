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
  }>(),
  { errors: () => ({}) },
)
const emit = defineEmits<{ 'update:modelValue': [value: DynamicValues] }>()
const unsupported = computed(() => unsupportedFieldTypes(props.fields))

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
    v-for="field in fields"
    :key="field.key"
    :label="field.type === 'checkbox' ? undefined : field.label"
    :required="field.required"
    :error="errors[field.key]"
  >
    <dynamic-field
      :field="field"
      :model-value="modelValue[field.key]"
      @update:model-value="update(field.key, $event)"
    />
  </el-form-item>
</template>
