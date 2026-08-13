<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ElButton, ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'

import { AdminAPIError, adminRequest, adminRequestVoid } from '../api/admin-api'
import AccessDeniedView from '../components/AccessDeniedView.vue'
import AdminDataTable from '../components/AdminDataTable.vue'
import type { AdminGroup, AdminTableAction, AdminTableColumn, GroupListResponse } from '../types/admin'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const router = useRouter()
const rows = ref<Record<string, unknown>[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const search = ref('')
const page = ref(1)
const total = ref(0)
const perPage = 10
let controller: AbortController | null = null
let debounceTimer: ReturnType<typeof setTimeout> | null = null
let sequence = 0

const columns: AdminTableColumn[] = [
  { prop: 'name', label: 'Название' },
  { prop: 'code', label: 'Код' },
  { prop: 'system', label: 'Тип', width: 150, formatter: (row) => row.system ? 'Системная' : 'Обычная' },
]
const actions: AdminTableAction[] = [
  { key: 'edit', label: 'Изменить', visible: (row) => (row as unknown as AdminGroup).can_update },
  { key: 'delete', label: 'Удалить', danger: true, visible: (row) => (row as unknown as AdminGroup).can_delete },
]

async function load(): Promise<void> {
  if (!props.permissions.has('core.group.read')) return
  controller?.abort()
  controller = new AbortController()
  const current = ++sequence
  loading.value = true
  error.value = null
  const query = new URLSearchParams({ search: search.value, page: String(page.value), per_page: String(perPage) })
  try {
    const response = await adminRequest<GroupListResponse>(`/api/admin/groups?${query}`, props.accessToken, { signal: controller.signal })
    if (current !== sequence) return
    rows.value = response.items as unknown as Record<string, unknown>[]
    total.value = response.pagination.total
  } catch (caught) {
    if (caught instanceof DOMException && caught.name === 'AbortError') return
    handleError(caught)
  } finally {
    if (current === sequence) loading.value = false
  }
}

function updateSearch(value: string): void {
  search.value = value.trim()
  page.value = 1
  if (debounceTimer !== null) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => void load(), 300)
}

async function handleAction(key: string, row: Record<string, unknown>): Promise<void> {
  const item = row as unknown as AdminGroup
  if (key === 'edit') {
    await router.push(`/admin/groups/${item.id}/edit`)
    return
  }
  if (key !== 'delete') return
  try {
    await ElMessageBox.confirm(`Удалить группу «${item.name}»? Пользователи потеряют назначенные через неё права.`, 'Удаление группы', { confirmButtonText: 'Удалить', cancelButtonText: 'Отмена', type: 'warning' })
  } catch {
    return
  }
  try {
    await adminRequestVoid(`/api/admin/groups/${item.id}`, props.accessToken, { method: 'DELETE' })
    ElMessage.success('Группа удалена')
    await load()
  } catch (caught) {
    handleError(caught)
  }
}

function handleError(caught: unknown): void {
  if (caught instanceof AdminAPIError && caught.status === 401) {
    emit('unauthorized')
    return
  }
  error.value = caught instanceof Error ? caught.message : 'Не удалось загрузить группы.'
}

onMounted(() => void load())
onBeforeUnmount(() => {
  controller?.abort()
  if (debounceTimer !== null) clearTimeout(debounceTimer)
})
</script>

<template>
  <access-denied-view v-if="!permissions.has('core.group.read')" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page">
    <header class="page-header"><div><h1>Группы</h1><p>Роли пользователей и настройки прав</p></div></header>
    <admin-data-table
      :columns="columns" :rows="rows" :actions="actions" :loading="loading" :error="error"
      :page="page" :per-page="perPage" :total="total" :search="search" empty-text="Групп пока нет"
      @update:search="updateSearch" @page-change="page = $event; load()" @retry="load" @action="handleAction"
    >
      <template #toolbar><el-button v-if="permissions.has('core.group.create')" type="primary" :icon="Plus" @click="router.push('/admin/groups/create')">Создать группу</el-button></template>
    </admin-data-table>
  </section>
</template>
