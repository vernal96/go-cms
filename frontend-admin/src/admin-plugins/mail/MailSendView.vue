<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElAlert, ElButton, ElCard, ElDescriptions, ElDescriptionsItem, ElForm, ElFormItem, ElMessage, ElOption, ElSelect, ElTag } from 'element-plus'
import { AdminAPIError } from '../../api/admin-api'
import { useRouter } from 'vue-router'
import AccessDeniedView from '../../components/AccessDeniedView.vue'
import DynamicFieldsForm from '../../components/fields/DynamicFieldsForm.vue'
import { createFieldValues, validateFieldValues, type DynamicFieldErrors, type DynamicValues } from '../../components/fields/model'
import { useSelectedSite } from '../../composables/use-selected-site'
import { listSendTemplates, previewMail, queueMail } from './api'
import MailHtmlPreview from './MailHtmlPreview.vue'
import type { MailAddress, MailTemplate, RenderedMailMessage } from './types'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const selected = useSelectedSite()
const router = useRouter()
const templates = ref<MailTemplate[]>([])
const templateID = ref<number | null>(null)
const values = ref<DynamicValues>({})
const fieldErrors = ref<DynamicFieldErrors>({})
const preview = ref<RenderedMailMessage | null>(null)
const loading = ref(false)
const previewing = ref(false)
const sending = ref(false)
const error = ref<string | null>(null)
const queuedID = ref<number | null>(null)
const template = computed(() => templates.value.find((item) => item.id === templateID.value) ?? null)

async function load(): Promise<void> {
  const siteID = selected.selectedSite.value?.id
  templates.value = []; choose(null); error.value = null
  if (!siteID || !props.permissions.has('mail.message.create')) return
  loading.value = true
  try { templates.value = (await listSendTemplates(props.accessToken, siteID)).items }
  catch (caught) { handleError(caught) }
  finally { loading.value = false }
}

function choose(id: number | null): void {
  templateID.value = id
  const selectedTemplate = templates.value.find((item) => item.id === id)
  values.value = createFieldValues(selectedTemplate?.variables ?? [])
  fieldErrors.value = {}
  preview.value = null
  queuedID.value = null
  error.value = null
}

function validate(): boolean {
  if (!template.value) { error.value = 'Выберите шаблон.'; return false }
  fieldErrors.value = validateFieldValues(template.value.variables, values.value)
  return Object.keys(fieldErrors.value).length === 0
}

async function renderPreview(): Promise<void> {
  const siteID = selected.selectedSite.value?.id
  if (!siteID || !templateID.value || previewing.value || !validate()) return
  previewing.value = true; error.value = null; preview.value = null
  try { preview.value = await previewMail(props.accessToken, siteID, templateID.value, values.value) }
  catch (caught) { handleError(caught) }
  finally { previewing.value = false }
}

async function send(): Promise<void> {
  const siteID = selected.selectedSite.value?.id
  if (!siteID || !templateID.value || sending.value || !preview.value || !validate()) return
  sending.value = true; error.value = null
  try {
    const queued = await queueMail(props.accessToken, siteID, templateID.value, values.value)
    queuedID.value = queued.id
    ElMessage.success(`Письмо #${queued.id} поставлено в очередь. Доставка выполняется асинхронно.`)
    preview.value = null
  } catch (caught) { handleError(caught) }
  finally { sending.value = false }
}

function addresses(items: MailAddress[]): string { return items.map((item) => item.name ? `${item.name} <${item.email}>` : item.email).join(', ') || '—' }
function handleError(caught: unknown): void {
  if (caught instanceof AdminAPIError && caught.status === 401) { emit('unauthorized'); return }
  error.value = caught instanceof Error ? caught.message : 'Операция с письмом не выполнена.'
}

watch(() => selected.selectedSite.value?.id, () => void load())
onMounted(() => void load())
</script>

<template>
  <access-denied-view v-if="!permissions.has('mail.message.create')" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page mail-send-page" v-loading="loading">
    <header class="page-header"><div><h1>Отправить письмо</h1><p>Предпросмотр обязателен; доставка после постановки в очередь выполняется асинхронно</p></div></header>
    <el-alert v-if="!selected.selectedSite.value" type="warning" :closable="false" title="Выберите сайт в боковой панели." />
    <el-alert v-if="error" type="error" :closable="false" :title="error" show-icon />
    <el-alert v-if="queuedID" type="success" :closable="false" :title="`Письмо #${queuedID} поставлено в очередь.`" show-icon>
      <el-button text type="primary" @click="router.push({ name: 'mail.history.detail', params: { messageId: queuedID } })">Открыть историю доставки</el-button>
    </el-alert>
    <el-card v-if="selected.selectedSite.value" shadow="never" header="1. Шаблон и данные">
      <el-form label-position="top">
        <el-form-item label="Шаблон">
          <el-select :model-value="templateID" filterable placeholder="Выберите включённый шаблон" @update:model-value="choose($event)">
            <el-option v-for="item in templates" :key="item.id" :value="item.id" :label="`${item.name} (${item.code})`" />
          </el-select>
        </el-form-item>
        <dynamic-fields-form v-if="template" :fields="template.variables" :model-value="values" :errors="fieldErrors" :site-id="selected.selectedSite.value.id" :access-token="accessToken" @update:model-value="values = $event; preview = null" />
      </el-form>
      <el-alert v-if="template && template.variables.length === 0" type="info" :closable="false" title="У этого шаблона нет переменных." />
      <el-button type="primary" :loading="previewing" :disabled="!template || sending" @click="renderPreview">Предпросмотр</el-button>
    </el-card>
    <el-card v-if="preview" shadow="never" header="2. Проверьте итоговое письмо">
      <el-alert v-if="preview.warnings.length" class="mail-warnings" type="warning" :closable="false" show-icon>
        <template #title>Не заполнены необязательные значения</template>
        <ul><li v-for="warning in preview.warnings" :key="`${warning.field}:${warning.variable}`"><code>{{ warning.variable }}</code> — {{ warning.message }}</li></ul>
      </el-alert>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="От">{{ addresses([preview.from]) }}</el-descriptions-item>
        <el-descriptions-item label="Кому">{{ addresses(preview.to) }}</el-descriptions-item>
        <el-descriptions-item v-if="preview.cc.length" label="CC">{{ addresses(preview.cc) }}</el-descriptions-item>
        <el-descriptions-item v-if="preview.bcc.length" label="BCC">{{ addresses(preview.bcc) }}</el-descriptions-item>
        <el-descriptions-item v-if="preview.reply_to" label="Reply-To">{{ addresses([preview.reply_to]) }}</el-descriptions-item>
        <el-descriptions-item label="Тема">{{ preview.subject || '—' }}</el-descriptions-item>
      </el-descriptions>
      <pre v-if="preview.content_type === 'text'" class="mail-text-preview">{{ preview.text_body }}</pre>
      <mail-html-preview v-else :html="preview.html_body" />
      <div class="mail-attachment-list"><strong>Вложения</strong><span v-if="!preview.attachments.length">Нет</span><el-tag v-for="item in preview.attachments" :key="`${item.source}:${item.file_id ?? 0}:${item.filename}`">{{ item.filename }} · {{ item.mime_type }} · {{ item.size }} байт</el-tag></div>
      <el-button type="success" :loading="sending" :disabled="previewing" @click="send">Поставить в очередь</el-button>
    </el-card>
  </section>
</template>

<style scoped>
.mail-send-page{display:grid;gap:16px}.mail-send-page :deep(.el-select){width:100%}.mail-send-page :deep(.el-card__body){display:grid;gap:16px}.mail-warnings ul{margin:8px 0 0}.mail-text-preview{white-space:pre-wrap;padding:16px;border:1px solid var(--el-border-color);border-radius:8px;background:var(--el-fill-color-lighter);font:inherit}.mail-attachment-list{display:flex;flex-wrap:wrap;gap:8px;align-items:center}
</style>
