<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ElButton, ElMessage, ElMessageBox, ElOption, ElSelect } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'

import { AdminAPIError, adminRequest, adminRequestVoid } from '../api/admin-api'
import AccessDeniedView from '../components/AccessDeniedView.vue'
import AdminDataTable from '../components/AdminDataTable.vue'
import type { AdminTableAction, AdminTableColumn, AdminUser, UserListResponse } from '../types/admin'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const router = useRouter()
const rows = ref<Record<string, unknown>[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const search = ref('')
const status = ref<'all' | 'active' | 'blocked'>('all')
const page = ref(1)
const total = ref(0)
const perPage = 10
let controller: AbortController | null = null
let debounceTimer: ReturnType<typeof setTimeout> | null = null
let sequence = 0

const columns: AdminTableColumn[] = [
  { prop: 'name', label: 'Пользователь', formatter: (row) => displayName(row as unknown as AdminUser) },
  { prop: 'login', label: 'Логин' },
  { prop: 'email', label: 'Email' },
  { prop: 'blocked', label: 'Статус', width: 130, formatter: (row) => row.blocked ? 'Заблокирован' : 'Активен' },
  { prop: 'last_login_at', label: 'Последний вход', width: 180, formatter: (row) => formatDate(row.last_login_at) },
]

const actions: AdminTableAction[] = [
  { key: 'edit', label: 'Изменить', visible: (row) => (row as unknown as AdminUser).capabilities.update },
  { key: 'block', label: 'Заблокировать', danger: true, visible: (row) => (row as unknown as AdminUser).capabilities.block },
  { key: 'unblock', label: 'Разблокировать', visible: (row) => (row as unknown as AdminUser).capabilities.unblock },
]

async function load(): Promise<void> {
  if (!props.permissions.has('core.user.read')) return
  controller?.abort()
  controller = new AbortController()
  const current = ++sequence
  loading.value = true
  error.value = null
  const query = new URLSearchParams({ search: search.value, status: status.value, page: String(page.value), per_page: String(perPage) })
  try {
    const response = await adminRequest<UserListResponse>(`/api/admin/users?${query}`, props.accessToken, { signal: controller.signal })
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
  const item = row as unknown as AdminUser
  if (key === 'edit') {
    await router.push(`/admin/users/${item.id}/edit`)
    return
  }
  if (key !== 'block' && key !== 'unblock') return
  const blocking = key === 'block'
  try {
    await ElMessageBox.confirm(
      blocking ? `Заблокировать пользователя ${displayName(item)}?` : `Разблокировать пользователя ${displayName(item)}?`,
      blocking ? 'Блокировка пользователя' : 'Разблокировка пользователя',
      { confirmButtonText: blocking ? 'Заблокировать' : 'Разблокировать', cancelButtonText: 'Отмена', type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await adminRequestVoid(`/api/admin/users/${item.id}/${blocking ? 'block' : 'unblock'}`, props.accessToken, { method: 'POST' })
    ElMessage.success(blocking ? 'Пользователь заблокирован' : 'Пользователь разблокирован')
    await load()
  } catch (caught) {
    handleError(caught)
  }
}

function displayName(item: AdminUser): string {
  return [item.last_name, item.name, item.middle_name].filter(Boolean).join(' ') || item.login
}

function formatDate(value: unknown): string {
  if (typeof value !== 'string' || !value) return '—'
  return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value))
}

function handleError(caught: unknown): void {
  if (caught instanceof AdminAPIError && caught.status === 401) {
    emit('unauthorized')
    return
  }
  error.value = caught instanceof Error ? caught.message : 'Не удалось загрузить пользователей.'
}

onMounted(() => void load())
onBeforeUnmount(() => {
  controller?.abort()
  if (debounceTimer !== null) clearTimeout(debounceTimer)
})
</script>

<template>
  <access-denied-view v-if="!permissions.has('core.user.read')" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page">
    <header class="page-header"><div><h1>Пользователи</h1><p>Учётные записи и состояние доступа</p></div></header>
    <admin-data-table
      :columns="columns" :rows="rows" :actions="actions" :loading="loading" :error="error"
      :page="page" :per-page="perPage" :total="total" :search="search" empty-text="Пользователей пока нет"
      @update:search="updateSearch" @page-change="page = $event; load()" @retry="load" @action="handleAction"
    >
      <template #toolbar>
        <el-select v-model="status" class="status-filter" aria-label="Статус пользователя" @change="page = 1; load()">
          <el-option label="Все" value="all" />
          <el-option label="Активные" value="active" />
          <el-option label="Заблокированные" value="blocked" />
        </el-select>
        <el-button v-if="permissions.has('core.user.create')" type="primary" :icon="Plus" @click="router.push('/admin/users/create')">Создать пользователя</el-button>
      </template>
    </admin-data-table>
  </section>
</template>
