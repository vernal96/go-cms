<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElAlert, ElButton, ElDatePicker, ElForm, ElFormItem, ElInput, ElMessage, ElOption, ElSelect, ElSkeleton, ElSwitch, ElTabPane, ElTabs } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'

import { AdminAPIError, adminRequest } from '../api/admin-api'
import DynamicFieldsForm from '../components/fields/DynamicFieldsForm.vue'
import RichTextEditor from '../components/RichTextEditor.vue'
import ResourceExtensionEditor from '../components/resource-extensions/ResourceExtensionEditor.vue'
import ResourceWidgetsEditor from '../components/resource-widgets/ResourceWidgetsEditor.vue'
import ResourceHistoryTab from '../components/ResourceHistoryTab.vue'
import { createFieldValues, unsupportedFieldTypes, validateFieldValues, type DynamicFieldErrors } from '../components/fields/model'
import { generateResourceCode } from '../resource-code'
import type { LibraryItemDetailsResponse, LibraryItemPayload, ResourceDetailsResponse, ResourceMetadata, ResourceOptionsResponse, ResourceWidget } from '../types/admin'

const props = defineProps<{ accessToken: string }>()
const route = useRoute()
const router = useRouter()
const siteId = computed(() => Number(route.params.siteId))
const originalLibraryId = computed(() => Number(route.params.resourceId))
const itemId = computed(() => route.params.itemId ? Number(route.params.itemId) : null)
const creating = computed(() => itemId.value === null)
const loading = ref(true)
const submitting = ref(false)
const moving = ref(false)
const errorMessage = ref('')
const moveError = ref('')
const metadata = ref<ResourceMetadata>({ types: [], templates: [], widgets: [], extensions: [] })
const libraries = ref<Array<{ id: number; display_title: string }>>([])
const fieldErrors = ref<DynamicFieldErrors>({})
const canUpdate = ref(true)
const canReadHistory = ref(false)
const canDeleteHistory = ref(false)
const resourceWidgets = ref<ResourceWidget[]>([])
const resourceVersion = ref(0)
const ownerLibraryId = ref(0)
const activeTab = ref('main')
const form = reactive({ library_id: 0, template_code: null as string | null, title: '', slug: '', annotation: '', content: '', is_public: true, is_searchable: true, published_at: null as Date | null, unpublished_at: null as Date | null, fields: {} as Record<string, unknown> })
const selectedTemplate = computed(() => metadata.value.templates.find((item) => item.code === form.template_code) ?? null)
const applicableExtensions = computed(() => metadata.value.extensions.filter((extension) => extension.applies_to.includes('page')))
const showWidgetsTab = computed(() => !creating.value && selectedTemplate.value?.supports_resource_widgets === true)
const showFieldsTab = computed(() => (selectedTemplate.value?.fields.length ?? 0) > 0)
const visibleTabNames = computed(() => [
  'main',
  ...(showWidgetsTab.value ? ['widgets'] : []),
  'settings',
  ...(showFieldsTab.value ? ['fields'] : []),
  ...(!creating.value && canReadHistory.value ? ['history'] : []),
  ...(!creating.value ? applicableExtensions.value.map((extension) => `extension:${extension.code}`) : []),
])

watch(visibleTabNames, (names) => {
  if (!names.includes(activeTab.value)) activeTab.value = names[0] ?? 'main'
}, { immediate: true })

async function load(): Promise<void> {
  loading.value = true
  try {
    const [loadedMetadata, options, libraryDetails] = await Promise.all([
      adminRequest<ResourceMetadata>(`/api/sites/${siteId.value}/resources/metadata`, props.accessToken),
      adminRequest<ResourceOptionsResponse>(`/api/sites/${siteId.value}/resources/options`, props.accessToken),
      adminRequest<ResourceDetailsResponse>(`/api/sites/${siteId.value}/resources/${originalLibraryId.value}`, props.accessToken),
    ])
    metadata.value = loadedMetadata
    const libraryTypeCodes = new Set(
      loadedMetadata.types
        .filter((item) => item.capabilities.owns_library_items === true)
        .map((item) => item.code),
    )
    libraries.value = options.items.filter((item) => libraryTypeCodes.has(item.type))
    form.library_id = originalLibraryId.value
    ownerLibraryId.value = originalLibraryId.value
    if (creating.value) {
      const defaultTemplate = libraryDetails.resource.type_settings.default_item_template
      form.template_code = typeof defaultTemplate === 'string' ? defaultTemplate : null
      form.fields = createFieldValues(selectedTemplate.value?.fields ?? [])
    } else {
      const details = await adminRequest<LibraryItemDetailsResponse>(`/api/sites/${siteId.value}/library-items/${itemId.value}`, props.accessToken)
      const item = details.item
      resourceVersion.value = item.version
      ownerLibraryId.value = item.library_id
      canUpdate.value = details.permissions.update
      canReadHistory.value = details.permissions.history_read
      canDeleteHistory.value = details.permissions.history_delete
      resourceWidgets.value = item.widgets
      Object.assign(form, { library_id: item.library_id, template_code: item.template_code, title: item.title, slug: item.slug, annotation: item.annotation, content: item.content, is_public: item.is_public, is_searchable: item.is_searchable, published_at: item.published_at ? new Date(item.published_at) : null, unpublished_at: item.unpublished_at ? new Date(item.unpublished_at) : null, fields: createFieldValues(loadedMetadata.templates.find((template) => template.code === item.template_code)?.fields ?? [], item.fields) })
    }
  } catch (error) { errorMessage.value = error instanceof Error ? error.message : 'Не удалось загрузить ресурс.' }
  finally { loading.value = false }
}

function changeTemplate(value: string | null): void { form.template_code = value; form.fields = createFieldValues(selectedTemplate.value?.fields ?? []); fieldErrors.value = {} }
function generateCode(): void { form.slug = generateResourceCode(form.title) }
async function submit(): Promise<void> {
  errorMessage.value = ''; fieldErrors.value = {}
  if (!form.title.trim()) { errorMessage.value = 'Заполните название.'; return }
  const fields = selectedTemplate.value?.fields ?? []
  if (unsupportedFieldTypes(fields).length) { errorMessage.value = 'Шаблон содержит неподдерживаемые поля.'; return }
  fieldErrors.value = validateFieldValues(fields, form.fields); if (Object.keys(fieldErrors.value).length) return
  const payload: LibraryItemPayload = { expected_version: creating.value ? undefined : resourceVersion.value, template_code: form.template_code, title: form.title.trim(), slug: form.slug.trim(), annotation: form.annotation, content: form.content, is_public: form.is_public, is_searchable: form.is_searchable, published_at: form.published_at?.toISOString() ?? null, unpublished_at: form.unpublished_at?.toISOString() ?? null, fields: { ...form.fields } }
  submitting.value = true
  try {
    if (creating.value) {
      const result = await adminRequest<LibraryItemDetailsResponse>(`/api/sites/${siteId.value}/resources/${originalLibraryId.value}/items`, props.accessToken, { method: 'POST', body: JSON.stringify(payload) })
      await router.replace(`/admin/sites/${siteId.value}/resources/${result.item.library_id}/items/${result.item.id}/edit`)
    } else {
      const updated = await adminRequest<LibraryItemDetailsResponse>(`/api/sites/${siteId.value}/library-items/${itemId.value}`, props.accessToken, { method: 'PATCH', body: JSON.stringify(payload) })
      resourceVersion.value = updated.item.version
      ElMessage.success('Ресурс сохранён')
    }
  } catch (error) { errorMessage.value = error instanceof AdminAPIError ? error.message : 'Не удалось сохранить ресурс.' }
  finally { submitting.value = false }
}

async function move(): Promise<void> {
  moveError.value = ''
  if (creating.value || form.library_id === ownerLibraryId.value || moving.value) return
  moving.value = true
  try {
    const moved = await adminRequest<LibraryItemDetailsResponse>(`/api/sites/${siteId.value}/library-items/${itemId.value}/move`, props.accessToken, {
      method: 'POST',
      body: JSON.stringify({ library_id: form.library_id, expected_version: resourceVersion.value }),
    })
    ownerLibraryId.value = moved.item.library_id
    form.library_id = moved.item.library_id
    resourceVersion.value = moved.item.version
    await router.replace(`/admin/sites/${siteId.value}/resources/${moved.item.library_id}/items/${moved.item.id}/edit`)
    ElMessage.success('Ресурс перемещён')
  } catch (error) {
    moveError.value = error instanceof AdminAPIError ? error.message : 'Не удалось переместить ресурс.'
  } finally {
    moving.value = false
  }
}

onMounted(() => void load())
</script>

<template>
  <section class="workspace-page">
    <header class="page-header"><div><h1>{{ creating ? 'Новый ресурс библиотеки' : (form.title || 'Ресурс библиотеки') }}</h1><p>Ресурс не входит в дерево сайта</p></div><div class="page-header-actions"><el-button v-if="!creating" :loading="moving" :disabled="loading || !canUpdate || form.library_id === ownerLibraryId" @click="move">Переместить</el-button><el-button type="primary" :loading="submitting" :disabled="loading || !canUpdate" @click="submit">Сохранить</el-button></div></header>
    <el-alert v-if="errorMessage" type="error" :closable="false" :title="errorMessage" />
    <el-alert v-if="moveError" type="error" :closable="false" :title="moveError" />
    <el-skeleton v-if="loading" :rows="10" animated />
    <el-form v-else :model="form" label-position="top" class="resource-editor-form" :class="{ 'is-readonly': !canUpdate }">
      <el-tabs v-model="activeTab" class="resource-tabs">
        <el-tab-pane label="Основное" name="main">
          <div class="resource-main-grid">
            <div class="resource-main-primary">
              <el-form-item label="Название" required><el-input v-model="form.title" :disabled="!canUpdate" /></el-form-item>
              <el-form-item label="Аннотация"><el-input v-model="form.annotation" type="textarea" :rows="7" :disabled="!canUpdate" /></el-form-item>
            </div>
            <div class="resource-main-secondary">
              <el-form-item label="Библиотека"><el-select v-model="form.library_id" class="full-width" :disabled="creating || !canUpdate"><el-option v-for="item in libraries" :key="item.id" :label="item.display_title" :value="item.id" /></el-select></el-form-item>
              <el-form-item label="Шаблон"><el-select :model-value="form.template_code" clearable class="full-width" :disabled="!canUpdate" @change="changeTemplate"><el-option v-for="item in metadata.templates" :key="item.code" :label="item.label" :value="item.code" /></el-select></el-form-item>
              <el-form-item label="Код"><el-input v-model="form.slug" :disabled="!canUpdate"><template #append><el-button :disabled="!canUpdate" @click="generateCode">Сформировать</el-button></template></el-input></el-form-item>
              <el-form-item label="Опубликован"><el-switch v-model="form.is_public" :disabled="!canUpdate" /></el-form-item>
            </div>
          </div>
          <el-form-item label="Контент" class="resource-content-field"><rich-text-editor v-model="form.content" :disabled="!canUpdate" /></el-form-item>
        </el-tab-pane>

        <el-tab-pane v-if="showWidgetsTab" label="Виджеты" name="widgets">
          <resource-widgets-editor
            v-if="selectedTemplate"
            v-model="resourceWidgets"
            :access-token="accessToken"
            :site-id="siteId"
            :resource-id="itemId!"
            :template="selectedTemplate"
            :definitions="metadata.widgets"
            :can-update="canUpdate"
            :resource-version="resourceVersion"
            @changed="load"
          />
        </el-tab-pane>

        <el-tab-pane label="Настройки" name="settings">
          <div class="resource-settings-grid">
            <el-form-item label="Начало публикации"><el-date-picker v-model="form.published_at" type="datetime" class="full-width" format="DD.MM.YYYY HH:mm" :disabled="!canUpdate" /></el-form-item>
            <el-form-item label="Окончание публикации"><el-date-picker v-model="form.unpublished_at" type="datetime" class="full-width" format="DD.MM.YYYY HH:mm" :disabled="!canUpdate" /></el-form-item>
            <el-form-item label="Доступен для поиска"><el-switch v-model="form.is_searchable" :disabled="!canUpdate" /></el-form-item>
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="showFieldsTab" label="Параметры полей" name="fields">
          <dynamic-fields-form v-model="form.fields" :fields="selectedTemplate!.fields" :errors="fieldErrors" />
        </el-tab-pane>

        <el-tab-pane v-if="!creating && canReadHistory" label="История" name="history">
          <resource-history-tab :access-token="accessToken" :site-id="siteId" :resource-id="itemId!" :resource-version="resourceVersion" :can-restore="canUpdate" :can-delete="canDeleteHistory" @changed="load" />
        </el-tab-pane>

        <el-tab-pane v-for="extension in (!creating ? applicableExtensions : [])" :key="extension.code" :label="extension.title" :name="`extension:${extension.code}`">
          <resource-extension-editor :metadata="extension" :site-id="siteId" :resource-id="itemId!" :access-token="accessToken" :can-update="canUpdate" />
        </el-tab-pane>
      </el-tabs>
    </el-form>
  </section>
</template>
