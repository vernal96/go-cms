<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ElButton, ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'

import { AdminAPIError, adminRequest, adminRequestVoid } from '../api/admin-api'
import AdminDataTable from '../components/AdminDataTable.vue'
import { useSelectedSite } from '../composables/use-selected-site'
import type {
  AdminTableAction,
  AdminTableColumn,
  PermissionSet,
  Site,
  SiteListResponse,
} from '../types/admin'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const router = useRouter()
const selected = useSelectedSite()
const rows = ref<Record<string, unknown>[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const search = ref('')
const page = ref(1)
const total = ref(0)
const perPage = 10
const permissions = ref<PermissionSet>({ read: false, create: false, update: false, delete: false })
let controller: AbortController | null = null
let debounceTimer: ReturnType<typeof setTimeout> | null = null
let sequence = 0

const columns: AdminTableColumn[] = [
  { prop: 'domain', label: 'Домен' },
  { prop: 'profile_code', label: 'Профиль' },
  { prop: 'locale', label: 'Локаль', width: 130 },
  { prop: 'is_public', label: 'Доступ', width: 130, formatter: (row) => row.is_public ? 'Публичный' : 'Закрытый' },
]
const actions: AdminTableAction[] = [
  { key: 'edit', label: 'Изменить', visible: (row) => permissions.value.update && Boolean((row as unknown as Site).capabilities?.edit) },
  { key: 'delete', label: 'Удалить', danger: true, visible: (row) => permissions.value.delete && Boolean((row as unknown as Site).capabilities?.delete) },
]

async function load(): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  const current = ++sequence
  loading.value = true
  error.value = null
  const query = new URLSearchParams({ search: search.value, page: String(page.value), per_page: String(perPage) })
  try {
    const response = await adminRequest<SiteListResponse>(
      `/api/sites?${query}`,
      props.accessToken,
      { signal: controller.signal },
    )
    if (current !== sequence) return
    rows.value = response.items as unknown as Record<string, unknown>[]
    total.value = response.pagination.total
    permissions.value = response.permissions
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
  const site = row as unknown as Site
  if (key === 'edit') {
	if (!site.capabilities?.edit) return
    await router.push(`/admin/sites/${site.id}/edit`)
    return
  }
  if (key !== 'delete') return
	if (!site.capabilities?.delete) return
  try {
    await ElMessageBox.confirm(`Удалить сайт ${site.domain} и все его ресурсы?`, 'Удаление сайта', {
      confirmButtonText: 'Удалить', cancelButtonText: 'Отмена', type: 'warning',
    })
  } catch {
    return
  }
  try {
    await adminRequestVoid(`/api/sites/${site.id}`, props.accessToken, { method: 'DELETE' })
    if (selected.selectedSite.value?.id === site.id) selected.clearSelected()
    if (rows.value.length === 1 && page.value > 1) page.value -= 1
    ElMessage.success('Сайт удалён')
    selected.refreshSelector()
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
  error.value = caught instanceof Error ? caught.message : 'Не удалось загрузить сайты.'
}

onMounted(() => void load())
onBeforeUnmount(() => {
  controller?.abort()
  if (debounceTimer !== null) clearTimeout(debounceTimer)
})
</script>

<template>
  <section class="workspace-page">
    <header class="page-header">
      <div><h1>Сайты</h1><p>Домены и профили проекта</p></div>
    </header>
    <admin-data-table
      :columns="columns"
      :rows="rows"
      :actions="actions"
      :loading="loading"
      :error="error"
      :page="page"
      :per-page="perPage"
      :total="total"
      :search="search"
      empty-text="Сайтов пока нет"
      @update:search="updateSearch"
      @page-change="page = $event; load()"
      @retry="load"
      @action="handleAction"
    >
      <template #toolbar>
        <el-button
          v-if="permissions.create || props.permissions.has('core.site.create')"
          type="primary"
          :icon="Plus"
          @click="router.push('/admin/sites/create')"
        >Создать сайт</el-button>
      </template>
    </admin-data-table>
  </section>
</template>
