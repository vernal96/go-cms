<script setup lang="ts">
import { useFieldValidation } from '../fields/use-field-validation'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElAlert, ElButton, ElDialog, ElForm } from 'element-plus'
import { AdminAPIError, adminRequest } from '../../api/admin-api'
import type { FieldDefinition } from '../../types/admin'
import DynamicFieldsForm from '../fields/DynamicFieldsForm.vue'
import { createFieldValues, fieldErrorMessage,  type DynamicFieldErrors, type DynamicValues } from '../fields/model'

const { validateFieldValues } = useFieldValidation()

interface SettingsState {
  code: string
  fields: FieldDefinition[]
  values: DynamicValues
  expected_updated_at: string
}
const props = defineProps<{
  mediaId: number
  siteId: number
  settingsCode: string
  accessToken: string
  resourceTemplates?: Array<{ code: string; label: string }>
}>()
const open = defineModel<boolean>({ required: true })
const state = ref<SettingsState>()
const values = ref<DynamicValues>({})
const errors = ref<DynamicFieldErrors>({})
const error = ref('')
const loading = ref(false)
const saving = ref(false)
const conflict = ref(false)
const url = computed(() => `/api/sites/${props.siteId}/media/${props.mediaId}/settings`)
let generation = 0
onBeforeUnmount(() => { generation++ })
async function load() {
  const current = ++generation
  state.value = undefined
  error.value = ''
  errors.value = {}
  conflict.value = false
  loading.value = true
  try {
    const result = await adminRequest<SettingsState>(`${url.value}?code=${encodeURIComponent(props.settingsCode)}`, props.accessToken)
    if (current !== generation) return
    state.value = result
    values.value = createFieldValues(result.fields, result.values)
  } catch (e) {
    if (current === generation) error.value = e instanceof Error ? e.message : 'Не удалось загрузить настройки.'
  } finally { if (current === generation) loading.value = false }
}
watch(() => [open.value, props.mediaId, props.siteId, props.settingsCode, props.accessToken], () => {
  if (open.value) void load()
  else generation++
}, { immediate: true })
async function save() {
  if (!state.value || saving.value || conflict.value) return
  errors.value = validateFieldValues(state.value.fields, values.value)
  if (Object.keys(errors.value).length) return
  saving.value = true
  error.value = ''
  const current = generation
  try {
    await adminRequest<SettingsState>(url.value, props.accessToken, {
      method: 'PUT', body: JSON.stringify({ code: props.settingsCode, values: values.value, expected_updated_at: state.value.expected_updated_at }),
    })
    if (current === generation) open.value = false
  } catch (e) {
    if (current !== generation) return
    if (e instanceof AdminAPIError) {
      for (const item of e.fieldErrors) errors.value[item.key] = fieldErrorMessage(item.rule, item.param)
      conflict.value = e.status === 409
    }
    error.value = conflict.value ? 'Изображение изменилось. Скопируйте введённые значения при необходимости и загрузите актуальные настройки.' : e instanceof Error ? e.message : 'Не удалось сохранить настройки.'
  } finally { saving.value = false }
}
function close() { if (!saving.value) open.value = false }
</script>

<template>
  <el-dialog :model-value="open" title="Настройки изображения" width="560px" :close-on-click-modal="false" :close-on-press-escape="!saving" :show-close="!saving" @update:model-value="close">
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <p v-if="loading">Загрузка настроек…</p>
    <el-form v-if="state" label-position="top" :disabled="saving" @submit.prevent="save">
      <dynamic-fields-form v-model="values" :fields="state.fields" :errors="errors" :site-id="siteId" :access-token="accessToken" :resource-templates="resourceTemplates" />
    </el-form>
    <template #footer>
      <el-button v-if="conflict || (!state && !loading)" @click="load">Загрузить заново</el-button>
      <el-button :disabled="saving" @click="close">Отмена</el-button>
      <el-button type="primary" :loading="saving" :disabled="!state || loading || conflict" @click="save">Сохранить</el-button>
    </template>
  </el-dialog>
</template>
