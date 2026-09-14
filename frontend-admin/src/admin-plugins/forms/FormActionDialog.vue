<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElAlert, ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElInputNumber, ElOption, ElSelect, ElSwitch } from 'element-plus'
import ConfigurationEditor from '../../components/fields/ConfigurationEditor.vue'
import type { ActionTypeMetadata, FormAction, FormField, FormStatus, FormTrigger } from './types'
const props = defineProps<{
 modelValue:boolean; action?:FormAction|null; actionTypes:ActionTypeMetadata[]; fields:FormField[]; statuses:FormStatus[];
 accessToken:string; siteID:number; permissions:ReadonlySet<string>; nextPosition:number
}>()
const emit = defineEmits<{ 'update:modelValue':[value:boolean]; save:[payload:Pick<FormAction,'code'|'name'|'enabled'|'trigger'|'action_type'|'config'|'position'>] }>()
const state = reactive({code:'',name:'',enabled:true,trigger_type:'submitted' as 'submitted'|'status_changed',from:'',to:'',action_type:'',position:0})
const values = ref<Record<string,unknown>>({})
const error = ref('')
const editor = ref<{ validate(): void }>()
const selectedType = computed(() => props.actionTypes.find(item => item.code === state.action_type))
const editorContext = computed(() => ({fields:props.fields,permissions:props.permissions,trigger:state.trigger_type}))
watch(() => state.action_type, () => { values.value = {}; error.value = '' }, {flush:'sync'})
function reset():void {
 const item = props.action
 Object.assign(state,{code:item?.code ?? '',name:item?.name ?? '',enabled:item?.enabled ?? true,trigger_type:item?.trigger.type ?? 'submitted',from:item?.trigger.from_status ?? '',to:item?.trigger.to_status ?? '',action_type:item?.action_type ?? props.actionTypes.find(type => type.available)?.code ?? '',position:item?.position ?? props.nextPosition})
 values.value = JSON.parse(JSON.stringify(item?.config ?? {}))
 error.value = ''
}
watch(() => [props.modelValue,props.action] as const,([open]) => {if(open) reset()}, {deep:true,immediate:true})
function save():void {
 try {
  if (!selectedType.value?.available) throw new Error('Выберите доступный тип действия.')
  editor.value?.validate()
  const trigger:FormTrigger = {type:state.trigger_type}
  if (trigger.type === 'status_changed') {trigger.from_status=state.from || undefined;trigger.to_status=state.to || undefined}
  emit('save',{code:state.code.trim(),name:state.name.trim(),enabled:state.enabled,trigger,action_type:state.action_type,config:JSON.parse(JSON.stringify(values.value)),position:state.position})
 } catch(caught) {error.value = caught instanceof Error ? caught.message : 'Проверьте настройки.'}
}
</script>

<template>
  <el-dialog class="forms-mail-dialog" :model-value="modelValue" :title="action ? 'Действие' : 'Новое действие'" width="min(780px, 96vw)" @update:model-value="emit('update:modelValue', $event)">
    <el-form label-position="top" class="action-editor" @submit.prevent="save">
      <el-form-item label="Название" required><el-input v-model="state.name" /></el-form-item>
      <el-form-item label="Код" required><el-input v-model="state.code" /></el-form-item>
      <el-form-item label="Включено"><el-switch v-model="state.enabled" /></el-form-item>
      <el-form-item label="Позиция"><el-input-number v-model="state.position" :min="0" /></el-form-item>
      <el-form-item label="Событие" required><el-select v-model="state.trigger_type"><el-option value="submitted" label="Форма отправлена" /><el-option value="status_changed" label="Статус изменён" /></el-select></el-form-item>
      <template v-if="state.trigger_type === 'status_changed'"><el-form-item label="Из статуса"><el-select v-model="state.from" clearable placeholder="Любой"><el-option v-for="status in statuses" :key="status.id" :label="status.name" :value="status.code" /></el-select></el-form-item><el-form-item label="В статус"><el-select v-model="state.to" clearable placeholder="Любой"><el-option v-for="status in statuses" :key="status.id" :label="status.name" :value="status.code" /></el-select></el-form-item></template>
      <el-form-item label="Тип действия" required><el-select v-model="state.action_type"><el-option v-for="type in actionTypes" :key="type.code" :value="type.code" :label="type.available ? type.label : `${type.label} (недоступно)`" :disabled="!type.available" /></el-select></el-form-item>
    </el-form>

    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <section v-if="selectedType" class="generic-action-editor"><el-form label-position="top"><configuration-editor :key="state.action_type" ref="editor" v-model="values" :fields="selectedType.fields ?? []" :editor="selectedType.editor_code" :site-id="siteID" :access-token="accessToken" :context="editorContext" /></el-form></section>
    <template #footer><el-button @click="emit('update:modelValue', false)">Отмена</el-button><el-button type="primary" @click="save">Сохранить</el-button></template>
  </el-dialog>
</template>

<style scoped>.action-editor{display:grid;grid-template-columns:1fr 1fr;gap:0 16px}.action-editor :deep(.el-select),.action-editor :deep(.el-input-number),.mail-action-editor :deep(.el-select){width:100%}.mail-action-editor,.generic-action-editor{padding-top:12px;border-top:1px solid var(--el-border-color)}@media(max-width:680px){.action-editor{grid-template-columns:1fr}}</style>
