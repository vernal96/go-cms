<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { ElAlert, ElButton, ElDatePicker, ElOption, ElPagination, ElSelect, ElTable, ElTableColumn, ElTag } from 'element-plus'
import { useRouter } from 'vue-router'
import { AdminAPIError } from '../../api/admin-api'
import AccessDeniedView from '../../components/AccessDeniedView.vue'
import { useSelectedSite } from '../../composables/use-selected-site'
import { getFormEditor, listForms, listResults } from './api'
import type { FormField, FormRecord, FormResultSummary, FormStatus } from './types'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const selected = useSelectedSite(); const router = useRouter()
const items = ref<FormResultSummary[]>([]); const columns = ref<FormField[]>([]); const forms = ref<FormRecord[]>([]); const statuses = ref<FormStatus[]>([])
const loading = ref(false); const error = ref<string | null>(null); const page = ref(1); const perPage = 20; const total = ref(0)
const filters = reactive<{ form_id?: number; status_id?: number; date_from: Date | null; date_to: Date | null }>({ date_from: null, date_to: null })
function handleError(caught: unknown): void { if (caught instanceof AdminAPIError && caught.status === 401) { emit('unauthorized'); return }; error.value = caught instanceof Error ? caught.message : 'Не удалось загрузить результаты.' }
async function loadForms(): Promise<void> { const siteID = selected.selectedSite.value?.id; if (!siteID || !props.permissions.has('forms.form.read')) return; try { forms.value = (await listForms(props.accessToken, siteID, 1, 100)).items } catch { forms.value = [] } }
async function loadStatuses(): Promise<void> { const siteID = selected.selectedSite.value?.id; statuses.value = []; filters.status_id = undefined; if (!siteID || !filters.form_id || !props.permissions.has('forms.form.read') || !props.permissions.has('forms.status.read') || !props.permissions.has('forms.action.read')) return; try { statuses.value = (await getFormEditor(props.accessToken, siteID, filters.form_id)).statuses } catch { statuses.value = [] } }
async function load(): Promise<void> { const siteID = selected.selectedSite.value?.id; if (!siteID || !props.permissions.has('forms.result.read')) return; loading.value = true; error.value = null; try { const response = await listResults(props.accessToken, siteID, page.value, perPage, { form_id: filters.form_id, status_id: filters.status_id, date_from: filters.date_from?.toISOString(), date_to: filters.date_to?.toISOString() }); items.value = response.items; columns.value = response.columns; total.value = response.pagination.total } catch (caught) { handleError(caught) } finally { loading.value = false } }
function display(value: unknown): string { if (value === null || value === undefined || value === '') return '—'; if (Array.isArray(value)) return value.map(display).join(', '); if (typeof value === 'boolean') return value ? 'Да' : 'Нет'; if (typeof value === 'object') return JSON.stringify(value); return String(value) }
function apply(): void { page.value = 1; void load() }
watch(() => filters.form_id, async () => { await loadStatuses(); apply() })
watch(() => selected.selectedSite.value?.id, async () => { page.value = 1; filters.form_id = undefined; filters.status_id = undefined; await loadForms(); await load() })
onMounted(async () => { await loadForms(); await load() })
</script>
<template>
  <access-denied-view v-if="!permissions.has('forms.result.read')" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page forms-results-page">
    <header class="page-header"><div><h1>Результаты форм</h1><p>Ответы, бизнес-статусы и история асинхронных действий</p></div></header>
    <el-alert v-if="!selected.selectedSite.value" type="warning" :closable="false" title="Выберите сайт в боковой панели." />
    <el-alert v-else-if="error" type="error" :closable="false" :title="error" show-icon />
    <div v-if="selected.selectedSite.value" class="result-filters"><el-select v-model="filters.form_id" clearable placeholder="Все формы"><el-option v-for="form in forms" :key="form.id" :value="form.id" :label="form.name" /></el-select><el-select v-model="filters.status_id" clearable :disabled="!filters.form_id" placeholder="Все статусы"><el-option v-for="status in statuses" :key="status.id" :value="status.id" :label="status.name" /></el-select><el-date-picker v-model="filters.date_from" type="datetime" placeholder="С даты" /><el-date-picker v-model="filters.date_to" type="datetime" placeholder="По дату" /><el-button type="primary" @click="apply">Применить</el-button></div>
    <el-table v-if="selected.selectedSite.value" v-loading="loading" :data="items" stripe empty-text="Результатов пока нет" @row-click="router.push({ name: 'forms.results.detail', params: { resultId: $event.id } })">
      <el-table-column prop="id" label="Результат" width="110"><template #default="{ row }">#{{ row.id }}</template></el-table-column>
      <el-table-column v-if="!filters.form_id" label="Форма" min-width="200"><template #default="{ row }"><strong>{{ row.form_name }}</strong><br><code>{{ row.form_code }}</code></template></el-table-column>
      <el-table-column label="Статус" width="170"><template #default="{ row }"><el-tag :color="row.status_color || undefined" effect="plain">{{ row.status_name }}</el-tag></template></el-table-column>
      <el-table-column v-for="column in columns" :key="column.id" :label="column.result_label || column.label" min-width="170"><template #default="{ row }">{{ display(row.values[column.code]) }}</template></el-table-column>
      <el-table-column label="Пользователь" width="150"><template #default="{ row }">{{ row.user_id ? `#${row.user_id}` : 'Аноним' }}</template></el-table-column>
      <el-table-column label="Создан" width="180"><template #default="{ row }">{{ new Date(row.created_at).toLocaleString() }}</template></el-table-column>
      <el-table-column label="" width="100"><template #default="{ row }"><el-button text type="primary" @click.stop="router.push({ name: 'forms.results.detail', params: { resultId: row.id } })">Открыть</el-button></template></el-table-column>
    </el-table>
    <el-pagination v-if="total > perPage" background layout="total, prev, pager, next" :current-page="page" :page-size="perPage" :total="total" @current-change="page = $event; load()" />
  </section>
</template>
<style scoped>.forms-results-page{display:grid;gap:16px}.forms-results-page :deep(.el-table__row){cursor:pointer}.result-filters{display:grid;grid-template-columns:repeat(4,minmax(170px,1fr)) auto;gap:10px}.result-filters>*{width:100%}@media(max-width:900px){.result-filters{grid-template-columns:repeat(2,1fr)}}@media(max-width:560px){.result-filters{grid-template-columns:1fr}}</style>

