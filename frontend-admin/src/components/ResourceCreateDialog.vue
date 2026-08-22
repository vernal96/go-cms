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
  ResourceTypeCode,
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
const metadata = ref<ResourceMetadata>({ types: [], templates: [], widgets: [], extensions: [] })
const parent = ref<ResourceTreeItem | null>(null)
const noTemplateValue = null as unknown as string
const form = reactive({
  type: 'page' as ResourceTypeCode,
  template_code: null as string | null,
  title: '',
  menu_title: '',
  slug: '',
  external_url: '',
  fields: {} as Record<string, unknown>,
  type_settings: { item_url_pattern: '', default_item_template: null as string | null },
})
const selectedTemplate = computed(
  () =>
    metadata.value.templates.find((item) => item.code === form.template_code) ??
    null,
)
const selectedType = computed(() => metadata.value.types.find((item) => item.code === form.type) ?? null)
function capability(snake: string, pascal: string): boolean {
  const values = selectedType.value?.capabilities as Record<string, unknown> | undefined
  return values?.[snake] === true || values?.[pascal] === true
}
const supportsTemplate = computed(() => capability('supports_template', 'SupportsTemplate'))
const supportsFields = computed(() => capability('supports_fields', 'SupportsFields'))
const ownsLibraryItems = computed(() => capability('owns_library_items', 'OwnsLibraryItems'))
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
    template_code: null,
    title: '',
    menu_title: '',
    slug: '',
    external_url: '',
    fields: {},
    type_settings: { item_url_pattern: '', default_item_template: null },
  })
  errorMessage.value = null
  serverFieldErrors.value = []
  localFieldErrors.value = {}
  visible.value = true
  metadataLoading.value = true
  try {
    metadata.value = await adminRequest<ResourceMetadata>(
      `/api/sites/${props.siteId}/resources/metadata`,
      props.accessToken,
    )
    form.type = metadata.value.types[0]?.code ?? 'link'
    form.template_code = null
    form.fields = createFieldValues(selectedTemplate.value?.fields ?? [])
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
    form.fields = createFieldValues(selectedTemplate.value?.fields ?? [])
    serverFieldErrors.value = []
    localFieldErrors.value = {}
  },
)

watch(
  () => form.type,
  (type) => {
    if (!visible.value) return
    if (type === 'link') form.fields = {}
    else form.fields = createFieldValues(selectedTemplate.value?.fields ?? [])
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
  if (form.type === 'link' && !form.external_url.trim()) {
    errorMessage.value = 'Укажите адрес ссылки.'
    return
  }
  const fields = supportsFields.value ? (selectedTemplate.value?.fields ?? []) : []
  if (unsupportedFieldTypes(fields).length > 0) {
    errorMessage.value =
      'Форма содержит неизвестные типы полей и не может быть отправлена.'
    return
  }
  localFieldErrors.value = validateFieldValues(fields, form.fields)
  if (Object.keys(localFieldErrors.value).length > 0) return
  const payload: ResourceCreatePayload = {
    parent_id: parent.value?.id ?? null,
    type: form.type,
    template_code: supportsTemplate.value ? form.template_code : null,
    title: form.title.trim(),
    menu_title: form.menu_title.trim(),
    slug: form.slug.trim(),
    fields: supportsFields.value ? { ...form.fields } : {},
    type_settings: ownsLibraryItems.value ? { ...form.type_settings } : {},
  }
  if (form.type === 'link') payload.external_url = form.external_url.trim()

  loading.value = true
  try {
    const created = await adminRequest<ResourceTreeItem>(
      `/api/sites/${props.siteId}/resources`,
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
      <el-form-item label="Тип" required>
        <el-select v-model="form.type" class="full-width">
          <el-option
            v-for="item in metadata.types"
            :key="item.code"
            :label="item.label"
            :value="item.code"
          />
        </el-select>
      </el-form-item>
      <el-form-item v-if="supportsTemplate" label="Шаблон">
        <el-select v-model="form.template_code" class="full-width">
          <el-option label="(без шаблона)" :value="noTemplateValue" />
          <el-option
            v-for="item in metadata.templates"
            :key="item.code"
            :label="item.label"
            :value="item.code"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="Название" required
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
      <el-form-item v-if="form.type === 'link'" label="Адрес ссылки" required>
        <el-input
          v-model="form.external_url"
          placeholder="https://example.com"
        />
      </el-form-item>
      <template v-if="ownsLibraryItems">
        <el-form-item label="Шаблон URL ресурса"><el-input v-model="form.type_settings.item_url_pattern" placeholder="/{slug}" /></el-form-item>
        <el-form-item label="Шаблон ресурса по умолчанию">
          <el-select v-model="form.type_settings.default_item_template" clearable class="full-width">
            <el-option v-for="item in metadata.templates" :key="item.code" :label="item.label" :value="item.code" />
          </el-select>
        </el-form-item>
      </template>
      <dynamic-fields-form
		v-if="supportsFields && selectedTemplate"
        :fields="selectedTemplate.fields"
        :model-value="form.fields"
        :errors="displayedFieldErrors"
        @update:model-value="form.fields = $event"
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
