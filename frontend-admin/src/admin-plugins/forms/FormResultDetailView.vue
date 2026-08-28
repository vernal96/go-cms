<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElAlert, ElButton, ElCard, ElDescriptions, ElDescriptionsItem, ElMessage, ElMessageBox, ElOption, ElSelect, ElTable, ElTableColumn, ElTag } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { AdminAPIError } from '../../api/admin-api'
import AccessDeniedView from '../../components/AccessDeniedView.vue'
import { useSelectedSite } from '../../composables/use-selected-site'
import { changeResultStatus, deleteResult, getResult } from './api'
import type { FormStatus, ResultDetailResponse } from './types'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const selected = useSelectedSite(); const route = useRoute(); const router = useRouter(); const resultID = computed(() => Number(route.params.resultId))
const detail = ref<ResultDetailResponse | null>(null); const statuses = ref<FormStatus[]>([]); const statusID = ref<number>(); const loading = ref(false); const error = ref<string | null>(null)
const executionLabels = { pending: 'В очереди', running: 'Выполняется', retryable: 'Будет повторено', succeeded: 'Выполнено', failed: 'Ошибка' } as const
function handleError(caught: unknown): void { if (caught instanceof AdminAPIError && caught.status === 401) { emit('unauthorized'); return }; error.value = caught instanceof Error ? caught.message : 'Не удалось открыть результат.' }
async function load(): Promise<void> { const siteID = selected.selectedSite.value?.id; if (!siteID || !resultID.value) return; loading.value = true; error.value = null; try { detail.value = await getResult(props.accessToken, siteID, resultID.value); statusID.value = detail.value.result.status_id; statuses.value = detail.value.available_statuses ?? [] } catch (caught) { handleError(caught) } finally { loading.value = false } }
async function saveStatus(): Promise<void> { const siteID = selected.selectedSite.value?.id; if (!siteID || !statusID.value || !detail.value) return; try { detail.value = await changeResultStatus(props.accessToken, siteID, resultID.value, statusID.value); statusID.value = detail.value.result.status_id; statuses.value = detail.value.available_statuses ?? statuses.value; ElMessage.success('Статус результата изменён') } catch (caught) { handleError(caught) } }
async function remove(): Promise<void> { const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { await ElMessageBox.confirm(`Удалить результат #${resultID.value}?`, 'Удаление результата', { type: 'warning' }); await deleteResult(props.accessToken, siteID, resultID.value); ElMessage.success('Результат удалён'); await router.push({ name: 'forms.results' }) } catch (caught) { if (caught instanceof Error) handleError(caught) } }
function display(value: unknown): string { if (value === null || value === undefined || value === '') return '—'; if (typeof value === 'boolean') return value ? 'Да' : 'Нет'; if (typeof value === 'object') return JSON.stringify(value); return String(value) }
function executionLabel(source: unknown): string { const status = (source as { status: keyof typeof executionLabels }).status; return executionLabels[status] ?? status }
watch(() => selected.selectedSite.value?.id, () => void load()); onMounted(() => void load())
</script>
<template>
  <access-denied-view v-if="!permissions.has('forms.result.read')" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page result-detail-page" v-loading="loading">
    <header class="page-header"><div><h1>Результат #{{ resultID }}</h1><p v-if="detail">{{ detail.result.form_name }} · {{ new Date(detail.result.created_at).toLocaleString() }}</p></div><span><el-button @click="router.push({ name: 'forms.results' })">К результатам</el-button> <el-button v-if="permissions.has('forms.result.delete')" type="danger" plain @click="remove">Удалить</el-button></span></header>
    <el-alert v-if="error" type="error" :closable="false" :title="error" show-icon />
    <template v-if="detail">
      <el-card><el-descriptions :column="2" border><el-descriptions-item label="Форма">{{ detail.result.form_name }} ({{ detail.result.form_code }})</el-descriptions-item><el-descriptions-item label="Пользователь">{{ detail.result.user_id ? `#${detail.result.user_id}` : 'Аноним' }}</el-descriptions-item><el-descriptions-item label="User-Agent">{{ detail.result.user_agent || '—' }}</el-descriptions-item><el-descriptions-item label="Адрес клиента">{{ detail.result.client_address || 'Не сохраняется' }}</el-descriptions-item></el-descriptions><div class="status-control"><strong>Бизнес-статус</strong><el-select v-model="statusID" :disabled="!permissions.has('forms.result.update') || !statuses.length"><el-option v-for="status in statuses" :key="status.id" :value="status.id" :label="status.name" /></el-select><el-button v-if="permissions.has('forms.result.update')" type="primary" :disabled="statusID === detail.result.status_id" @click="saveStatus">Изменить</el-button></div></el-card>
      <el-card><template #header><strong>Ответы</strong></template><el-table :data="detail.values" stripe><el-table-column prop="result_label" label="Поле" min-width="220" /><el-table-column prop="field_code" label="Код" min-width="160" /><el-table-column label="Значение" min-width="300"><template #default="{ row }">{{ display(row.value) }}</template></el-table-column></el-table></el-card>
      <el-card v-if="detail.uploads.length"><template #header><strong>Загруженные файлы</strong></template><el-table :data="detail.uploads"><el-table-column prop="field_code" label="Поле" /><el-table-column prop="filename" label="Имя" /><el-table-column prop="mime_type" label="MIME" /><el-table-column label="Размер"><template #default="{ row }">{{ row.size.toLocaleString() }} байт</template></el-table-column><el-table-column label="Временные байты"><template #default="{ row }"><el-tag :type="row.spool_deleted_at ? 'info' : 'warning'">{{ row.spool_deleted_at ? 'Удалены' : 'Ожидают действий' }}</el-tag></template></el-table-column></el-table></el-card>
      <el-card><template #header><strong>Исполнения действий</strong></template><el-table :data="detail.action_executions" empty-text="Действия не запускались"><el-table-column prop="action_name" label="Действие" /><el-table-column prop="action_type" label="Тип" /><el-table-column label="Статус"><template #default="{ row }"><el-tag :type="row.status === 'succeeded' ? 'success' : row.status === 'failed' ? 'danger' : 'warning'">{{ executionLabel(row) }}</el-tag></template></el-table-column><el-table-column prop="attempt_count" label="Попытки" width="100" /><el-table-column prop="external_reference" label="Внешняя ссылка" /><el-table-column prop="safe_error" label="Безопасная ошибка" min-width="240" /></el-table></el-card>
    </template>
  </section>
</template>
<style scoped>.result-detail-page{display:grid;gap:16px}.status-control{display:flex;align-items:center;gap:12px;margin-top:16px}.status-control .el-select{min-width:220px}@media(max-width:650px){.status-control{align-items:stretch;flex-direction:column}}</style>
