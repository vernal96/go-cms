<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElAlert, ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElInputNumber, ElOption, ElSelect, ElSwitch } from 'element-plus'
import { listMailTemplates } from '../mail/api'
import type { MailTemplate } from '../mail/types'
import type { ActionTypeMetadata, FormAction, FormField, FormStatus, FormTrigger } from './types'

const props = defineProps<{
  modelValue: boolean; action?: FormAction | null; actionTypes: ActionTypeMetadata[]; fields: FormField[]; statuses: FormStatus[];
  accessToken: string; siteID: number; permissions: ReadonlySet<string>; nextPosition: number
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; save: [payload: Pick<FormAction, 'code' | 'name' | 'enabled' | 'trigger' | 'action_type' | 'config' | 'position'>] }>()
const state = reactive({ code: '', name: '', enabled: true, trigger_type: 'submitted' as 'submitted' | 'status_changed', from: '', to: '', action_type: '', position: 0, template_code: '', attachments: [] as string[], generic: {} as Record<string, string> })
const mappings = reactive<Record<string, string>>({})
const templates = ref<MailTemplate[]>([])
const loadingTemplates = ref(false)
const templateError = ref('')
const selectedTemplate = computed(() => templates.value.find((item) => item.code === state.template_code))
const scalarFields = computed(() => props.fields.filter((item) => item.type !== 'forms.captcha' && item.type !== 'forms.upload'))
const uploadFields = computed(() => props.fields.filter((item) => item.type === 'forms.upload'))
const selectedType = computed(() => props.actionTypes.find((item) => item.code === state.action_type))

function clearObject(target: Record<string, string>): void { for (const key of Object.keys(target)) delete target[key] }
function reset(): void {
  const item = props.action
  const config = item?.config ?? {}
  Object.assign(state, {
    code: item?.code ?? '', name: item?.name ?? '', enabled: item?.enabled ?? true,
    trigger_type: item?.trigger.type ?? 'submitted', from: item?.trigger.from_status ?? '', to: item?.trigger.to_status ?? '',
    action_type: item?.action_type ?? props.actionTypes.find((type) => type.available)?.code ?? '', position: item?.position ?? props.nextPosition,
    template_code: String(config.template_code ?? ''), attachments: Array.isArray(config.attachments) ? [...config.attachments] as string[] : [], generic: {},
  })
  clearObject(mappings)
  if (config.values && typeof config.values === 'object') Object.assign(mappings, config.values)
  const generic: Record<string, string> = {}
  for (const field of selectedType.value?.fields ?? []) generic[field.key] = String(config[field.key] ?? '')
  state.generic = generic
}
async function loadTemplates(): Promise<void> {
  if (!props.permissions.has('mail.template.read') || !props.siteID) return
  loadingTemplates.value = true; templateError.value = ''
  try { templates.value = (await listMailTemplates(props.accessToken, props.siteID, 1, 100)).items }
  catch (caught) { templateError.value = caught instanceof Error ? caught.message : 'Не удалось загрузить шаблоны Mail.' }
  finally { loadingTemplates.value = false }
}
watch(() => [props.modelValue, props.action] as const, ([open]) => { if (open) { reset(); if (state.action_type === 'mail') void loadTemplates() } }, { deep: true, immediate: true })
watch(() => state.action_type, (value) => { if (props.modelValue && value === 'mail' && templates.value.length === 0) void loadTemplates() })
watch(selectedTemplate, (template) => {
  if (!template) return
  for (const variable of template.variables) if (!(variable.key in mappings)) mappings[variable.key] = ''
})

function save(): void {
  const trigger: FormTrigger = { type: state.trigger_type }
  if (state.trigger_type === 'status_changed') { trigger.from_status = state.from || undefined; trigger.to_status = state.to || undefined }
  let config: Record<string, unknown>
  if (state.action_type === 'mail') {
    const values: Record<string, string> = {}
    for (const [key, value] of Object.entries(mappings)) if (value) values[key] = value
    config = { template_code: state.template_code, values, attachments: [...state.attachments] }
  } else {
    config = {}
    for (const field of selectedType.value?.fields ?? []) config[field.key] = state.generic[field.key] ?? ''
  }
  emit('save', { code: state.code.trim(), name: state.name.trim(), enabled: state.enabled, trigger, action_type: state.action_type, config, position: state.position })
}
</script>

<template>
  <el-dialog :model-value="modelValue" :title="action ? 'Действие' : 'Новое действие'" width="min(780px, 96vw)" @update:model-value="emit('update:modelValue', $event)">
    <el-form label-position="top" class="action-editor" @submit.prevent="save">
      <el-form-item label="Название" required><el-input v-model="state.name" /></el-form-item>
      <el-form-item label="Код" required><el-input v-model="state.code" /></el-form-item>
      <el-form-item label="Включено"><el-switch v-model="state.enabled" /></el-form-item>
      <el-form-item label="Позиция"><el-input-number v-model="state.position" :min="0" /></el-form-item>
      <el-form-item label="Событие" required><el-select v-model="state.trigger_type"><el-option value="submitted" label="Форма отправлена" /><el-option value="status_changed" label="Статус изменён" /></el-select></el-form-item>
      <template v-if="state.trigger_type === 'status_changed'"><el-form-item label="Из статуса"><el-select v-model="state.from" clearable placeholder="Любой"><el-option v-for="status in statuses" :key="status.id" :label="status.name" :value="status.code" /></el-select></el-form-item><el-form-item label="В статус"><el-select v-model="state.to" clearable placeholder="Любой"><el-option v-for="status in statuses" :key="status.id" :label="status.name" :value="status.code" /></el-select></el-form-item></template>
      <el-form-item label="Тип действия" required><el-select v-model="state.action_type"><el-option v-for="type in actionTypes" :key="type.code" :value="type.code" :label="type.available ? type.label : `${type.label} (недоступно)`" :disabled="!type.available" /></el-select></el-form-item>
    </el-form>

    <section v-if="state.action_type === 'mail'" class="mail-action-editor">
      <el-alert v-if="!permissions.has('mail.template.read')" type="warning" :closable="false" title="Для настройки письма нужно право mail.template.read." />
      <el-alert v-else-if="templateError" type="error" :closable="false" :title="templateError" />
      <el-form label-position="top">
        <el-form-item label="Шаблон Mail" required><el-select v-model="state.template_code" :loading="loadingTemplates" filterable><el-option v-for="template in templates" :key="template.id" :value="template.code" :label="`${template.name} (${template.code})${template.enabled ? '' : ' — выключен'}`" :disabled="!template.enabled" /></el-select></el-form-item>
        <template v-if="selectedTemplate"><h3>Переменные шаблона</h3><el-form-item v-for="variable in selectedTemplate.variables.filter((item) => item.type !== 'file')" :key="variable.key" :label="`${variable.label}${variable.required ? ' *' : ''}`"><el-select v-model="mappings[variable.key]" clearable placeholder="Поле формы"><el-option v-for="field in scalarFields" :key="field.id" :value="field.code" :label="`${field.label} (${field.code})`" /></el-select></el-form-item></template>
        <el-form-item v-if="state.trigger_type === 'submitted'" label="Вложения из отправки"><el-select v-model="state.attachments" multiple clearable placeholder="Без вложений"><el-option v-for="field in uploadFields" :key="field.id" :value="field.code" :label="`${field.label} (${field.code})`" /></el-select></el-form-item>
        <el-alert v-else type="info" :closable="false" title="Отложенные действия по смене статуса не могут использовать временные файлы отправки." />
      </el-form>
    </section>
    <section v-else-if="selectedType" class="generic-action-editor"><el-form label-position="top"><el-form-item v-for="field in selectedType.fields" :key="field.key" :label="field.label" :required="field.required"><el-input v-model="state.generic[field.key]" /></el-form-item></el-form></section>
    <template #footer><el-button @click="emit('update:modelValue', false)">Отмена</el-button><el-button type="primary" @click="save">Сохранить</el-button></template>
  </el-dialog>
</template>

<style scoped>.action-editor{display:grid;grid-template-columns:1fr 1fr;gap:0 16px}.action-editor :deep(.el-select),.action-editor :deep(.el-input-number),.mail-action-editor :deep(.el-select){width:100%}.mail-action-editor,.generic-action-editor{padding-top:12px;border-top:1px solid var(--el-border-color)}@media(max-width:680px){.action-editor{grid-template-columns:1fr}}</style>
