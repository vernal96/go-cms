<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import {
  ElAlert,
  ElButton,
  ElDatePicker,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElSelect,
  ElSkeleton,
  ElSwitch,
  ElTabPane,
  ElTabs,
} from 'element-plus'
import { RefreshRight } from '@element-plus/icons-vue'
import { useRoute } from 'vue-router'

import { AdminAPIError, adminRequest, adminRequestVoid } from '../api/admin-api'
import DynamicFieldsForm from '../components/fields/DynamicFieldsForm.vue'
import TabbedDynamicFieldsForm from '../components/fields/TabbedDynamicFieldsForm.vue'
import RichTextEditor from '../components/RichTextEditor.vue'
import ResourceExtensionEditor from '../components/resource-extensions/ResourceExtensionEditor.vue'
import ResourceWidgetsEditor from '../components/resource-widgets/ResourceWidgetsEditor.vue'
import LibraryItemsTab from '../components/LibraryItemsTab.vue'
import ResourceHistoryTab from '../components/ResourceHistoryTab.vue'
import {
  createFieldValues,
  fieldErrorMessage,
  unsupportedFieldTypes,
  validateFieldValues,
  type DynamicFieldErrors,
} from '../components/fields/model'
import { generateResourceCode } from '../resource-code'
import type {
  ResourceDetailsResponse,
  ResourceMetadata,
  ResourceOption,
  ResourceOptionsResponse,
  ResourceUpdatePayload,
  ResourceWidget,
  ResourceTypeCapabilities,
  ResourceTypeCode,
  SiteDetailsResponse,
} from '../types/admin'
import type { FieldValidationError } from '../types/auth'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const route = useRoute()
const previousTitle = document.title

const loading = ref(true)
const submitting = ref(false)
const deleting = ref(false)
const loadError = ref<string | null>(null)
const submitError = ref<string | null>(null)
const activeTab = ref('main')
const metadata = ref<ResourceMetadata>({ types: [], templates: [], widgets: [], extensions: [] })
const options = ref<ResourceOption[]>([])
const canUpdate = ref(false)
const canDelete = ref(false)
const canRestore = ref(false)
const deleted = ref(false)
const deletedAt = ref<string | null>(null)
const serverFieldErrors = ref<FieldValidationError[]>([])
const localFieldErrors = ref<DynamicFieldErrors>({})
const localSettingsErrors = ref<DynamicFieldErrors>({})
const resourceWidgets = ref<ResourceWidget[]>([])
const resourcePath = ref<string | null>(null)
const resourceVersion = ref(0)
const canReadHistory = ref(false)
const canDeleteHistory = ref(false)
const siteDomain = ref('')
const noTemplateValue = '__no_template__'

const form = reactive({
  parent_id: null as number | null,
  type: 'page' as ResourceTypeCode,
  template_code: null as string | null,
  title: '',
  annotation: '',
  menu_title: '',
  slug: '',
  content: '',
  external_url: '',
  target_resource_id: null as number | null,
  content_type: null as string | null,
  is_public: true,
  is_searchable: true,
  in_menu: true,
  in_sitemap: true,
  position: 1,
  published_at: null as Date | null,
  unpublished_at: null as Date | null,
  fields: {} as Record<string, unknown>,
  type_settings: {} as Record<string, unknown>,
})
const templateSelection = computed(() => form.template_code ?? noTemplateValue)

const resourceId = computed(() => Number(route.params.resourceId))
const siteId = computed(() => Number(route.params.siteId))
const selectedTemplate = computed(() =>
  metadata.value.templates.find((item) => item.code === form.template_code) ?? null,
)
const selectedType = computed(() => metadata.value.types.find((item) => item.code === form.type) ?? null)
const settingsFields = computed(() => selectedType.value?.settings_fields ?? [])
const contentTypes = computed(() => selectedType.value?.content_types ?? [])
const selectedContentType = computed(() => contentTypes.value.find((item) => item.code === form.content_type) ?? null)
const contentEditor = computed(() => selectedContentType.value?.editor ?? 'textarea')
function capability(name: keyof ResourceTypeCapabilities): boolean {
  return selectedType.value?.capabilities[name] === true
}
const showWidgetsTab = computed(() =>
  capability('supports_widgets') && selectedTemplate.value?.supports_resource_widgets === true,
)
const showFieldsTab = computed(() =>
  capability('supports_fields') && (selectedTemplate.value?.fields.length ?? 0) > 0,
)
const supportsTemplate = computed(() => capability('supports_template'))
const supportsContent = computed(() => capability('supports_content'))
const supportsExternalURL = computed(() => capability('supports_external_url'))
const supportsTargetResource = computed(() => capability('supports_target_resource'))
const ownsLibraryItems = computed(() => capability('owns_library_items'))
const mutableType = computed(() => capability('mutable_type'))
const applicableExtensions = computed(() =>
  metadata.value.extensions.filter((extension) => extension.applies_to.includes(form.type)),
)
const hideInMenu = computed({
  get: () => !form.in_menu,
  set: (value: boolean) => { form.in_menu = !value },
})
const displayedFieldErrors = computed<DynamicFieldErrors>(() => {
  const result = { ...localFieldErrors.value }
  for (const error of serverFieldErrors.value) result[error.key] = fieldErrorMessage(error.rule, error.param)
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
  try {
    const [details, loadedMetadata, loadedOptions, loadedSite] = await Promise.all([
      adminRequest<ResourceDetailsResponse>(`/api/sites/${siteId.value}/resources/${resourceId.value}`, props.accessToken),
      adminRequest<ResourceMetadata>(`/api/sites/${siteId.value}/resources/metadata`, props.accessToken),
      adminRequest<ResourceOptionsResponse>(`/api/sites/${siteId.value}/resources/options`, props.accessToken),
      adminRequest<SiteDetailsResponse>(`/api/sites/${siteId.value}`, props.accessToken),
    ])
    metadata.value = loadedMetadata
    options.value = loadedOptions.items
    canUpdate.value = details.permissions.update
    canDelete.value = details.permissions.delete
		canRestore.value = details.permissions.restore
		canReadHistory.value = details.permissions.history_read
		canDeleteHistory.value = details.permissions.history_delete
    const item = details.resource
		resourcePath.value = item.path
		resourceVersion.value = item.version
    siteDomain.value = loadedSite.site.domain
    deleted.value = item.deleted
    deletedAt.value = item.deleted_at
    resourceWidgets.value = item.widgets ?? []
    Object.assign(form, {
      parent_id: item.parent_id,
      type: item.type,
      template_code: item.template_code,
      title: item.title,
      annotation: item.annotation,
      menu_title: item.menu_title,
      slug: item.slug,
      content: item.content,
      external_url: item.external_url ?? '',
      target_resource_id: item.target_resource_id,
      content_type: item.content_type,
      is_public: item.is_public,
      is_searchable: item.is_searchable,
      in_menu: item.in_menu,
      in_sitemap: item.in_sitemap,
      position: item.sort + 1,
      published_at: item.published_at ? new Date(item.published_at) : null,
      unpublished_at: item.unpublished_at ? new Date(item.unpublished_at) : null,
      fields: createFieldValues(
        loadedMetadata.templates.find((template) => template.code === item.template_code)?.fields ?? [],
        item.fields,
      ),
		type_settings: createFieldValues(
			loadedMetadata.types.find((type) => type.code === item.type)?.settings_fields ?? [],
			{
				...(loadedMetadata.types.find((type) => type.code === item.type)?.settings_defaults ?? {}),
				...item.type_settings,
			},
		),
    })
    document.title = `${item.title} — Админка`
  } catch (error) {
    handleError(error, 'Не удалось загрузить ресурс.')
  } finally {
    loading.value = false
  }
}

async function changeType(value: ResourceTypeCode): Promise<void> {
  if (value === form.type) return
  const nextType = metadata.value.types.find((item) => item.code === value)
  if (!mutableType.value || nextType?.capabilities.mutable_type !== true) return
  if (nextType.capabilities.supports_widgets !== true && resourceWidgets.value.length) {
    ElMessage.error('Сначала удалите виджеты, затем измените тип ресурса.')
    return
  }
  try {
    await ElMessageBox.confirm(
      'При смене типа несовместимые данные и настройки будут полностью очищены.',
      'Сменить тип ресурса?',
      { type: 'warning', confirmButtonText: 'Сменить', cancelButtonText: 'Отмена' },
    )
  } catch { return }
  form.type = value
  serverFieldErrors.value = []
  localFieldErrors.value = {}
	localSettingsErrors.value = {}
  form.template_code = null
  form.fields = {}
	form.type_settings = createFieldValues(nextType.settings_fields ?? [], nextType.settings_defaults ?? {})
  if (nextType.capabilities.supports_content !== true) {
    form.content_type = null
    form.content = ''
	} else form.content_type = nextType.content_types?.length === 1 ? nextType.content_types[0]!.code : null
  if (nextType.capabilities.supports_external_url !== true) form.external_url = ''
  if (nextType.capabilities.supports_target_resource !== true) form.target_resource_id = null
	if (activeTab.value.startsWith('extension:') || activeTab.value === 'fields') activeTab.value = 'main'
}

async function changeTemplate(selectedValue: string): Promise<void> {
	const value = selectedValue === noTemplateValue ? null : selectedValue
  if (value === form.template_code) return
  const nextTemplate = metadata.value.templates.find((item) => item.code === value)
  if (resourceWidgets.value.length && !nextTemplate?.supports_resource_widgets) {
    ElMessage.error('Сначала удалите виджеты: выбранный шаблон не содержит области для виджетов.')
    return
  }
  try {
    await ElMessageBox.confirm(
      'Параметры предыдущего шаблона будут полностью очищены.',
      'Сменить шаблон?',
      { type: 'warning', confirmButtonText: 'Сменить', cancelButtonText: 'Отмена' },
    )
  } catch { return }
  form.template_code = value
  form.fields = createFieldValues(selectedTemplate.value?.fields ?? [])
  serverFieldErrors.value = []
  localFieldErrors.value = {}
	localSettingsErrors.value = {}
  if (!selectedTemplate.value?.supports_resource_widgets && activeTab.value === 'widgets') activeTab.value = 'main'
  if (!selectedTemplate.value?.fields.length && activeTab.value === 'fields') activeTab.value = 'main'
}

function generateCode(): void {
  const generated = generateResourceCode(form.title)
  if (!generated) {
    ElMessage.warning('Не удалось сформировать код из заголовка')
    return
  }
  form.slug = generated
}

function viewResource(): void {
  if (!siteDomain.value) return
  const path = resourcePath.value?.startsWith('/')
    ? resourcePath.value
    : `/${resourcePath.value ?? ''}`
  window.open(`${window.location.protocol}//${siteDomain.value}${path || '/'}`, '_blank', 'noopener,noreferrer')
}

async function submit(): Promise<void> {
  submitError.value = null
  serverFieldErrors.value = []
  localFieldErrors.value = {}
  if (!form.title.trim()) {
    submitError.value = 'Заполните заголовок ресурса.'
    activeTab.value = 'main'
    return
  }
  if (supportsExternalURL.value && !form.external_url.trim()) {
    submitError.value = 'Укажите внешний URL.'
    activeTab.value = 'main'
    return
  }
  if (supportsTargetResource.value && form.target_resource_id === null) {
    submitError.value = 'Выберите целевой ресурс.'
    activeTab.value = 'main'
    return
  }
  if (form.published_at && form.unpublished_at && form.unpublished_at <= form.published_at) {
    submitError.value = 'Дата окончания публикации должна быть позже даты начала.'
    activeTab.value = 'settings'
    return
  }
  const fields = capability('supports_fields') ? (selectedTemplate.value?.fields ?? []) : []
  if (unsupportedFieldTypes(fields).length) {
    submitError.value = 'Форма содержит неизвестные типы полей и не может быть отправлена.'
    activeTab.value = showFieldsTab.value ? 'fields' : 'main'
    return
  }
  localFieldErrors.value = validateFieldValues(fields, form.fields)
  if (Object.keys(localFieldErrors.value).length) {
    activeTab.value = showFieldsTab.value ? 'fields' : 'main'
    return
  }
	if (unsupportedFieldTypes(settingsFields.value).length) {
		submitError.value = 'Настройки типа содержат неизвестные типы полей.'
		activeTab.value = 'settings'
		return
	}
	localSettingsErrors.value = validateFieldValues(settingsFields.value, form.type_settings)
	if (Object.keys(localSettingsErrors.value).length) {
		activeTab.value = 'settings'
		return
	}

	const payload: ResourceUpdatePayload = {
		expected_version: resourceVersion.value,
    parent_id: form.parent_id,
    type: form.type,
    template_code: supportsTemplate.value ? form.template_code : null,
    title: form.title.trim(),
    annotation: form.annotation,
    menu_title: form.menu_title.trim(),
    slug: form.slug.trim(),
    content_type: supportsContent.value ? form.content_type : null,
    content: supportsContent.value ? form.content : '',
    target_resource_id: supportsTargetResource.value ? form.target_resource_id : null,
    external_url: supportsExternalURL.value ? form.external_url.trim() : null,
    is_public: form.is_public,
    is_searchable: form.is_searchable,
    in_menu: form.in_menu,
    in_sitemap: form.in_sitemap,
    sort: Math.max(0, form.position - 1),
    published_at: form.published_at?.toISOString() ?? null,
    unpublished_at: form.unpublished_at?.toISOString() ?? null,
    fields: capability('supports_fields') ? { ...form.fields } : {},
    type_settings: { ...form.type_settings },
  }
  submitting.value = true
  try {
    const response = await adminRequest<ResourceDetailsResponse>(
      `/api/sites/${siteId.value}/resources/${resourceId.value}`,
      props.accessToken,
      { method: 'PATCH', body: JSON.stringify(payload) },
    )
		form.slug = response.resource.slug
		resourceVersion.value = response.resource.version
    form.position = response.resource.sort + 1
    form.fields = createFieldValues(fields, response.resource.fields)
    document.title = `${response.resource.title} — Админка`
    notifyTreeChanged()
    ElMessage.success('Ресурс сохранён')
  } catch (error) {
    if (error instanceof AdminAPIError) serverFieldErrors.value = error.fieldErrors
    handleError(error, 'Не удалось сохранить ресурс.', true)
  } finally {
    submitting.value = false
  }
}

async function changeDeleted(next: boolean): Promise<void> {
  if (deleting.value || !canDelete.value || (!next && !canRestore.value)) return
  try {
    await ElMessageBox.confirm(
      next
        ? 'Ресурс и все его потомки будут помечены удалёнными.'
        : 'Будет восстановлен только текущий ресурс.',
      next ? 'Удалить ресурс?' : 'Восстановить ресурс?',
      { type: 'warning', confirmButtonText: next ? 'Удалить' : 'Восстановить', cancelButtonText: 'Отмена' },
    )
  } catch { return }
  deleting.value = true
  try {
    if (next) {
      await adminRequestVoid(`/api/sites/${siteId.value}/resources/${resourceId.value}`, props.accessToken, { method: 'DELETE' })
    } else {
      await adminRequestVoid(`/api/sites/${siteId.value}/resources/${resourceId.value}/restore`, props.accessToken, {
        method: 'POST', body: JSON.stringify({ with_descendants: false }),
      })
    }
    notifyTreeChanged()
    await load()
    ElMessage.success(next ? 'Ресурс удалён' : 'Ресурс восстановлен')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Не удалось изменить статус ресурса.')
  } finally {
    deleting.value = false
  }
}

function notifyTreeChanged(): void {
  window.dispatchEvent(new CustomEvent('admin:resource-tree-changed', { detail: { siteId: siteId.value } }))
}

function handleError(error: unknown, fallback: string, submitRequest = false): void {
  if (error instanceof AdminAPIError && error.status === 401) { emit('unauthorized'); return }
  const message = error instanceof Error ? error.message : fallback
  if (submitRequest) submitError.value = message
  else loadError.value = message
}

function descendantIDs(items: ResourceOption[], rootID: number): Set<number> {
  const result = new Set<number>([rootID])
  let changed = true
  while (changed) {
    changed = false
    for (const item of items) {
      if (item.parent_id !== null && result.has(item.parent_id) && !result.has(item.id)) {
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
onBeforeUnmount(() => { document.title = previousTitle })
watch(() => [route.params.siteId, route.params.resourceId], () => void load())
</script>

<template>
  <section class="workspace-page resource-edit-page">
    <header class="page-header resource-page-header">
      <div>
        <h1>{{ form.title || 'Ресурс' }}</h1>
        <p v-if="deleted" class="resource-deleted-caption">Удалён {{ deletedAt ? new Date(deletedAt).toLocaleString() : '' }}</p>
        <p v-else>Редактирование ресурса</p>
      </div>
      <div class="page-header-actions">
        <el-button :disabled="!siteDomain" @click="viewResource">Посмотреть</el-button>
        <el-button type="primary" :loading="submitting" :disabled="!canUpdate || loading" @click="submit">
          Сохранить
        </el-button>
      </div>
    </header>

    <el-alert v-if="loadError" type="error" :closable="false" :title="loadError" show-icon />
    <el-skeleton v-else-if="loading" :rows="10" animated />
    <el-form v-else :model="form" label-position="top" class="resource-editor-form" :class="{ 'is-readonly': !canUpdate }">
      <el-alert v-if="submitError" class="form-alert" type="error" :closable="false" :title="submitError" show-icon />
      <el-tabs v-model="activeTab" class="resource-tabs">
        <el-tab-pane label="Основное" name="main">
          <div class="resource-main-grid">
            <div class="resource-main-primary">
              <el-form-item label="Заголовок" required><el-input v-model="form.title" :disabled="!canUpdate" /></el-form-item>
              <el-form-item label="Аннотация (введение)"><el-input v-model="form.annotation" type="textarea" :rows="7" :disabled="!canUpdate" /></el-form-item>
            </div>
            <div class="resource-main-secondary">
              <el-form-item v-if="supportsTemplate" label="Шаблон">
                <el-select :model-value="templateSelection" class="full-width" :disabled="!canUpdate" @change="changeTemplate">
                  <el-option label="(без шаблона)" :value="noTemplateValue" />
                  <el-option v-for="item in metadata.templates" :key="item.code" :label="item.label" :value="item.code" />
                </el-select>
              </el-form-item>
              <el-form-item label="Код">
                <el-input v-model="form.slug" :disabled="!canUpdate">
                  <template #append><el-button :disabled="!canUpdate" aria-label="Сформировать код по заголовку" title="Сформировать по заголовку" @click="generateCode"><el-icon><RefreshRight /></el-icon></el-button></template>
                </el-input>
              </el-form-item>
              <el-form-item label="Пункт меню"><el-input v-model="form.menu_title" :disabled="!canUpdate" /></el-form-item>
              <div class="resource-switches">
                <el-form-item label="Опубликован"><el-switch v-model="form.is_public" :disabled="!canUpdate" /></el-form-item>
                <el-form-item label="Не показывать в меню"><el-switch v-model="hideInMenu" :disabled="!canUpdate" /></el-form-item>
              </div>
            </div>
          </div>
			<el-form-item v-if="supportsContent && contentTypes.length > 1" label="Тип содержимого" required class="resource-content-field">
				<el-select v-model="form.content_type" class="full-width" :disabled="!canUpdate || deleted">
					<el-option v-for="item in contentTypes" :key="item.code" :label="item.label" :value="item.code" />
				</el-select>
			</el-form-item>
          <el-form-item v-if="supportsContent" label="Контент" class="resource-content-field">
			<rich-text-editor v-if="contentEditor === 'html'" v-model="form.content" :disabled="!canUpdate" />
			<el-input v-else v-model="form.content" type="textarea" :rows="8" :disabled="!canUpdate" />
          </el-form-item>
          <el-form-item v-if="supportsExternalURL" label="Внешний URL" required class="resource-content-field">
            <el-input v-model="form.external_url" placeholder="https://example.com" :disabled="!canUpdate" />
          </el-form-item>
          <el-form-item v-if="supportsTargetResource" label="Целевой ресурс" required class="resource-content-field">
            <el-select v-model="form.target_resource_id" filterable class="full-width" :disabled="!canUpdate">
              <el-option v-for="item in options.filter((candidate) => candidate.id !== resourceId)" :key="item.id" :label="item.display_title" :value="item.id" />
            </el-select>
          </el-form-item>
        </el-tab-pane>

        <el-tab-pane v-if="showWidgetsTab" label="Виджеты" name="widgets">
          <resource-widgets-editor
            v-if="selectedTemplate"
            v-model="resourceWidgets"
            :access-token="accessToken"
            :site-id="siteId"
            :resource-id="resourceId"
              :template="selectedTemplate!"
            :definitions="metadata.widgets"
			:can-update="canUpdate && !deleted"
			:resource-version="resourceVersion"
			@changed="load"
            @unauthorized="emit('unauthorized')"
          />
        </el-tab-pane>

        <el-tab-pane label="Настройки" name="settings">
          <div class="resource-settings-grid">
            <div>
              <el-form-item label="Родительский ресурс">
                <el-select v-model="form.parent_id" clearable class="full-width" :disabled="!canUpdate || deleted">
                  <el-option v-for="item in parentOptions" :key="item.id" :label="item.indentedTitle" :value="item.id" />
                </el-select>
              </el-form-item>
              <el-form-item label="Тип ресурса">
                <el-select :model-value="form.type" class="full-width" :disabled="!canUpdate || deleted || !mutableType" @change="changeType">
                  <el-option v-for="item in metadata.types" :key="item.code" :label="item.label" :value="item.code" />
                </el-select>
              </el-form-item>
				<el-form-item label="Тип содержимого"><el-input :model-value="selectedContentType?.label ?? form.content_type ?? 'не используется'" disabled /></el-form-item>
              <el-form-item label="Позиция в меню"><el-input-number v-model="form.position" :min="1" :step="1" class="full-width" :disabled="!canUpdate || deleted" /></el-form-item>
				<dynamic-fields-form
					v-if="settingsFields.length"
					v-model="form.type_settings"
					:fields="settingsFields"
					:errors="localSettingsErrors"
					:site-id="siteId"
					:access-token="accessToken"
					:resource-templates="metadata.templates"
				/>
            </div>
            <div>
              <el-form-item label="Начало публикации"><el-date-picker v-model="form.published_at" type="datetime" class="full-width" format="DD.MM.YYYY HH:mm" :disabled="!canUpdate" /></el-form-item>
              <el-form-item label="Окончание публикации"><el-date-picker v-model="form.unpublished_at" type="datetime" class="full-width" format="DD.MM.YYYY HH:mm" :disabled="!canUpdate" /></el-form-item>
              <el-form-item label="Доступен для поиска"><el-switch v-model="form.is_searchable" :disabled="!canUpdate" /></el-form-item>
              <el-form-item label="Удалён">
                <el-switch :model-value="deleted" :loading="deleting" :disabled="!canDelete || (deleted && !canRestore)" @change="changeDeleted" />
                <span v-if="deleted && !canRestore" class="field-help">Сначала восстановите родительский ресурс</span>
              </el-form-item>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="showFieldsTab" label="Параметры полей" name="fields">
          <div :class="{ 'dynamic-fields-readonly': !canUpdate }">
            <tabbed-dynamic-fields-form
              v-model="form.fields"
              :fields="selectedTemplate!.fields"
              :editor-tabs="selectedTemplate!.editor_tabs"
              :errors="displayedFieldErrors"
              :site-id="siteId"
              :access-token="accessToken"
            />
          </div>
        </el-tab-pane>

		<el-tab-pane v-if="ownsLibraryItems" label="Ресурсы" name="library-items">
          <library-items-tab :access-token="accessToken" :site-id="siteId" :library-id="resourceId" />
		</el-tab-pane>

		<el-tab-pane v-if="canReadHistory" label="История" name="history">
			<resource-history-tab :access-token="accessToken" :site-id="siteId" :resource-id="resourceId" :resource-version="resourceVersion" :can-restore="canUpdate && !deleted" :can-delete="canDeleteHistory" @changed="load" @unauthorized="emit('unauthorized')" />
		</el-tab-pane>

			<el-tab-pane
				v-for="extension in applicableExtensions"
				:key="extension.code"
				:label="extension.title"
				:name="`extension:${extension.code}`"
			>
				<resource-extension-editor
					:key="`${resourceId}:${extension.code}`"
					:metadata="extension"
					:site-id="siteId"
					:resource-id="resourceId"
					:access-token="accessToken"
					:can-update="canUpdate"
					@unauthorized="emit('unauthorized')"
				/>
			</el-tab-pane>
      </el-tabs>
    </el-form>
  </section>
</template>
