<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import {
  ElAlert,
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElOption,
  ElSelect,
} from 'element-plus'

import { AdminAPIError, adminRequest } from '../api/admin-api'
import type { FieldValidationError } from '../types/auth'
import type {
  ResourceCreatePayload,
  ResourceMetadata,
  ResourceTreeItem,
} from '../types/admin'
import DynamicFieldsForm from './fields/DynamicFieldsForm.vue'
import {
  createFieldValues,
  fieldErrorMessage,
  unsupportedFieldTypes,
  validateFieldValues,
  type DynamicFieldErrors,
} from './fields/model'

const props = defineProps<{ accessToken: string; siteId: number }>()
const emit = defineEmits<{
  created: [item: ResourceTreeItem, parentId: number | null]
  error: [error: unknown]
}>()

const visible = ref(false)
const loading = ref(false)
const metadataLoading = ref(false)
const errorMessage = ref<string | null>(null)
const serverFieldErrors = ref<FieldValidationError[]>([])
const localFieldErrors = ref<DynamicFieldErrors>({})
const metadata = ref<ResourceMetadata>({ types: [], templates: [] })
const parent = ref<ResourceTreeItem | null>(null)
const form = reactive({
  type: 'page' as 'page' | 'link',
  template_code: '',
  title: '',
  menu_title: '',
  slug: '',
  external_url: '',
  settings: {} as Record<string, unknown>,
})
const selectedTemplate = computed(
  () =>
    metadata.value.templates.find((item) => item.code === form.template_code) ??
    null,
)
const displayedFieldErrors = computed<DynamicFieldErrors>(() => {
  const result = { ...localFieldErrors.value }
  for (const error of serverFieldErrors.value) {
    result[error.key] = fieldErrorMessage(error.rule, error.param)
  }
  return result
})
const title = computed(() =>
  parent.value
    ? `Создать ресурс в «${parent.value.display_title}»`
    : 'Создать корневой ресурс',
)

async function open(parentItem: ResourceTreeItem | null): Promise<void> {
  parent.value = parentItem
  Object.assign(form, {
    type: 'page',
    template_code: '',
    title: '',
    menu_title: '',
    slug: '',
    external_url: '',
    settings: {},
  })
  errorMessage.value = null
  serverFieldErrors.value = []
  localFieldErrors.value = {}
  visible.value = true
  metadataLoading.value = true
  try {
    metadata.value = await adminRequest<ResourceMetadata>(
      `/api/admin/sites/${props.siteId}/resource-metadata`,
      props.accessToken,
    )
    form.type = metadata.value.types[0]?.code ?? 'link'
    form.template_code = metadata.value.templates[0]?.code ?? ''
    form.settings = createFieldValues(selectedTemplate.value?.fields ?? [])
  } catch (error) {
    errorMessage.value = 'Не удалось загрузить типы ресурсов.'
    emit('error', error)
  } finally {
    metadataLoading.value = false
  }
}

watch(
  () => form.template_code,
  (code, previous) => {
    if (!previous || code === previous || !visible.value) return
    form.settings = createFieldValues(selectedTemplate.value?.fields ?? [])
    serverFieldErrors.value = []
    localFieldErrors.value = {}
  },
)

watch(
  () => form.type,
  (type) => {
    if (!visible.value) return
    if (type === 'link') form.settings = {}
    else form.settings = createFieldValues(selectedTemplate.value?.fields ?? [])
  },
)

async function submit(): Promise<void> {
  errorMessage.value = null
  serverFieldErrors.value = []
  localFieldErrors.value = {}
  if (!form.title.trim()) {
    errorMessage.value = 'Заполните заголовок ресурса.'
    return
  }
  if (form.type === 'page' && !form.template_code) {
    errorMessage.value = 'Выберите шаблон страницы.'
    return
  }
  if (form.type === 'link' && !form.external_url.trim()) {
    errorMessage.value = 'Укажите адрес ссылки.'
    return
  }
  const fields =
    form.type === 'page' ? (selectedTemplate.value?.fields ?? []) : []
  if (unsupportedFieldTypes(fields).length > 0) {
    errorMessage.value =
      'Форма содержит неизвестные типы полей и не может быть отправлена.'
    return
  }
  localFieldErrors.value = validateFieldValues(fields, form.settings)
  if (Object.keys(localFieldErrors.value).length > 0) return
  const payload: ResourceCreatePayload = {
    parent_id: parent.value?.id ?? null,
    type: form.type,
    title: form.title.trim(),
    menu_title: form.menu_title.trim(),
    slug: form.slug.trim(),
    settings: { ...form.settings },
  }
  if (form.type === 'page') payload.template_code = form.template_code
  else payload.external_url = form.external_url.trim()

  loading.value = true
  try {
    const created = await adminRequest<ResourceTreeItem>(
      `/api/admin/sites/${props.siteId}/resources`,
      props.accessToken,
      { method: 'POST', body: JSON.stringify(payload) },
    )
    visible.value = false
    emit('created', created, parent.value?.id ?? null)
  } catch (error) {
    if (error instanceof AdminAPIError)
      serverFieldErrors.value = error.fieldErrors
    errorMessage.value =
      error instanceof Error ? error.message : 'Не удалось создать ресурс.'
    emit('error', error)
  } finally {
    loading.value = false
  }
}

defineExpose({ open })
</script>

<template>
  <el-dialog v-model="visible" :title="title" width="520px" destroy-on-close>
    <el-alert
      v-if="errorMessage"
      class="dialog-alert"
      type="error"
      :closable="false"
      :title="errorMessage"
    />
    <el-form label-position="top" :model="form" :disabled="metadataLoading">
      <el-form-item label="Тип">
        <el-select v-model="form.type" class="full-width">
          <el-option
            v-for="item in metadata.types"
            :key="item.code"
            :label="item.label"
            :value="item.code"
          />
        </el-select>
      </el-form-item>
      <el-form-item v-if="form.type === 'page'" label="Шаблон">
        <el-select v-model="form.template_code" class="full-width">
          <el-option
            v-for="item in metadata.templates"
            :key="item.code"
            :label="item.label"
            :value="item.code"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="Название"
        ><el-input v-model="form.title"
      /></el-form-item>
      <el-form-item label="Название в меню"
        ><el-input v-model="form.menu_title"
      /></el-form-item>
      <el-form-item label="Код"
        ><el-input
          v-model="form.slug"
          placeholder="Оставьте пустым для генерации по заголовку"
      /></el-form-item>
      <el-form-item v-if="form.type === 'link'" label="Адрес ссылки">
        <el-input
          v-model="form.external_url"
          placeholder="https://example.com"
        />
      </el-form-item>
      <dynamic-fields-form
        v-if="form.type === 'page' && selectedTemplate"
        :fields="selectedTemplate.fields"
        :model-value="form.settings"
        :errors="displayedFieldErrors"
        @update:model-value="form.settings = $event"
      />
    </el-form>
    <template #footer>
      <el-button @click="visible = false">Отмена</el-button>
      <el-button
        type="primary"
        :loading="loading"
        :disabled="metadataLoading"
        @click="submit"
        >Создать</el-button
      >
    </template>
  </el-dialog>
</template>
