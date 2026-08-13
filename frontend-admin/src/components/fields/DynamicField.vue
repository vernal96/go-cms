<script setup lang="ts">
import { ElAlert } from 'element-plus'
import type { FieldDefinition } from '../../types/admin'
import CheckboxField from './CheckboxField.vue'
import NumberField from './NumberField.vue'
import RadioField from './RadioField.vue'
import SelectField from './SelectField.vue'
import TextareaField from './TextareaField.vue'
import TextField from './TextField.vue'
import FileField from './FileField.vue'

defineProps<{ field: FieldDefinition }>()
const model = defineModel<unknown>()
</script>

<template>
  <text-field
    v-if="
      field.type === 'string' ||
      field.type === 'email' ||
      field.type === 'phone'
    "
    v-model="model"
    :kind="field.type"
  />
  <number-field
    v-else-if="field.type === 'int' || field.type === 'float'"
    v-model="model"
    :kind="field.type"
    :step="field.options?.step"
  />
  <checkbox-field
    v-else-if="field.type === 'checkbox'"
    v-model="model"
    :label="field.label"
  />
  <radio-field
    v-else-if="field.type === 'radio'"
    v-model="model"
    :choices="field.options?.choices ?? []"
  />
  <select-field
    v-else-if="field.type === 'select'"
    v-model="model"
    :choices="field.options?.choices ?? []"
    :multiple="field.options?.multiple ?? false"
  />
  <textarea-field v-else-if="field.type === 'textarea'" v-model="model" />
  <file-field
    v-else-if="field.type === 'file'"
    v-model="model"
    :storages="field.options?.storages"
    :mime-types="field.options?.mime_types"
  />
  <el-alert
    v-else
    type="error"
    :closable="false"
    :title="`Тип поля «${field.type}» не поддерживается.`"
  />
</template>
