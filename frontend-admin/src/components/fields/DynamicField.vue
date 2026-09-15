<script setup lang="ts">
import { computed, inject } from 'vue'
import { ElAlert } from 'element-plus'
import { adminAccessTokenKey } from '../../admin-context'
import MediaImageField from '../images/MediaImageField.vue'
import { adminPluginRegistryKey } from '../../admin-plugins/context'
import type { FieldDefinition } from '../../types/admin'
import CheckboxField from './CheckboxField.vue'
import NumberField from './NumberField.vue'
import RadioField from './RadioField.vue'
import SelectField from './SelectField.vue'
import TextareaField from './TextareaField.vue'
import TextField from './TextField.vue'
import FileField from './FileField.vue'
import JsonField from './JsonField.vue'
import RepeaterField from './RepeaterField.vue'
import { isMultipleField, type DynamicFieldErrors } from './model'
import MultipleField from './MultipleField.vue'
import ResourcePickerField from './ResourcePickerField.vue'
import RichTextEditor from '../RichTextEditor.vue'

const props = defineProps<{
	field: FieldDefinition
 errors?: DynamicFieldErrors
 fieldPath?: string
	siteId?: number
	accessToken?: string
	resourceTemplates?: Array<{ code: string; label: string }>
}>()
const injectedToken = inject(adminAccessTokenKey)
const token = computed(() => props.accessToken || injectedToken?.value || '')
const registry = inject(adminPluginRegistryKey, undefined)
const customEditor = computed(() => props.field.editor ? registry?.fieldEditor(props.field.editor) : undefined)
const model = defineModel<unknown>()
const control = computed(() => props.field.editor || props.field.type)
const resourceIDs = computed<number[]>(() => Array.isArray(model.value) ? model.value.filter((item): item is number => typeof item === 'number') : [])
</script>

<template>
	<multiple-field v-if="isMultipleField(field) && control !== 'select'" v-model="model" :field="field" :errors="errors" :field-path="fieldPath" :site-id="siteId" :access-token="token" :resource-templates="resourceTemplates" />
	<component v-else-if="customEditor" :is="customEditor" v-model="model" :field="field" :site-id="siteId" :access-token="token" :resource-templates="resourceTemplates" />
	<el-alert v-else-if="field.editor?.includes('.')" type="error" :closable="false" :title="`Редактор «${field.editor}» недоступен.`" />
	<rich-text-editor v-else-if="field.editor === 'html'" :model-value="typeof model === 'string' ? model : ''" @update:model-value="model = $event" />
	<select-field v-else-if="field.editor === 'resource-template'" v-model="model" :choices="(resourceTemplates ?? []).map((item) => ({ value: item.code, label: item.label }))" :multiple="false" />
	<resource-picker-field v-else-if="field.editor === 'resource-picker'" :model-value="typeof model === 'number' ? model : undefined" :site-id="siteId ?? 0" :access-token="accessToken ?? ''" @update:model-value="model = $event" />
	<resource-picker-field v-else-if="field.editor === 'resource-multi-picker'" :model-value="resourceIDs" :site-id="siteId ?? 0" :access-token="accessToken ?? ''" multiple @update:model-value="model = $event" />
	<repeater-field v-else-if="control === 'repeater'" v-model="model" :field="field" :site-id="siteId" :access-token="token" :resource-templates="resourceTemplates" :errors="errors" :field-path="fieldPath" />
 <media-image-field v-else-if="control === 'media'" :model-value="typeof model === 'number' ? model : null" :access-token="token" @update:model-value="model = $event" />
	<json-field v-else-if="control === 'json'" v-model="model" />
  <text-field
    v-else-if="
      control === 'string' ||
      control === 'email' ||
      control === 'phone'
    "
    v-model="model"
    :kind="control as 'string' | 'email' | 'phone'"
  />
  <number-field
    v-else-if="control === 'int' || control === 'float'"
    v-model="model"
    :kind="control as 'int' | 'float'"
    :step="field.options?.step"
  />
  <checkbox-field
    v-else-if="control === 'checkbox'"
    v-model="model"
    :label="field.label"
  />
  <radio-field
    v-else-if="control === 'radio'"
    v-model="model"
    :choices="field.options?.choices ?? []"
  />
  <select-field
    v-else-if="control === 'select'"
    v-model="model"
    :choices="field.options?.choices ?? []"
    :multiple="field.options?.multiple ?? false"
    :min-items="Math.max(field.required ? 1 : 0, field.options?.min_items ?? 0)"
    :max-items="field.options?.max_items ?? 0"
    :errors="errors" :field-path="fieldPath ?? field.key"
  />
  <textarea-field v-else-if="control === 'textarea'" v-model="model" />
  <file-field
    v-else-if="control === 'file'"
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
