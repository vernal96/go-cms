<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElAlert, ElButton, ElCard, ElCheckbox, ElForm, ElFormItem, ElInput, ElMessage, ElOption, ElSelect, ElSwitch, ElTag } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import RichTextEditor from '../../components/RichTextEditor.vue'
import AccessDeniedView from '../../components/AccessDeniedView.vue'
import { useSelectedSite } from '../../composables/use-selected-site'
import type { FieldDefinition } from '../../types/admin'
import { createMailTemplate, getMailTemplate, listMailSiteVariables, updateMailTemplate } from './api'
import MailAddressFields from './MailAddressFields.vue'
import MailAddressListEditor from './MailAddressListEditor.vue'
import MailAttachmentsEditor from './MailAttachmentsEditor.vue'
import MailVariablesEditor from './MailVariablesEditor.vue'
import type { MailSiteVariable, MailTemplate, MailTemplatePayload } from './types'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const route = useRoute()
const router = useRouter()
const selected = useSelectedSite()
const loading = ref(false)
const saving = ref(false)
const error = ref<string | null>(null)
const replyToEnabled = ref(false)
const siteVariables = ref<MailSiteVariable[]>([])
const uploadStorage = ref('')
const uploadPath = ref('')
const selectedPlaceholder = ref('')
const form = reactive<MailTemplatePayload>(emptyTemplate())
const addressErrors = reactive({
  from: '',
  to: {} as Record<number, string>,
  cc: {} as Record<number, string>,
  bcc: {} as Record<number, string>,
  replyTo: '',
})
const templateID = computed(() => Number(route.params.templateId ?? 0))
const editing = computed(() => Number.isInteger(templateID.value) && templateID.value > 0)
const canSave = computed(() => props.permissions.has(editing.value ? 'mail.template.update' : 'mail.template.create'))
const canAccess = computed(() => editing.value ? props.permissions.has('mail.template.read') : props.permissions.has('mail.template.create'))
const sitePlaceholders = computed(() => siteVariables.value.map((item) => `{{${item.variable}}}`))
const dataPlaceholders = computed(() => form.variables.filter((item) => item.key).map((item) => `{{data.${item.key}}}`))

function emptyTemplate(): MailTemplatePayload {
  return {
    code: '', name: '', enabled: true,
    from: { name: '', email: '' }, to: [{ name: '', email: '' }], cc: [], bcc: [], reply_to: null,
    subject: '', content_type: 'text', text_body: '', html_body: '', attachments: [], variables: [],
  }
}

function assignTemplate(item: MailTemplate): void {
  Object.assign(form, {
    code: item.code, name: item.name, enabled: item.enabled,
    from: { ...item.from }, to: item.to.map((value) => ({ ...value })), cc: item.cc.map((value) => ({ ...value })),
    bcc: item.bcc.map((value) => ({ ...value })), reply_to: item.reply_to ? { ...item.reply_to } : null,
    subject: item.subject, content_type: item.content_type, text_body: item.text_body, html_body: item.html_body,
    attachments: item.attachments.map((value) => ({ ...value })), variables: item.variables.map(cloneField),
  })
  replyToEnabled.value = item.reply_to != null
}

function cloneField(value: FieldDefinition): FieldDefinition {
  return { ...value, required: value.required, rules: [...value.rules], options: value.options ? { ...value.options, choices: value.options.choices?.map((item) => ({ ...item })), storages: [...(value.options.storages ?? [])], mime_types: [...(value.options.mime_types ?? [])] } : undefined }
}

async function load(): Promise<void> {
  error.value = null
  siteVariables.value = []
  selectedPlaceholder.value = ''
  if (!canAccess.value) return
  const siteID = selected.selectedSite.value?.id
  if (!siteID) return
  loading.value = true
  try {
    const editor = await listMailSiteVariables(props.accessToken, siteID)
    siteVariables.value = editor.items
    uploadStorage.value = editor.upload_storage
    uploadPath.value = editor.upload_path
    if (editing.value) assignTemplate(await getMailTemplate(props.accessToken, siteID, templateID.value))
    else assignTemplate({ ...emptyTemplate(), id: 0, site_id: siteID, created_at: '', updated_at: '' })
  } catch (caught) {
    handleError(caught)
  } finally { loading.value = false }
}

function validate(): string | null {
  clearAddressErrors()
  if (!/^[a-z][a-z0-9_]{1,63}$/.test(form.code)) return 'Код должен содержать 2–64 строчных латинских символа, цифры или подчёркивания.'
  if (!form.name.trim()) return 'Укажите название шаблона.'
  if (!form.from.email.trim()) {
    addressErrors.from = 'Email отправителя обязателен.'
    return addressErrors.from
  }
  let recipientCount = 0
  for (const group of ['to', 'cc', 'bcc'] as const) {
    form[group].forEach((address, index) => {
      const used = Boolean(address.name.trim() || address.email.trim())
      if (!used) return
      if (!address.email.trim()) addressErrors[group][index] = 'Email обязателен для заполненной строки.'
      else recipientCount += 1
    })
  }
  if (Object.keys(addressErrors.to).length || Object.keys(addressErrors.cc).length || Object.keys(addressErrors.bcc).length) return 'Заполните email во всех используемых строках адресов.'
  if (recipientCount === 0) return 'Добавьте хотя бы один адрес получателя.'
  if (replyToEnabled.value && !form.reply_to?.email.trim()) {
    addressErrors.replyTo = 'Email Reply-To обязателен.'
    return addressErrors.replyTo
  }
  const keys = new Set<string>()
  for (const variable of form.variables) {
    if (!/^[a-z][a-z0-9_]*$/.test(variable.key) || !variable.label.trim()) return 'У каждой переменной должны быть корректный ключ и название.'
    if (keys.has(variable.key)) return `Переменная data.${variable.key} объявлена повторно.`
    keys.add(variable.key)
  }
  for (const attachment of form.attachments) {
    if (attachment.source === 'static' && !attachment.file_id) return 'Выберите файл для каждого статического вложения.'
    if (attachment.source === 'variable' && !attachment.variable) return 'Выберите файловую переменную для каждого переменного вложения.'
    if (attachment.source === 'site' && !attachment.variable) return 'Выберите файловое поле сайта для каждого вложения из настроек сайта.'
  }
  return null
}

function clearAddressErrors(): void {
  addressErrors.from = ''
  addressErrors.replyTo = ''
  addressErrors.to = {}
  addressErrors.cc = {}
  addressErrors.bcc = {}
}

async function save(): Promise<void> {
  const siteID = selected.selectedSite.value?.id
  if (!siteID || !canSave.value || saving.value) return
  const validation = validate()
  if (validation) { error.value = validation; return }
  saving.value = true
  error.value = null
  const payload: MailTemplatePayload = JSON.parse(JSON.stringify({ ...form, reply_to: replyToEnabled.value ? form.reply_to : null }))
  try {
    if (editing.value) await updateMailTemplate(props.accessToken, siteID, templateID.value, payload)
    else await createMailTemplate(props.accessToken, siteID, payload)
    ElMessage.success(editing.value ? 'Шаблон сохранён' : 'Шаблон создан')
    await router.push({ name: 'mail.templates' })
  } catch (caught) { handleError(caught) }
  finally { saving.value = false }
}

function toggleReplyTo(value: boolean): void {
  replyToEnabled.value = value
  if (value && !form.reply_to) form.reply_to = { name: '', email: '' }
}

async function copyPlaceholder(value: string): Promise<void> {
  selectedPlaceholder.value = value
  try { await navigator.clipboard.writeText(value); ElMessage.success(`Скопировано: ${value}`) }
  catch { ElMessage.warning('Не удалось скопировать переменную.') }
}

function insertPlaceholder(target: 'subject' | 'body'): void {
  const value = selectedPlaceholder.value
  if (!value) return
  if (target === 'subject') form.subject += value
  else if (form.content_type === 'html') form.html_body += value
  else form.text_body += value
}

function handleError(caught: unknown): void {
  if (caught && typeof caught === 'object' && 'status' in caught && caught.status === 401) { emit('unauthorized'); return }
  error.value = caught instanceof Error ? caught.message : 'Операция с шаблоном не выполнена.'
}

watch(() => [selected.selectedSite.value?.id, route.params.templateId], () => void load())
onMounted(() => void load())
</script>

<template>
  <access-denied-view v-if="!canAccess" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page mail-template-form" v-loading="loading">
    <header class="page-header">
      <div><h1>{{ editing ? 'Редактирование шаблона' : 'Новый шаблон' }}</h1><p>Адреса, содержимое, переменные и вложения</p></div>
      <div class="page-actions">
        <el-button @click="router.push({ name: 'mail.templates' })">Отмена</el-button>
        <el-button type="primary" :loading="saving" :disabled="!canSave" @click="save">Сохранить</el-button>
      </div>
    </header>
    <el-alert v-if="!selected.selectedSite.value" type="warning" :closable="false" title="Выберите сайт в боковой панели." />
    <el-alert v-if="error" type="error" :closable="false" :title="error" />

    <el-form v-if="selected.selectedSite.value" label-position="top">
      <el-card shadow="never" header="Основное">
        <div class="mail-form-grid">
          <el-form-item label="Код"><el-input v-model="form.code" placeholder="feedback_notification" /></el-form-item>
          <el-form-item label="Название"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="Включён"><el-switch v-model="form.enabled" /></el-form-item>
        </div>
      </el-card>

      <el-card shadow="never" header="Переменные">
        <p class="mail-help">Для каждого значения выберите обязательность. Типы, правила и обязательные значения проверяются при предпросмотре и отправке.</p>
        <mail-variables-editor v-model="form.variables" />
        <div v-if="sitePlaceholders.length" class="mail-placeholder-list">
          <strong>Сайт</strong>
          <el-tag v-for="placeholder in sitePlaceholders" :key="placeholder" class="mail-placeholder" @click="copyPlaceholder(placeholder)">{{ placeholder }}</el-tag>
        </div>
        <div v-if="dataPlaceholders.length" class="mail-placeholder-list">
          <strong>Данные шаблона</strong>
          <el-tag v-for="placeholder in dataPlaceholders" :key="placeholder" class="mail-placeholder" @click="copyPlaceholder(placeholder)">{{ placeholder }}</el-tag>
        </div>
        <div v-if="selectedPlaceholder" class="mail-placeholder-actions">
          <span>Выбрано: <code>{{ selectedPlaceholder }}</code></span>
          <el-button size="small" @click="insertPlaceholder('subject')">Вставить в тему</el-button>
          <el-button size="small" @click="insertPlaceholder('body')">Вставить в тело</el-button>
        </div>
      </el-card>

      <el-card shadow="never" header="Адреса">
        <el-form-item label="От"><mail-address-fields v-model="form.from" sender email-required :error="addressErrors.from" /></el-form-item>
        <el-form-item label="Кому"><mail-address-list-editor v-model="form.to" :errors="addressErrors.to" /></el-form-item>
        <el-form-item label="Копия (CC)"><mail-address-list-editor v-model="form.cc" :errors="addressErrors.cc" /></el-form-item>
        <el-form-item label="Скрытая копия (BCC)"><mail-address-list-editor v-model="form.bcc" :errors="addressErrors.bcc" /></el-form-item>
        <el-checkbox :model-value="replyToEnabled" @update:model-value="toggleReplyTo(Boolean($event))">Добавить Reply-To</el-checkbox>
        <el-form-item v-if="replyToEnabled && form.reply_to" label="Reply-To"><mail-address-fields v-model="form.reply_to" email-required :error="addressErrors.replyTo" /></el-form-item>
      </el-card>

      <el-card shadow="never" header="Содержимое">
        <el-form-item label="Тема"><el-input v-model="form.subject" /></el-form-item>
        <el-form-item label="Тип содержимого">
          <el-select v-model="form.content_type"><el-option value="text" label="Обычный текст" /><el-option value="html" label="HTML" /></el-select>
        </el-form-item>
        <el-form-item v-if="form.content_type === 'text'" label="Текст письма"><el-input v-model="form.text_body" type="textarea" :rows="12" /></el-form-item>
        <el-form-item v-else label="HTML письма"><rich-text-editor v-model="form.html_body" /></el-form-item>
      </el-card>

      <el-card shadow="never" header="Вложения">
        <mail-attachments-editor v-model="form.attachments" :variables="form.variables" :site-variables="siteVariables" :access-token="accessToken" :permissions="permissions" :upload-storage="uploadStorage" :upload-path="uploadPath" />
      </el-card>
    </el-form>
  </section>
</template>

<style scoped>
.mail-template-form { display: grid; gap: 16px; }
.mail-template-form :deep(.el-form) { display: grid; gap: 16px; }
.mail-form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0 16px; }
.mail-help { margin: 0 0 12px; color: var(--el-text-color-secondary); }
.mail-placeholder-list { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin-top: 14px; }
.mail-placeholder-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin-top: 12px; }
.mail-placeholder { cursor: pointer; font-family: ui-monospace, monospace; }
.page-actions { display: flex; gap: 8px; }
@media (max-width: 760px) { .mail-form-grid { grid-template-columns: 1fr; } }
</style>
