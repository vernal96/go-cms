<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElButton, ElDialog, ElEmpty, ElMessage, ElMessageBox, ElPagination, ElTable, ElTableColumn } from 'element-plus'
import { AdminAPIError, adminRequest } from '../api/admin-api'
import type { ResourceDetailsResponse, ResourceRevision, ResourceRevisionPage } from '../types/admin'

const props = defineProps<{ accessToken: string; siteId: number; resourceId: number; resourceVersion: number; canRestore: boolean; canDelete: boolean }>()
const emit = defineEmits<{ changed: []; unauthorized: [] }>()
const items = ref<ResourceRevision[]>([])
const total = ref(0)
const page = ref(1)
const perPage = 20
const loading = ref(false)
const pending = ref(false)
const selected = ref<ResourceRevision | null>(null)
const detailOpen = ref(false)

async function load(): Promise<void> {
	loading.value = true
	try {
		const result = await adminRequest<ResourceRevisionPage>(`/api/sites/${props.siteId}/resources/${props.resourceId}/revisions?page=${page.value}&per_page=${perPage}`, props.accessToken)
		items.value = result.items; total.value = result.total
	} catch (error) { handle(error, 'Не удалось загрузить историю.') } finally { loading.value = false }
}
async function open(item: ResourceRevision): Promise<void> {
	try { selected.value = await adminRequest<ResourceRevision>(`/api/sites/${props.siteId}/resources/${props.resourceId}/revisions/${item.version}`, props.accessToken); detailOpen.value = true }
	catch (error) { handle(error, 'Не удалось открыть ревизию.') }
}
function openRow(row: unknown): void { void open(row as ResourceRevision) }
function restoreRow(row: unknown): void { void restore(row as ResourceRevision) }
async function restore(item: ResourceRevision): Promise<void> {
	try { await ElMessageBox.confirm(`Текущее состояние будет заменено содержимым версии ${item.version}. История сохранится, а восстановление создаст новую версию.`, 'Восстановить ревизию?', { type: 'warning', confirmButtonText: 'Восстановить', cancelButtonText: 'Отмена' }) } catch { return }
	pending.value = true
	try {
		await adminRequest<ResourceDetailsResponse>(`/api/sites/${props.siteId}/resources/${props.resourceId}/revisions/${item.version}/restore`, props.accessToken, { method: 'POST', body: JSON.stringify({ expected_version: props.resourceVersion }) })
		selected.value = null; detailOpen.value = false; ElMessage.success('Ревизия восстановлена'); emit('changed'); await load()
	} catch (error) { handle(error, 'Не удалось восстановить ревизию.') } finally { pending.value = false }
}
async function purge(): Promise<void> {
	try { await ElMessageBox.confirm(`Будут навсегда удалены все ${total.value} ревизий этого ресурса. Текущий ресурс и его версия не изменятся.`, 'Очистить историю?', { type: 'error', confirmButtonText: 'Очистить', cancelButtonText: 'Отмена' }) } catch { return }
	pending.value = true
	try { const result = await adminRequest<{ count: number }>(`/api/sites/${props.siteId}/resources/${props.resourceId}/revisions`, props.accessToken, { method: 'DELETE' }); ElMessage.success(`Удалено ревизий: ${result.count}`); page.value = 1; await load() }
	catch (error) { handle(error, 'Не удалось очистить историю.') } finally { pending.value = false }
}
function handle(error: unknown, fallback: string): void { if (error instanceof AdminAPIError && error.status === 401) emit('unauthorized'); else ElMessage.error(error instanceof Error ? error.message : fallback) }
watch(page, () => void load())
onMounted(() => void load())
</script>

<template>
	<div v-loading="loading">
		<div class="history-actions"><el-button v-if="canDelete" type="danger" plain :disabled="pending || total === 0" @click="purge">Очистить историю</el-button></div>
		<el-empty v-if="!loading && items.length === 0" description="История пока пуста" />
		<el-table v-else :data="items">
			<el-table-column prop="version" label="Версия" width="100" />
			<el-table-column label="Дата"><template #default="scope">{{ new Date(scope.row.created_at).toLocaleString() }}</template></el-table-column>
			<el-table-column prop="created_by_name" label="Автор" />
			<el-table-column label="Действия" width="220"><template #default="scope"><el-button link @click="openRow(scope.row)">Открыть</el-button><el-button v-if="canRestore" link type="primary" :disabled="pending" @click="restoreRow(scope.row)">Восстановить</el-button></template></el-table-column>
		</el-table>
		<el-pagination v-if="total > perPage" v-model:current-page="page" :page-size="perPage" :total="total" layout="prev, pager, next" />
		<el-dialog v-model="detailOpen" :title="selected ? `Версия ${selected.version}` : 'Ревизия'" width="720px"><pre class="snapshot">{{ JSON.stringify(selected?.snapshot, null, 2) }}</pre></el-dialog>
	</div>
</template>

<style scoped>.history-actions{display:flex;justify-content:flex-end;margin-bottom:1rem}.snapshot{max-height:60vh;overflow:auto;white-space:pre-wrap}</style>
