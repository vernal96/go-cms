<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElAlert, ElFormItem, ElOption, ElSelect } from 'element-plus'
import { listMailTemplates } from '../mail/api'
import type { MailTemplate } from '../mail/types'
import type { FormField } from './types'

const props = defineProps<{siteId?:number;accessToken?:string;context?:Record<string,unknown>}>()
const model = defineModel<Record<string,unknown>>({required:true})
const permissions = computed(() => props.context?.permissions as ReadonlySet<string> | undefined)
const fields = computed(() => (props.context?.fields ?? []) as FormField[])
const templates=ref<MailTemplate[]>([])
const loading=ref(false)
const error=ref('')
const templateCode=computed({get:() => String(model.value.template_code ?? ''),set:(value:string) => {model.value={...model.value,template_code:value}}})
const attachments=computed({get:() => (model.value.attachments ?? []) as string[],set:(value:string[]) => {model.value={...model.value,attachments:value}}})
const mappings=computed(() => (model.value.values ?? {}) as Record<string,string>)
const selectedTemplate=computed(() => templates.value.find(item => item.code === templateCode.value))
const scalarFields=computed(() => fields.value.filter(item => item.type !== 'forms.captcha' && item.type !== 'forms.upload'))
const uploads=computed(() => fields.value.filter(item => item.type === 'forms.upload'))
function mapping(key:string,value:string):void {
 const values={...mappings.value}
 if(value) values[key]=value; else delete values[key]
 model.value={...model.value,values}
}
let requestVersion=0
watch(() => [props.siteId,props.accessToken,permissions.value?.has('mail.template.read')] as const,async () => {
 const version=++requestVersion
 templates.value=[];error.value=''
 if(!props.siteId || !props.accessToken || !permissions.value?.has('mail.template.read')) {loading.value=false;return}
 loading.value=true
 try {const result=await listMailTemplates(props.accessToken,props.siteId,1,100);if(version===requestVersion) templates.value=result.items}
 catch(caught) {if(version===requestVersion) error.value=caught instanceof Error ? caught.message : 'Не удалось загрузить шаблоны Mail.'}
 finally {if(version===requestVersion) loading.value=false}
},{immediate:true})
function validate():void {if(!templateCode.value) throw new Error('Выберите шаблон Mail.')}
defineExpose({validate})
</script>
<template>
 <section class="mail-action-editor">
  <el-alert v-if="!permissions?.has('mail.template.read')" type="warning" :closable="false" title="Для настройки письма нужно право mail.template.read." />
  <el-alert v-else-if="error" type="error" :closable="false" :title="error" />
  <el-form-item label="Шаблон Mail" required><el-select v-model="templateCode" :loading="loading" filterable><el-option v-for="template in templates" :key="template.id" :value="template.code" :label="`${template.name} (${template.code})${template.enabled ? '' : ' — выключен'}`" :disabled="!template.enabled" /></el-select></el-form-item>
  <template v-if="selectedTemplate"><h3>Переменные шаблона</h3><el-form-item v-for="variable in selectedTemplate.variables.filter(item => item.type !== 'file')" :key="variable.key" :label="variable.label" :required="variable.required"><el-select :model-value="mappings[variable.key]" clearable placeholder="Поле формы" @update:model-value="mapping(variable.key,$event)"><el-option v-for="field in scalarFields" :key="field.id" :value="field.code" :label="`${field.label} (${field.code})`" /></el-select></el-form-item></template>
  <el-form-item v-if="context?.trigger === 'submitted'" label="Вложения из отправки"><el-select v-model="attachments" multiple clearable placeholder="Без вложений"><el-option v-for="field in uploads" :key="field.id" :value="field.code" :label="`${field.label} (${field.code})`" /></el-select></el-form-item>
  <el-alert v-else type="info" :closable="false" title="Отложенные действия по смене статуса не могут использовать временные файлы отправки." />
 </section>
</template>
<style scoped>.el-select{width:100%}</style>
