<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElAlert, ElButton, ElCard, ElDescriptions, ElDescriptionsItem, ElTable, ElTableColumn, ElTag } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { AdminAPIError } from '../../api/admin-api'
import AccessDeniedView from '../../components/AccessDeniedView.vue'
import { useSelectedSite } from '../../composables/use-selected-site'
import { getMailMessage } from './api'
import MailHtmlPreview from './MailHtmlPreview.vue'
import type { MailAddress, MailMessageDetailResponse, MailMessageStatus } from './types'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const route = useRoute()
const router = useRouter()
const selected = useSelectedSite()
const detail = ref<MailMessageDetailResponse | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const messageID = computed(() => Number(route.params.messageId ?? 0))
const statusLabels: Record<MailMessageStatus, string> = { queued: 'В очереди', sending: 'Отправляется', retryable: 'Ожидает повтора', accepted: 'Принято транспортом', failed: 'Ошибка' }

async function load(): Promise<void> {
  const siteID = selected.selectedSite.value?.id
  detail.value = null; error.value = null
  if (!siteID || !Number.isInteger(messageID.value) || messageID.value <= 0) return
  loading.value = true
  try { detail.value = await getMailMessage(props.accessToken, siteID, messageID.value) }
  catch (caught) { handleError(caught) }
  finally { loading.value = false }
}
function addresses(items: MailAddress[]): string { return items.map((item) => item.name ? `${item.name} <${item.email}>` : item.email).join(', ') || '—' }
function date(value?: string | null): string { return value ? new Date(value).toLocaleString() : '—' }
function origin(): string {
  const item = detail.value?.message
  if (!item) return '—'
  if (item.origin === 'manual') return item.requested_by_name || (item.requested_by ? `Пользователь #${item.requested_by}` : 'Ручная отправка')
  return [item.origin_source || 'Система', item.origin_event, item.origin_reference].filter(Boolean).join(' · ')
}
function handleError(caught: unknown): void {
  if (caught instanceof AdminAPIError && caught.status === 401) { emit('unauthorized'); return }
  error.value = caught instanceof Error ? caught.message : 'Не удалось загрузить письмо.'
}
watch(() => [selected.selectedSite.value?.id, route.params.messageId], () => void load())
onMounted(() => void load())
</script>

<template>
  <access-denied-view v-if="!permissions.has('mail.message.read')" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page mail-detail-page" v-loading="loading">
    <header class="page-header"><div><h1>Письмо #{{ messageID }}</h1><p>Снимок сообщения и история доставки</p></div><el-button @click="router.push({ name: 'mail.history' })">К истории</el-button></header>
    <el-alert v-if="!selected.selectedSite.value" type="warning" :closable="false" title="Выберите сайт в боковой панели." />
    <el-alert v-if="error" type="error" :closable="false" :title="error" show-icon />
    <template v-if="detail">
      <el-card shadow="never" header="Жизненный цикл">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Статус"><el-tag>{{ statusLabels[detail.message.status] }}</el-tag></el-descriptions-item>
          <el-descriptions-item label="Запрошено">{{ date(detail.message.requested_at) }}</el-descriptions-item>
          <el-descriptions-item label="Принято транспортом">{{ date(detail.message.accepted_at) }}</el-descriptions-item>
          <el-descriptions-item label="Источник">{{ origin() }}</el-descriptions-item>
          <el-descriptions-item label="Шаблон">{{ detail.message.template_name }} ({{ detail.message.template_code }})</el-descriptions-item>
          <el-descriptions-item label="Message-ID">{{ detail.message.rfc_message_id }}</el-descriptions-item>
          <el-descriptions-item label="Попыток">{{ detail.message.attempt_count }}</el-descriptions-item>
        </el-descriptions>
      </el-card>
      <el-card shadow="never" header="Неизменяемое содержимое">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="От">{{ addresses([detail.message.from]) }}</el-descriptions-item>
          <el-descriptions-item label="Кому">{{ addresses(detail.message.to) }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.message.cc.length" label="CC">{{ addresses(detail.message.cc) }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.message.bcc.length" label="BCC">{{ addresses(detail.message.bcc) }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.message.reply_to" label="Reply-To">{{ addresses([detail.message.reply_to]) }}</el-descriptions-item>
          <el-descriptions-item label="Тема">{{ detail.message.subject || '—' }}</el-descriptions-item>
        </el-descriptions>
        <pre v-if="detail.message.content_type === 'text'" class="mail-text-preview">{{ detail.message.text_body }}</pre>
        <mail-html-preview v-else :html="detail.message.html_body" title="HTML-содержимое письма" />
        <el-table :data="detail.message.attachments" empty-text="Вложений нет">
          <el-table-column prop="filename" label="Файл" min-width="220" /><el-table-column prop="mime_type" label="MIME" min-width="180" /><el-table-column prop="size" label="Размер, байт" width="140" /><el-table-column prop="checksum_sha256" label="SHA-256" min-width="260" />
        </el-table>
      </el-card>
      <el-card shadow="never" header="Попытки доставки">
        <el-alert type="info" :closable="false" title="Статус «Принято транспортом» означает принятие SMTP/транспортом, а не гарантированное попадание в почтовый ящик." />
        <el-table :data="detail.attempts" empty-text="Попыток пока нет">
          <el-table-column prop="attempt_number" label="#" width="70" />
          <el-table-column label="Начало" width="180"><template #default="{ row }">{{ date(row.started_at) }}</template></el-table-column>
          <el-table-column label="Завершение" width="180"><template #default="{ row }">{{ date(row.finished_at) }}</template></el-table-column>
          <el-table-column prop="status" label="Статус" width="130" /><el-table-column prop="driver" label="Драйвер" width="120" /><el-table-column prop="response_code" label="Ответ" width="120" /><el-table-column prop="remote_message_id" label="Provider ID" min-width="180" /><el-table-column prop="safe_error" label="Безопасная ошибка" min-width="260" />
        </el-table>
      </el-card>
    </template>
  </section>
</template>

<style scoped>.mail-detail-page{display:grid;gap:16px}.mail-detail-page :deep(.el-card__body){display:grid;gap:16px}.mail-text-preview{white-space:pre-wrap;padding:16px;border:1px solid var(--el-border-color);border-radius:8px;background:var(--el-fill-color-lighter);font:inherit}</style>
