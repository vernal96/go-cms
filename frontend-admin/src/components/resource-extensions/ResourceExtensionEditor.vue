<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  ElAlert,
  ElButton,
  ElCard,
  ElDescriptions,
  ElDescriptionsItem,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElSkeleton,
  ElSwitch,
  ElTabPane,
  ElTabs,
  ElTag,
} from 'element-plus'

import { AdminAPIError, adminRequest } from '../../api/admin-api'
import type {
  ResourceExtensionMetadata,
  SEOPreview,
} from '../../types/admin'

const props = defineProps<{
  metadata: ResourceExtensionMetadata
  siteId: number
  resourceId: number
  accessToken: string
  canUpdate: boolean
}>()
const emit = defineEmits<{ unauthorized: [] }>()

const loading = ref(true)
const saving = ref(false)
const previewing = ref(false)
const loadError = ref<string | null>(null)
const actionError = ref<string | null>(null)
const fieldErrors = ref<Record<string, string>>({})
const preview = ref<Record<string, unknown> | null>(null)
const activeSEOGroup = ref('general')
type ExtensionValue = string | number | boolean | null | undefined

const values = reactive<Record<string, ExtensionValue>>({})

const endpoint = computed(() =>
  `/api/admin/sites/${props.siteId}/resources/${props.resourceId}/extensions/${props.metadata.code}`,
)
const seoPreview = computed(() =>
  props.metadata.code === 'seo' ? preview.value as SEOPreview | null : null,
)
const isSEO = computed(() => props.metadata.code === 'seo')
const seoFieldGroups = computed(() => [
  { name: 'general', label: 'Основные', fields: props.metadata.fields.filter((field) => !field.key.startsWith('og_')) },
  { name: 'opengraph', label: 'OpenGraph', fields: props.metadata.fields.filter((field) => field.key.startsWith('og_')) },
].filter((group) => group.fields.length > 0))

async function load(): Promise<void> {
  loading.value = true
  loadError.value = null
  try {
    const loaded = await adminRequest<Record<string, unknown>>(endpoint.value, props.accessToken)
    replaceValues(loaded)
  } catch (error) {
    handleError(error, 'Не удалось загрузить данные расширения.', false)
  } finally {
    loading.value = false
  }
}

async function save(): Promise<void> {
  if (!props.canUpdate || saving.value) return
  saving.value = true
  clearActionErrors()
  try {
    const saved = await adminRequest<Record<string, unknown>>(endpoint.value, props.accessToken, {
      method: 'PATCH',
      body: JSON.stringify(values),
    })
    replaceValues(saved)
    ElMessage.success(`${props.metadata.title}: настройки сохранены`)
  } catch (error) {
    handleError(error, 'Не удалось сохранить настройки расширения.', true)
  } finally {
    saving.value = false
  }
}

async function buildPreview(): Promise<void> {
  if (previewing.value) return
  previewing.value = true
  clearActionErrors()
  try {
    preview.value = await adminRequest<Record<string, unknown>>(
      `${endpoint.value}/preview`,
      props.accessToken,
      { method: 'POST', body: JSON.stringify(values) },
    )
  } catch (error) {
    handleError(error, 'Не удалось построить предпросмотр.', true)
  } finally {
    previewing.value = false
  }
}

function replaceValues(next: Record<string, unknown>): void {
  for (const key of Object.keys(values)) delete values[key]
  for (const [key, value] of Object.entries(next)) {
    values[key] = typeof value === 'string' ||
      typeof value === 'number' ||
      typeof value === 'boolean' ||
      value === null ||
      value === undefined
      ? value
      : ''
  }
}

function clearActionErrors(): void {
  actionError.value = null
  fieldErrors.value = {}
}

function textValue(key: string): string | number {
  const value = values[key]
  return typeof value === 'string' || typeof value === 'number' ? value : ''
}

function updateText(key: string, value: string | number): void {
  values[key] = value
}

function updateSwitch(key: string, value: boolean): void {
  values[key] = value
}

async function copyVariable(code: string): Promise<void> {
  try {
    if (navigator.clipboard?.writeText) await navigator.clipboard.writeText(code)
    else {
      const input = document.createElement('textarea')
      input.value = code
      input.setAttribute('readonly', '')
      input.style.position = 'fixed'
      input.style.opacity = '0'
      document.body.appendChild(input)
      input.select()
      const copied = document.execCommand('copy')
      input.remove()
      if (!copied) throw new Error('Clipboard API is unavailable')
    }
    ElMessage.success('Переменная скопирована')
  } catch {
    ElMessage.error('Не удалось скопировать переменную')
  }
}

function handleError(error: unknown, fallback: string, action: boolean): void {
  if (error instanceof AdminAPIError && error.status === 401) {
    emit('unauthorized')
    return
  }
  if (error instanceof AdminAPIError) {
    fieldErrors.value = Object.fromEntries(
      error.fieldErrors.map((field) => [field.key, field.param || error.message]),
    )
  }
  const message = error instanceof Error ? error.message : fallback
  if (action) actionError.value = message
  else loadError.value = message
}

onMounted(() => void load())
</script>

<template>
  <div class="resource-extension-editor">
    <el-alert v-if="loadError" type="error" :closable="false" :title="loadError" show-icon />
    <el-skeleton v-else-if="loading" :rows="8" animated />
    <template v-else>
      <el-alert v-if="!canUpdate" type="info" :closable="false" title="Настройки доступны только для чтения" show-icon />
      <el-alert v-if="actionError" type="error" :closable="false" :title="actionError" show-icon />

      <el-form label-position="top">
        <el-tabs v-if="isSEO" v-model="activeSEOGroup">
          <el-tab-pane v-for="group in seoFieldGroups" :key="group.name" :label="group.label" :name="group.name">
            <div class="extension-fields">
              <el-form-item
                v-for="field in group.fields"
                :key="field.key"
                :label="field.label"
                :error="fieldErrors[field.key]"
              >
                <el-switch
                  v-if="field.control === 'switch'"
                  :model-value="values[field.key] === true"
                  :disabled="!canUpdate"
                  @update:model-value="updateSwitch(field.key, $event === true)"
                />
                <el-input
                  v-else
                  :model-value="textValue(field.key)"
                  :type="field.control === 'textarea' ? 'textarea' : 'text'"
                  :rows="field.rows ?? 3"
                  :disabled="!canUpdate"
                  @update:model-value="updateText(field.key, $event)"
                />
              </el-form-item>
            </div>
          </el-tab-pane>
        </el-tabs>
        <div v-else class="extension-fields">
          <el-form-item
            v-for="field in metadata.fields"
            :key="field.key"
            :label="field.label"
            :error="fieldErrors[field.key]"
          >
            <el-switch
              v-if="field.control === 'switch'"
              :model-value="values[field.key] === true"
              :disabled="!canUpdate"
              @update:model-value="updateSwitch(field.key, $event === true)"
            />
            <el-input
              v-else
              :model-value="textValue(field.key)"
              :type="field.control === 'textarea' ? 'textarea' : 'text'"
              :rows="field.rows ?? 3"
              :disabled="!canUpdate"
              @update:model-value="updateText(field.key, $event)"
            />
          </el-form-item>
        </div>
      </el-form>

      <el-card v-if="metadata.variables.length" shadow="never" class="extension-variables">
        <template #header>Доступные переменные</template>
        <div class="variable-list">
          <el-tag
            v-for="variable in metadata.variables"
            :key="variable.code"
            effect="plain"
            class="extension-variable"
            role="button"
            tabindex="0"
            @click="copyVariable(variable.code)"
            @keydown.enter="copyVariable(variable.code)"
          >
            {{ variable.code }}
          </el-tag>
        </div>
      </el-card>

      <div class="extension-actions">
        <el-button :loading="previewing" @click="buildPreview">Предпросмотр</el-button>
        <el-button type="primary" :loading="saving" :disabled="!canUpdate" @click="save">
          Сохранить {{ metadata.title }}
        </el-button>
      </div>

      <el-card v-if="seoPreview" shadow="never" class="seo-preview">
        <template #header>Предпросмотр SEO</template>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="Title">
            {{ seoPreview.title }} ({{ seoPreview.title_characters }})
          </el-descriptions-item>
          <el-descriptions-item label="Description">
            {{ seoPreview.description }} ({{ seoPreview.description_characters }})
          </el-descriptions-item>
          <el-descriptions-item label="Canonical">{{ seoPreview.canonical_url || '—' }}</el-descriptions-item>
          <el-descriptions-item label="Robots">
            {{ seoPreview.robots.index ? 'index' : 'noindex' }},
            {{ seoPreview.robots.follow ? 'follow' : 'nofollow' }}
          </el-descriptions-item>
          <el-descriptions-item label="Open Graph title">{{ seoPreview.open_graph.title }}</el-descriptions-item>
          <el-descriptions-item label="Open Graph description">{{ seoPreview.open_graph.description }}</el-descriptions-item>
        </el-descriptions>
        <el-alert
          v-for="warning in seoPreview.warnings"
          :key="`${warning.field}:${warning.variable}`"
          class="preview-warning"
          type="warning"
          :closable="false"
          :title="`${warning.variable}: значение отсутствует`"
          show-icon
        />
      </el-card>
      <pre v-else-if="preview" class="extension-preview-json">{{ JSON.stringify(preview, null, 2) }}</pre>
    </template>
  </div>
</template>

<style scoped>
.resource-extension-editor,
.extension-fields {
  display: grid;
  gap: 16px;
}

.extension-fields {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.variable-list,
.extension-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.extension-variable {
  cursor: pointer;
  transition: background-color .18s ease, border-color .18s ease, color .18s ease, box-shadow .18s ease;
}

.extension-variable:hover,
.extension-variable:focus-visible {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--el-color-primary) 12%, transparent);
}

.extension-actions {
  justify-content: flex-end;
}

.preview-warning {
  margin-top: 12px;
}

.extension-preview-json {
  overflow: auto;
  padding: 16px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
}

@media (max-width: 900px) {
  .extension-fields {
    grid-template-columns: 1fr;
  }
}
</style>
