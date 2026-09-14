<script setup lang="ts">
import { computed, inject, ref, watch } from 'vue'
import { ElAlert } from 'element-plus'
import { adminPluginRegistryKey } from '../../admin-plugins/context'
import type { ConfigField, FieldDefinition } from '../../types/admin'
import DynamicFieldsForm from './DynamicFieldsForm.vue'
import { validateFieldValues } from './model'

const props = defineProps<{ fields: ConfigField[]; editor?: string; siteId?: number; accessToken?: string; context?: Record<string, unknown> }>()
const model = defineModel<Record<string, unknown>>({ required: true })
const registry = inject(adminPluginRegistryKey, undefined)
const customEditor = computed(() => props.editor ? registry?.configEditor(props.editor) : undefined)
const custom = ref<{ validate?: () => void }>()
const fields = computed<FieldDefinition[]>(() => props.fields.map(field => ({ ...field, rules: field.rules ?? [] })))
const errors = ref<Record<string,string>>({})
watch([() => props.fields, model],([fields]) => {
 const values = {...model.value}
 let changed=false
 for(const field of fields) if(!Object.hasOwn(values,field.key) && Object.hasOwn(field,'default')) {values[field.key]=JSON.parse(JSON.stringify(field.default));changed=true}
 if(changed) model.value=values
},{immediate:true,deep:true})

function validate(): void {
 if (props.editor) {
  if (!customEditor.value) throw new Error(`Редактор «${props.editor}» недоступен.`)
  custom.value?.validate?.()
  return
 }
 errors.value = validateFieldValues(fields.value, model.value)
 if (Object.keys(errors.value).length) throw new Error('Проверьте настройки.')
}
defineExpose({ validate })
</script>

<template>
 <component v-if="customEditor" :is="customEditor" ref="custom" v-model="model" :fields="fields" :site-id="siteId" :access-token="accessToken" :context="context" />
 <el-alert v-else-if="editor" type="error" :closable="false" :title="`Редактор «${editor}» недоступен.`" />
 <dynamic-fields-form v-else v-model="model" :fields="fields" :errors="errors" :site-id="siteId" :access-token="accessToken" />
</template>
