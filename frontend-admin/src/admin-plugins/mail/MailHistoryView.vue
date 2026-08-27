<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { ElAlert, ElButton, ElDatePicker, ElInput, ElMessage, ElMessageBox, ElOption, ElPagination, ElSelect, ElTable, ElTableColumn, ElTag } from 'element-plus'
import { useRouter } from 'vue-router'
import { AdminAPIError } from '../../api/admin-api'
import AccessDeniedView from '../../components/AccessDeniedView.vue'
import { useSelectedSite } from '../../composables/use-selected-site'
import { deleteMailMessage, listMailMessages } from './api'
import type { MailMessageFilters, MailMessageStatus, MailMessageSummary } from './types'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const router = useRouter()
const selected = useSelectedSite()
const items = ref<MailMessageSummary[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const page = ref(1)
const perPage = 20
const total = ref(0)
const filters = reactive<{ status: MailMessageStatus | ''; template_code: string; recipient: string; date_from: Date | null; date_to: Date | null }>({ status: '', template_code: '', recipient: '', date_from: null, date_to: null })

const statusLabels: Record<MailMessageStatus, string> = { queued: 'В очереди', sending: 'Отправляется', retryable: 'Ожидает повтора', accepted: 'Принято транспортом', failed: 'Ошибка' }
const statusTypes: Record<MailMessageStatus, 'info' | 'warning' | 'success' | 'danger'> = { queued: 'info', sending: 'warning', retryable: 'warning', accepted: 'success', failed: 'danger' }

async function load(): Promise<void> {
  const siteID = selected.selectedSite.value?.id
  if (!siteID || !props.permissions.has('mail.message.read')) { items.value = []; total.value = 0; return }
  loading.value = true; error.value = null
  try {
    const query: MailMessageFilters = { status: filters.status, template_code: filters.template_code.trim(), recipient: filters.recipient.trim(), date_from: filters.date_from?.toISOString(), date_to: filters.date_to?.toISOString() }
    const response = await listMailMessages(props.accessToken, siteID, page.value, perPage, query)
    items.value = response.items; total.value = response.pagination.total
  } catch (caught) { handleError(caught) }
  finally { loading.value = false }
}

async function remove(row: unknown): Promise<void> {
  const item = row as MailMessageSummary
  const siteID = selected.selectedSite.value?.id
  if (!siteID || !props.permissions.has('mail.message.delete') || !terminal(item)) return
  try { await ElMessageBox.confirm(`Удалить письмо #${item.id} из истории?`, 'Удаление истории', { confirmButtonText: 'Удалить', cancelButtonText: 'Отмена', type: 'warning' }) }
  catch { return }
  try { await deleteMailMessage(props.accessToken, siteID, item.id); ElMessage.success('Запись истории удалена'); await load() }
  catch (caught) { handleError(caught) }
}

function terminal(row: unknown): boolean { const item = row as MailMessageSummary; return item.status === 'accepted' || item.status === 'failed' }
function recipients(row: unknown): string { return (row as MailMessageSummary).recipients.join(', ') || '—' }
function origin(row: unknown): string {
  const item = row as MailMessageSummary
  if (item.origin === 'manual') return item.requested_by_name || (item.requested_by ? `Пользователь #${item.requested_by}` : 'Ручная отправка')
  return [item.origin_source || 'Система', item.origin_event, item.origin_reference].filter(Boolean).join(' · ')
}
function latest(row: unknown): string {
  const item = row as MailMessageSummary
  const attempt = item.latest_attempt
  if (!attempt) return item.attempt_count ? `${item.attempt_count} попыток` : 'Попыток нет'
  if (attempt.status === 'failed') return `${item.attempt_count}: ${attempt.safe_error || 'ошибка транспорта'}`
  return `${item.attempt_count}: ${attempt.status === 'accepted' ? 'принято' : 'выполняется'}`
}
function statusLabel(row: unknown): string { return statusLabels[(row as MailMessageSummary).status] }
function statusType(row: unknown): 'info' | 'warning' | 'success' | 'danger' { return statusTypes[(row as MailMessageSummary).status] }
function applyFilters(): void { page.value = 1; void load() }
function handleError(caught: unknown): void {
  if (caught instanceof AdminAPIError && caught.status === 401) { emit('unauthorized'); return }
  error.value = caught instanceof Error ? caught.message : 'Не удалось загрузить историю писем.'
}

watch(() => selected.selectedSite.value?.id, () => { page.value = 1; void load() })
onMounted(() => void load())
</script>

<template>
  <access-denied-view v-if="!permissions.has('mail.message.read')" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page mail-history-page">
    <header class="page-header"><div><h1>История писем</h1><p>Неизменяемые снимки сообщений и результаты попыток доставки</p></div></header>
    <el-alert v-if="!selected.selectedSite.value" type="warning" :closable="false" title="Выберите сайт в боковой панели." />
    <el-alert v-else-if="error" type="error" :closable="false" :title="error" show-icon />
    <div v-if="selected.selectedSite.value" class="mail-history-filters">
      <el-select v-model="filters.status" clearable placeholder="Все статусы"><el-option v-for="(label, value) in statusLabels" :key="value" :value="value" :label="label" /></el-select>
      <el-input v-model="filters.template_code" clearable placeholder="Код шаблона" />
      <el-input v-model="filters.recipient" clearable placeholder="Email получателя" />
      <el-date-picker v-model="filters.date_from" type="datetime" placeholder="С даты" />
      <el-date-picker v-model="filters.date_to" type="datetime" placeholder="По дату" />
      <el-button type="primary" @click="applyFilters">Применить</el-button>
    </div>
    <el-table v-if="selected.selectedSite.value" v-loading="loading" :data="items" stripe empty-text="История пока пуста" @row-click="router.push({ name: 'mail.history.detail', params: { messageId: $event.id } })">
      <el-table-column label="Дата" width="180"><template #default="{ row }">{{ new Date(row.requested_at).toLocaleString() }}</template></el-table-column>
      <el-table-column label="Шаблон" min-width="200"><template #default="{ row }"><strong>{{ row.template_name }}</strong><br><code>{{ row.template_code }}</code></template></el-table-column>
      <el-table-column label="Получатели" min-width="260"><template #default="{ row }">{{ recipients(row) }}</template></el-table-column>
      <el-table-column label="Источник" min-width="180"><template #default="{ row }">{{ origin(row) }}</template></el-table-column>
      <el-table-column label="Статус" width="180"><template #default="{ row }"><el-tag :type="statusType(row)">{{ statusLabel(row) }}</el-tag></template></el-table-column>
      <el-table-column label="Попытки / результат" min-width="220"><template #default="{ row }">{{ latest(row) }}</template></el-table-column>
      <el-table-column label="Действия" width="160" align="right"><template #default="{ row }">
        <el-button text type="primary" @click.stop="router.push({ name: 'mail.history.detail', params: { messageId: row.id } })">Открыть</el-button>
        <el-button v-if="permissions.has('mail.message.delete') && terminal(row)" text type="danger" @click.stop="remove(row)">Удалить</el-button>
      </template></el-table-column>
    </el-table>
    <el-pagination v-if="total > perPage" background layout="total, prev, pager, next" :current-page="page" :page-size="perPage" :total="total" @current-change="page = $event; load()" />
  </section>
</template>

<style scoped>.mail-history-page{display:grid;gap:16px}.mail-history-page :deep(.el-table__row){cursor:pointer}.mail-history-filters{display:grid;grid-template-columns:180px 1fr 1fr 210px 210px auto;gap:10px}@media(max-width:1100px){.mail-history-filters{grid-template-columns:1fr 1fr}}</style>
