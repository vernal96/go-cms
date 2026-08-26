<script setup lang="ts">
import { computed } from 'vue'
import { ElAlert } from 'element-plus'
import type { FieldDefinition } from '../../types/admin'
import CheckboxField from './CheckboxField.vue'
import NumberField from './NumberField.vue'
import RadioField from './RadioField.vue'
import SelectField from './SelectField.vue'
import TextareaField from './TextareaField.vue'
import TextField from './TextField.vue'
import FileField from './FileField.vue'
import JsonField from './JsonField.vue'
import ResourcePickerField from './ResourcePickerField.vue'
import RichTextEditor from '../RichTextEditor.vue'

defineProps<{
	field: FieldDefinition
	siteId?: number
	accessToken?: string
	resourceTemplates?: Array<{ code: string; label: string }>
}>()
const model = defineModel<unknown>()
const resourceIDs = computed<number[]>(() => Array.isArray(model.value) ? model.value.filter((item): item is number => typeof item === 'number') : [])
</script>

<template>
	<rich-text-editor v-if="field.editor === 'html'" :model-value="typeof model === 'string' ? model : ''" @update:model-value="model = $event" />
	<select-field v-else-if="field.editor === 'resource-template'" v-model="model" :choices="(resourceTemplates ?? []).map((item) => ({ value: item.code, label: item.label }))" :multiple="false" />
	<resource-picker-field v-else-if="field.editor === 'resource-picker'" :model-value="typeof model === 'number' ? model : undefined" :site-id="siteId ?? 0" :access-token="accessToken ?? ''" @update:model-value="model = $event" />
	<resource-picker-field v-else-if="field.editor === 'resource-multi-picker'" :model-value="resourceIDs" :site-id="siteId ?? 0" :access-token="accessToken ?? ''" multiple @update:model-value="model = $event" />
	<json-field v-else-if="field.type === 'json'" v-model="model" />
  <text-field
    v-else-if="
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
