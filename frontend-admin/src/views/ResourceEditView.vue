<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import {
  ElAlert,
  ElButton,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElSelect,
  ElSkeleton,
  ElSwitch,
} from 'element-plus'
import { useRoute, useRouter } from 'vue-router'

import { AdminAPIError, adminRequest } from '../api/admin-api'
import DynamicFieldsForm from '../components/fields/DynamicFieldsForm.vue'
import {
  createFieldValues,
  fieldErrorMessage,
  unsupportedFieldTypes,
  validateFieldValues,
  type DynamicFieldErrors,
} from '../components/fields/model'
import type {
  ResourceDetailsResponse,
  ResourceMetadata,
  ResourceOption,
  ResourceOptionsResponse,
  ResourceUpdatePayload,
} from '../types/admin'
import type { FieldValidationError } from '../types/auth'

const props = defineProps<{
  accessToken: string
  permissions: ReadonlySet<string>
}>()
const emit = defineEmits<{ unauthorized: [] }>()
const route = useRoute()
const router = useRouter()
const loading = ref(true)
const submitting = ref(false)
const loadError = ref<string | null>(null)
const submitError = ref<string | null>(null)
const serverFieldErrors = ref<FieldValidationError[]>([])
const localFieldErrors = ref<DynamicFieldErrors>({})
const metadata = ref<ResourceMetadata>({ types: [], templates: [] })
const options = ref<ResourceOption[]>([])
const canUpdate = ref(false)

const form = reactive({
  parent_id: null as number | null,
  type: 'page' as 'page' | 'link',
  template_code: '',
  title: '',
  menu_title: '',
  slug: '',
  content: '',
  external_url: '',
  is_public: true,
  is_searchable: true,
  in_menu: true,
  in_sitemap: true,
  sort: 0,
  settings: {} as Record<string, unknown>,
})

const resourceId = computed(() => Number(route.params.resourceId))
const siteId = computed(() => Number(route.params.siteId))
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
const parentOptions = computed(() => {
  const excluded = descendantIDs(options.value, resourceId.value)
  return options.value
    .filter((item) => !excluded.has(item.id))
    .map((item) => ({
      ...item,
      indentedTitle: `${'— '.repeat(optionDepth(item, options.value))}${item.display_title}`,
    }))
})

async function load(): Promise<void> {
  loading.value = true
  loadError.value = null
  submitError.value = null
  try {
    const [details, loadedMetadata, loadedOptions] = await Promise.all([
      adminRequest<ResourceDetailsResponse>(
        `/api/admin/sites/${siteId.value}/resources/${resourceId.value}`,
        props.accessToken,
      ),
      adminRequest<ResourceMetadata>(
        `/api/admin/sites/${siteId.value}/resource-metadata`,
        props.accessToken,
      ),
      adminRequest<ResourceOptionsResponse>(
        `/api/admin/sites/${siteId.value}/resource-options`,
        props.accessToken,
      ),
    ])
    metadata.value = loadedMetadata
    options.value = loadedOptions.items
    canUpdate.value = details.permissions.update
    const resource = details.resource
    Object.assign(form, {
      parent_id: resource.parent_id,
      type: resource.type,
      template_code: resource.template_code ?? '',
      title: resource.title,
      menu_title: resource.menu_title,
      slug: resource.slug,
      content: resource.content,
      external_url: resource.external_url ?? '',
      is_public: resource.is_public,
      is_searchable: resource.is_searchable,
      in_menu: resource.in_menu,
      in_sitemap: resource.in_sitemap,
      sort: resource.sort,
      settings: createFieldValues(
        loadedMetadata.templates.find(
          (item) => item.code === resource.template_code,
        )?.fields ?? [],
        resource.settings,
      ),
    })
  } catch (error) {
    handleError(error, 'Не удалось загрузить ресурс.')
  } finally {
    loading.value = false
  }
}

async function changeType(value: 'page' | 'link'): Promise<void> {
  if (value === form.type) return
  try {
    await ElMessageBox.confirm(
      'При смене типа несовместимые данные и настройки будут полностью очищены.',
      'Сменить тип ресурса?',
      {
        type: 'warning',
        confirmButtonText: 'Сменить',
        cancelButtonText: 'Отмена',
      },
    )
  } catch {
    return
  }
  form.type = value
  serverFieldErrors.value = []
  localFieldErrors.value = {}
  if (value === 'link') {
    form.template_code = ''
    form.content = ''
    form.settings = {}
  } else {
    form.external_url = ''
    form.template_code = metadata.value.templates[0]?.code ?? ''
    form.settings = createFieldValues(selectedTemplate.value?.fields ?? [])
  }
}

async function changeTemplate(value: string): Promise<void> {
  if (value === form.template_code) return
  try {
    await ElMessageBox.confirm(
      'Настройки предыдущего шаблона будут полностью очищены.',
      'Сменить шаблон?',
      {
        type: 'warning',
        confirmButtonText: 'Сменить',
        cancelButtonText: 'Отмена',
      },
    )
  } catch {
    return
  }
  form.template_code = value
  form.settings = createFieldValues(selectedTemplate.value?.fields ?? [])
  serverFieldErrors.value = []
  localFieldErrors.value = {}
}

async function submit(): Promise<void> {
  submitError.value = null
  serverFieldErrors.value = []
  localFieldErrors.value = {}
  if (!form.title.trim() || (form.parent_id !== null && !form.slug.trim())) {
    submitError.value = 'Заполните название и slug дочернего ресурса.'
    return
  }
  if (form.type === 'page' && !form.template_code) {
    submitError.value = 'Выберите шаблон страницы.'
    return
  }
  if (form.type === 'link' && !form.external_url.trim()) {
    submitError.value = 'Укажите адрес ссылки.'
    return
  }
  const fields =
    form.type === 'page' ? (selectedTemplate.value?.fields ?? []) : []
  if (unsupportedFieldTypes(fields).length > 0) {
    submitError.value =
      'Форма содержит неизвестные типы полей и не может быть отправлена.'
    return
  }
  localFieldErrors.value = validateFieldValues(fields, form.settings)
  if (Object.keys(localFieldErrors.value).length > 0) return

  const payload: ResourceUpdatePayload = {
    parent_id: form.parent_id,
    type: form.type,
    template_code: form.type === 'page' ? form.template_code : null,
    title: form.title.trim(),
    menu_title: form.menu_title.trim(),
    slug: form.slug.trim(),
    content: form.type === 'page' ? form.content : '',
    external_url: form.type === 'link' ? form.external_url.trim() : null,
    is_public: form.is_public,
    is_searchable: form.is_searchable,
    in_menu: form.in_menu,
    in_sitemap: form.in_sitemap,
    sort: form.sort,
    settings: form.type === 'page' ? { ...form.settings } : {},
  }
  submitting.value = true
  try {
    const response = await adminRequest<ResourceDetailsResponse>(
      `/api/admin/sites/${siteId.value}/resources/${resourceId.value}`,
      props.accessToken,
      { method: 'PATCH', body: JSON.stringify(payload) },
    )
    form.settings = createFieldValues(fields, response.resource.settings)
    ElMessage.success('Ресурс сохранён')
  } catch (error) {
    if (error instanceof AdminAPIError)
      serverFieldErrors.value = error.fieldErrors
    handleError(error, 'Не удалось сохранить ресурс.', true)
  } finally {
    submitting.value = false
  }
}

function handleError(
  error: unknown,
  fallback: string,
  submittingRequest = false,
): void {
  if (error instanceof AdminAPIError && error.status === 401) {
    emit('unauthorized')
    return
  }
  const message = error instanceof Error ? error.message : fallback
  if (submittingRequest) submitError.value = message
  else loadError.value = message
}

function descendantIDs(items: ResourceOption[], rootID: number): Set<number> {
  const result = new Set<number>([rootID])
  let changed = true
  while (changed) {
    changed = false
    for (const item of items) {
      if (
        item.parent_id !== null &&
        result.has(item.parent_id) &&
        !result.has(item.id)
      ) {
        result.add(item.id)
        changed = true
      }
    }
  }
  return result
}

function optionDepth(item: ResourceOption, items: ResourceOption[]): number {
  const byID = new Map(items.map((candidate) => [candidate.id, candidate]))
  const seen = new Set<number>([item.id])
  let depth = 0
  let parentID = item.parent_id
  while (parentID !== null && !seen.has(parentID)) {
    seen.add(parentID)
    depth += 1
    parentID = byID.get(parentID)?.parent_id ?? null
  }
  return depth
}

onMounted(() => void load())
watch(
  () => [route.params.siteId, route.params.resourceId],
  () => void load(),
)
</script>

<template>
  <section class="workspace-page resource-edit-page">
    <header class="page-header">
      <div>
        <h1>Редактор ресурса</h1>
        <p>Основные свойства и поля шаблона</p>
      </div>
      <el-button @click="router.push('/admin/sites')">К сайтам</el-button>
    </header>
    <el-skeleton v-if="loading" animated :rows="10" />
    <el-alert
      v-else-if="loadError"
      type="error"
      :closable="false"
      :title="loadError"
    />
    <el-form
      v-else
      class="resource-edit-form"
      label-position="top"
      :model="form"
      @submit.prevent="submit"
    >
      <el-alert
        v-if="submitError"
        class="form-alert"
        type="error"
        :closable="false"
        :title="submitError"
      />
      <div class="resource-form-grid">
        <el-form-item label="Родитель">
          <el-select
            v-model="form.parent_id"
            class="full-width"
            clearable
            placeholder="Корневой ресурс"
            @clear="form.parent_id = null"
          >
            <el-option
              v-for="item in parentOptions"
              :key="item.id"
              :label="item.indentedTitle"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Тип" required>
          <el-select
            :model-value="form.type"
            class="full-width"
            @change="changeType"
          >
            <el-option
              v-for="item in metadata.types"
              :key="item.code"
              :label="item.label"
              :value="item.code"
            />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.type === 'page'" label="Шаблон" required>
          <el-select
            :model-value="form.template_code"
            class="full-width"
            @change="changeTemplate"
          >
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
        <el-form-item label="Slug"
          ><el-input v-model="form.slug"
        /></el-form-item>
        <el-form-item label="Сортировка"
          ><el-input-number v-model="form.sort" class="full-width" :step="1"
        /></el-form-item>
      </div>
      <el-form-item v-if="form.type === 'page'" label="Контент">
        <el-input v-model="form.content" type="textarea" :rows="8" />
      </el-form-item>
      <el-form-item v-else label="Внешний URL" required>
        <el-input
          v-model="form.external_url"
          placeholder="https://example.com"
        />
      </el-form-item>
      <div class="resource-flags">
        <el-form-item label="Публичный"
          ><el-switch v-model="form.is_public"
        /></el-form-item>
        <el-form-item label="В поиске"
          ><el-switch v-model="form.is_searchable"
        /></el-form-item>
        <el-form-item label="В меню"
          ><el-switch v-model="form.in_menu"
        /></el-form-item>
        <el-form-item label="В sitemap"
          ><el-switch v-model="form.in_sitemap"
        /></el-form-item>
      </div>
      <dynamic-fields-form
        v-if="form.type === 'page' && selectedTemplate"
        :fields="selectedTemplate.fields"
        :model-value="form.settings"
        :errors="displayedFieldErrors"
        @update:model-value="form.settings = $event"
      />
      <div class="form-actions">
        <el-button @click="router.push('/admin/sites')">Отмена</el-button>
        <el-button
          type="primary"
          native-type="submit"
          :loading="submitting"
          :disabled="!canUpdate"
        >
          Сохранить
        </el-button>
      </div>
    </el-form>
  </section>
</template>
