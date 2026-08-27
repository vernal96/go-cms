<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElAlert, ElButton, ElMessage, ElMessageBox, ElPagination, ElTable, ElTableColumn, ElTag } from 'element-plus'
import { useRouter } from 'vue-router'
import { AdminAPIError } from '../../api/admin-api'
import AccessDeniedView from '../../components/AccessDeniedView.vue'
import { useSelectedSite } from '../../composables/use-selected-site'
import { deleteMailTemplate, listMailTemplates } from './api'
import type { MailTemplate } from './types'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const router = useRouter()
const selected = useSelectedSite()
const items = ref<MailTemplate[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const page = ref(1)
const perPage = 20
const total = ref(0)

async function load(): Promise<void> {
  if (!props.permissions.has('mail.template.read')) return
  const siteID = selected.selectedSite.value?.id
  if (!siteID) { items.value = []; total.value = 0; return }
  loading.value = true
  error.value = null
  try {
    const response = await listMailTemplates(props.accessToken, siteID, page.value, perPage)
    items.value = response.items
    total.value = response.pagination.total
  } catch (caught) { handleError(caught) }
  finally { loading.value = false }
}

async function remove(row: unknown): Promise<void> {
  const item = row as MailTemplate
  const siteID = selected.selectedSite.value?.id
  if (!siteID || !props.permissions.has('mail.template.delete')) return
  try {
    await ElMessageBox.confirm(`Удалить шаблон «${item.name}»? Существующая история писем сохранится.`, 'Удаление шаблона', { confirmButtonText: 'Удалить', cancelButtonText: 'Отмена', type: 'warning' })
  } catch { return }
  try {
    await deleteMailTemplate(props.accessToken, siteID, item.id)
    ElMessage.success('Шаблон удалён')
    await load()
  } catch (caught) { handleError(caught) }
}

function handleError(caught: unknown): void {
  if (caught instanceof AdminAPIError && caught.status === 401) { emit('unauthorized'); return }
  error.value = caught instanceof Error ? caught.message : 'Не удалось загрузить шаблоны.'
}

watch(() => selected.selectedSite.value?.id, () => { page.value = 1; void load() })
onMounted(() => void load())
</script>

<template>
  <access-denied-view v-if="!permissions.has('mail.template.read')" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page mail-list-page">
    <header class="page-header">
      <div><h1>Шаблоны писем</h1><p>Site-scoped адреса, содержимое, переменные и вложения</p></div>
      <el-button v-if="permissions.has('mail.template.create')" type="primary" :icon="Plus" @click="router.push({ name: 'mail.templates.create' })">Создать шаблон</el-button>
    </header>
    <el-alert v-if="!selected.selectedSite.value" type="warning" :closable="false" title="Выберите сайт в боковой панели." />
    <el-alert v-else-if="error" type="error" :closable="false" :title="error" show-icon />
    <el-table v-else v-loading="loading" :data="items" stripe empty-text="Шаблонов пока нет">
      <el-table-column prop="name" label="Название" min-width="220" />
      <el-table-column prop="code" label="Код" min-width="190" />
      <el-table-column prop="transport" label="Транспорт" min-width="130" />
      <el-table-column label="Состояние" width="130"><template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? 'Включён' : 'Выключен' }}</el-tag></template></el-table-column>
      <el-table-column label="Изменён" width="180"><template #default="{ row }">{{ new Date(row.updated_at).toLocaleString() }}</template></el-table-column>
      <el-table-column label="Действия" width="190" align="right"><template #default="{ row }">
        <el-button v-if="permissions.has('mail.template.update')" text type="primary" @click="router.push({ name: 'mail.templates.edit', params: { templateId: row.id } })">Изменить</el-button>
        <el-button v-if="permissions.has('mail.template.delete')" text type="danger" @click="remove(row)">Удалить</el-button>
      </template></el-table-column>
    </el-table>
    <el-pagination v-if="total > perPage" background layout="total, prev, pager, next" :current-page="page" :page-size="perPage" :total="total" @current-change="page = $event; load()" />
  </section>
</template>

<style scoped>.mail-list-page{display:grid;gap:16px}</style>
