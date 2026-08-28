<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElAlert, ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElMessage, ElMessageBox, ElPagination, ElSwitch, ElTable, ElTableColumn, ElTag } from 'element-plus'
import { useRouter } from 'vue-router'
import { AdminAPIError } from '../../api/admin-api'
import AccessDeniedView from '../../components/AccessDeniedView.vue'
import { useSelectedSite } from '../../composables/use-selected-site'
import { createForm, deleteForm, listForms, setFormEnabled } from './api'
import type { FormPayload, FormRecord } from './types'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const router = useRouter()
const selected = useSelectedSite()
const items = ref<FormRecord[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const search = ref('')
const page = ref(1)
const perPage = 20
const total = ref(0)
const createOpen = ref(false)
const saving = ref(false)
const form = reactive<FormPayload>({ code: '', name: '', description: '', enabled: true })

function handleError(caught: unknown, fallback: string): void {
  if (caught instanceof AdminAPIError && caught.status === 401) { emit('unauthorized'); return }
  error.value = caught instanceof Error ? caught.message : fallback
}
async function load(): Promise<void> {
  const siteID = selected.selectedSite.value?.id
  if (!siteID || !props.permissions.has('forms.form.read')) { items.value = []; total.value = 0; return }
  loading.value = true; error.value = null
  try {
    const response = await listForms(props.accessToken, siteID, page.value, perPage, search.value)
    items.value = response.items; total.value = response.pagination.total
  } catch (caught) { handleError(caught, 'Не удалось загрузить формы.') }
  finally { loading.value = false }
}
function openCreate(): void {
  Object.assign(form, { code: '', name: '', description: '', enabled: true })
  createOpen.value = true
}
async function submitCreate(): Promise<void> {
  const siteID = selected.selectedSite.value?.id
  if (!siteID || !form.code.trim() || !form.name.trim()) return
  saving.value = true; error.value = null
  try {
    const created = await createForm(props.accessToken, siteID, { ...form, code: form.code.trim(), name: form.name.trim(), description: form.description.trim() })
    createOpen.value = false
    ElMessage.success('Форма создана с обязательными полями и статусом «Новый»')
    await router.push({ name: 'forms.edit', params: { formId: created.id } })
  } catch (caught) { handleError(caught, 'Не удалось создать форму.') }
  finally { saving.value = false }
}

async function toggle(source: unknown, enabled: boolean): Promise<void> {
	const row = source as FormRecord
  const siteID = selected.selectedSite.value?.id
  if (!siteID) return
  try {
    const updated = await setFormEnabled(props.accessToken, siteID, row.id, enabled)
    items.value = items.value.map((item) => item.id === row.id ? updated : item)
    ElMessage.success(enabled ? 'Форма включена' : 'Форма выключена')
  } catch (caught) { handleError(caught, 'Не удалось изменить состояние формы.') }
}
async function remove(source: unknown): Promise<void> {
	const row = source as FormRecord
  const siteID = selected.selectedSite.value?.id
  if (!siteID) return
  try { await ElMessageBox.confirm(`Удалить форму «${row.name}» вместе с результатами? Это действие необратимо.`, 'Удаление формы', { confirmButtonText: 'Удалить', cancelButtonText: 'Отмена', type: 'warning' }) }
  catch { return }
  try { await deleteForm(props.accessToken, siteID, row.id); ElMessage.success('Форма удалена'); await load() }
  catch (caught) { handleError(caught, 'Не удалось удалить форму.') }
}
function applySearch(): void { page.value = 1; void load() }
watch(() => selected.selectedSite.value?.id, () => { page.value = 1; void load() })
onMounted(() => void load())
</script>

<template>
  <access-denied-view v-if="!permissions.has('forms.form.read')" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page forms-list-page">
    <header class="page-header"><div><h1>Формы</h1><p>Конструктор публичных форм и обработка результатов</p></div>
      <el-button v-if="permissions.has('forms.form.create')" type="primary" :icon="Plus" @click="openCreate">Создать форму</el-button>
    </header>
    <el-alert v-if="!selected.selectedSite.value" type="warning" :closable="false" title="Выберите сайт в боковой панели." />
    <el-alert v-else-if="error" type="error" :closable="false" :title="error" show-icon />
    <div v-if="selected.selectedSite.value" class="forms-list-tools"><el-input v-model="search" clearable placeholder="Название или код" @keyup.enter="applySearch" /><el-button @click="applySearch">Найти</el-button></div>
    <el-table v-if="selected.selectedSite.value" v-loading="loading" :data="items" stripe empty-text="Форм пока нет">
      <el-table-column prop="name" label="Название" min-width="240" />
      <el-table-column prop="code" label="Код" min-width="180"><template #default="{ row }"><code>{{ row.code }}</code></template></el-table-column>
      <el-table-column label="Состояние" width="180"><template #default="{ row }">
        <el-switch v-if="permissions.has('forms.form.update')" :model-value="row.enabled" inline-prompt active-text="Включена" inactive-text="Выключена" @update:model-value="toggle(row, Boolean($event))" />
        <el-tag v-else :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? 'Включена' : 'Выключена' }}</el-tag>
      </template></el-table-column>
      <el-table-column label="Изменена" width="180"><template #default="{ row }">{{ new Date(row.updated_at).toLocaleString() }}</template></el-table-column>
      <el-table-column label="Действия" width="210" align="right"><template #default="{ row }">
        <el-button text type="primary" @click="router.push({ name: 'forms.edit', params: { formId: row.id } })">Открыть</el-button>
        <el-button v-if="permissions.has('forms.form.delete')" text type="danger" @click="remove(row)">Удалить</el-button>
      </template></el-table-column>
    </el-table>
    <el-pagination v-if="total > perPage" background layout="total, prev, pager, next" :current-page="page" :page-size="perPage" :total="total" @current-change="page = $event; load()" />

    <el-dialog v-model="createOpen" title="Новая форма" width="min(560px, 94vw)" destroy-on-close>
      <el-form label-position="top" @submit.prevent="submitCreate">
        <el-form-item label="Название" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="Код" required><el-input v-model="form.code" placeholder="feedback" /></el-form-item>
        <el-form-item label="Описание"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="Принимать отправки"><el-switch v-model="form.enabled" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="createOpen = false">Отмена</el-button><el-button type="primary" :loading="saving" @click="submitCreate">Создать</el-button></template>
    </el-dialog>
  </section>
</template>

<style scoped>.forms-list-page{display:grid;gap:16px}.forms-list-tools{display:flex;gap:8px;max-width:560px}.forms-list-tools .el-input{flex:1}</style>
