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
import RichTextEditor from '../components/RichTextEditor.vue'
import ResourceExtensionEditor from '../components/resource-extensions/ResourceExtensionEditor.vue'
import ResourceWidgetsEditor from '../components/resource-widgets/ResourceWidgetsEditor.vue'
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
const resourceWidgets = ref<ResourceWidget[]>([])
const resourcePath = ref<string | null>(null)
const siteDomain = ref('')
const noTemplateValue = null as unknown as string

const form = reactive({
  parent_id: null as number | null,
  type: 'page' as 'page' | 'link',
  template_code: null as string | null,
  title: '',
  annotation: '',
  menu_title: '',
  slug: '',
  content: '',
  external_url: '',
  is_public: true,
  is_searchable: true,
  in_menu: true,
  in_sitemap: true,
  position: 1,
  published_at: null as Date | null,
  unpublished_at: null as Date | null,
  settings: {} as Record<string, unknown>,
})

const resourceId = computed(() => Number(route.params.resourceId))
const siteId = computed(() => Number(route.params.siteId))
const selectedTemplate = computed(() =>
  metadata.value.templates.find((item) => item.code === form.template_code) ?? null,
)
const showWidgetsTab = computed(() =>
  form.type === 'page' && selectedTemplate.value?.supports_resource_widgets === true,
)
const showFieldsTab = computed(() =>
  form.type === 'page' && (selectedTemplate.value?.fields.length ?? 0) > 0,
)
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
      adminRequest<ResourceDetailsResponse>(`/api/admin/sites/${siteId.value}/resources/${resourceId.value}`, props.accessToken),
      adminRequest<ResourceMetadata>(`/api/admin/sites/${siteId.value}/resource-metadata`, props.accessToken),
      adminRequest<ResourceOptionsResponse>(`/api/admin/sites/${siteId.value}/resource-options`, props.accessToken),
      adminRequest<SiteDetailsResponse>(`/api/admin/sites/${siteId.value}`, props.accessToken),
    ])
    metadata.value = loadedMetadata
    options.value = loadedOptions.items
    canUpdate.value = details.permissions.update
    canDelete.value = details.permissions.delete
    canRestore.value = details.permissions.restore
    const item = details.resource
    resourcePath.value = item.path
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
      is_public: item.is_public,
      is_searchable: item.is_searchable,
      in_menu: item.in_menu,
      in_sitemap: item.in_sitemap,
      position: item.sort + 1,
      published_at: item.published_at ? new Date(item.published_at) : null,
      unpublished_at: item.unpublished_at ? new Date(item.unpublished_at) : null,
      settings: createFieldValues(
        loadedMetadata.templates.find((template) => template.code === item.template_code)?.fields ?? [],
        item.settings,
      ),
    })
    document.title = `${item.title} — Админка`
  } catch (error) {
    handleError(error, 'Не удалось загрузить ресурс.')
  } finally {
    loading.value = false
  }
}

async function changeType(value: 'page' | 'link'): Promise<void> {
  if (value === form.type) return
  if (value === 'link' && resourceWidgets.value.length) {
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
  if (value === 'link') {
    form.template_code = null
    form.content = ''
    form.settings = {}
		if (activeTab.value.startsWith('extension:') || activeTab.value === 'fields') activeTab.value = 'main'
  } else {
    form.external_url = ''
    form.template_code = null
    form.settings = {}
  }
}

async function changeTemplate(value: string | null): Promise<void> {
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
  form.settings = createFieldValues(selectedTemplate.value?.fields ?? [])
  serverFieldErrors.value = []
  localFieldErrors.value = {}
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
  if (form.type === 'link' && !form.external_url.trim()) {
    submitError.value = 'Укажите внешний URL.'
    activeTab.value = 'main'
    return
  }
  if (form.published_at && form.unpublished_at && form.unpublished_at <= form.published_at) {
    submitError.value = 'Дата окончания публикации должна быть позже даты начала.'
    activeTab.value = 'settings'
    return
  }
  const fields = form.type === 'page' ? (selectedTemplate.value?.fields ?? []) : []
  if (unsupportedFieldTypes(fields).length) {
    submitError.value = 'Форма содержит неизвестные типы полей и не может быть отправлена.'
    activeTab.value = showFieldsTab.value ? 'fields' : 'main'
    return
  }
  localFieldErrors.value = validateFieldValues(fields, form.settings)
  if (Object.keys(localFieldErrors.value).length) {
    activeTab.value = showFieldsTab.value ? 'fields' : 'main'
    return
  }

  const payload: ResourceUpdatePayload = {
    parent_id: form.parent_id,
    type: form.type,
    template_code: form.type === 'page' ? form.template_code : null,
    title: form.title.trim(),
    annotation: form.annotation,
    menu_title: form.menu_title.trim(),
    slug: form.slug.trim(),
    content_type: form.type === 'page' ? 'html' : null,
    content: form.type === 'page' ? form.content : '',
    external_url: form.type === 'link' ? form.external_url.trim() : null,
    is_public: form.is_public,
    is_searchable: form.is_searchable,
    in_menu: form.in_menu,
    in_sitemap: form.in_sitemap,
    sort: Math.max(0, form.position - 1),
    published_at: form.published_at?.toISOString() ?? null,
    unpublished_at: form.unpublished_at?.toISOString() ?? null,
    settings: form.type === 'page' ? { ...form.settings } : {},
  }
  submitting.value = true
  try {
    const response = await adminRequest<ResourceDetailsResponse>(
      `/api/admin/sites/${siteId.value}/resources/${resourceId.value}`,
      props.accessToken,
      { method: 'PATCH', body: JSON.stringify(payload) },
    )
    form.slug = response.resource.slug
    form.position = response.resource.sort + 1
    form.settings = createFieldValues(fields, response.resource.settings)
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
      await adminRequestVoid(`/api/admin/sites/${siteId.value}/resources/${resourceId.value}`, props.accessToken, { method: 'DELETE' })
    } else {
      await adminRequestVoid(`/api/admin/sites/${siteId.value}/resources/${resourceId.value}/restore`, props.accessToken, {
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
              <el-form-item v-if="form.type === 'page'" label="Шаблон">
                <el-select :model-value="form.template_code" class="full-width" :disabled="!canUpdate" @change="changeTemplate">
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
          <el-form-item v-if="form.type === 'page'" label="Контент" class="resource-content-field">
            <rich-text-editor v-model="form.content" :disabled="!canUpdate" />
          </el-form-item>
          <el-form-item v-else label="Внешний URL" required class="resource-content-field">
            <el-input v-model="form.external_url" placeholder="https://example.com" :disabled="!canUpdate" />
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
                <el-select :model-value="form.type" class="full-width" :disabled="!canUpdate || deleted" @change="changeType">
                  <el-option v-for="item in metadata.types" :key="item.code" :label="item.label" :value="item.code" />
                </el-select>
              </el-form-item>
              <el-form-item label="Тип содержимого"><el-input model-value="HTML" disabled /></el-form-item>
              <el-form-item label="Позиция в меню"><el-input-number v-model="form.position" :min="1" :step="1" class="full-width" :disabled="!canUpdate || deleted" /></el-form-item>
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
            <dynamic-fields-form v-model="form.settings" :fields="selectedTemplate!.fields" :errors="displayedFieldErrors" />
          </div>
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
