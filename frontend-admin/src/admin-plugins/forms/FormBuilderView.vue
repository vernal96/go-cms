<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElAlert, ElButton, ElCard, ElForm, ElFormItem, ElInput, ElMessage, ElMessageBox, ElSwitch, ElTabPane, ElTable, ElTableColumn, ElTabs, ElTag } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { AdminAPIError } from '../../api/admin-api'
import AccessDeniedView from '../../components/AccessDeniedView.vue'
import { useSelectedSite } from '../../composables/use-selected-site'
import {
  createAction, createStatus, deleteAction,
  deleteStatus, getFormEditor, updateAction, updateForm, updateStatus,
} from './api'
import FormActionDialog from './FormActionDialog.vue'
import FormStructureEditor from './FormStructureEditor.vue'
import FormStatusDialog from './FormStatusDialog.vue'
import type { FormAction, FormEditorResponse, FormPayload, FormStatus } from './types'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const route = useRoute(); const router = useRouter(); const selected = useSelectedSite()
const formID = computed(() => Number(route.params.formId))
const detail = ref<FormEditorResponse | null>(null)
const loading = ref(false); const saving = ref(false); const error = ref<string | null>(null); const tab = ref('layout')
const form = reactive<FormPayload>({ code: '', name: '', description: '', enabled: false })
const structureEditor = ref<{ ensureLeave: () => Promise<boolean> }>()
const statusOpen = ref(false); const currentStatus = ref<FormStatus | null>(null)
const actionOpen = ref(false); const currentAction = ref<FormAction | null>(null)

const canUpdate = computed(() => props.permissions.has('forms.form.update'))
const sortedStatuses = computed(() => [...(detail.value?.statuses ?? [])].sort((a, b) => a.position - b.position || a.id - b.id))
const sortedActions = computed(() => [...(detail.value?.actions ?? [])].sort((a, b) => a.position - b.position || a.id - b.id))
function handleError(caught: unknown, fallback: string): void {
  if (caught instanceof AdminAPIError && caught.status === 401) { emit('unauthorized'); return }
  error.value = caught instanceof Error ? caught.message : fallback
}
async function load(resetSettings = true): Promise<void> {
  const siteID = selected.selectedSite.value?.id
  if (!siteID || !formID.value || !props.permissions.has('forms.form.read')) return
  loading.value = true; error.value = null
  try {
    detail.value = await getFormEditor(props.accessToken, siteID, formID.value)
    if (resetSettings) Object.assign(form, { code: detail.value.form.code, name: detail.value.form.name, description: detail.value.form.description, enabled: detail.value.form.enabled })
  } catch (caught) { handleError(caught, 'Не удалось открыть форму.') }
  finally { loading.value = false }
}
async function saveForm(): Promise<void> {
  const siteID = selected.selectedSite.value?.id; if (!siteID) return
  saving.value = true
  try { const updated = await updateForm(props.accessToken, siteID, formID.value, { ...form, code: form.code.trim(), name: form.name.trim(), description: form.description.trim() }); if (detail.value) detail.value.form = updated; ElMessage.success('Настройки формы сохранены') }
  catch (caught) { handleError(caught, 'Не удалось сохранить форму.') }
  finally { saving.value = false }
}
function editStatus(source?: unknown): void { currentStatus.value = source ? source as FormStatus : null; statusOpen.value = true }
async function saveStatus(payload: Pick<FormStatus, 'code' | 'name' | 'color' | 'position' | 'is_default'>): Promise<void> { const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { if (currentStatus.value) await updateStatus(props.accessToken, siteID, formID.value, currentStatus.value.id, payload); else await createStatus(props.accessToken, siteID, formID.value, payload); statusOpen.value = false; ElMessage.success('Статус сохранён'); await load(false) } catch (caught) { handleError(caught, 'Не удалось сохранить статус.') } }
async function removeStatus(source: unknown): Promise<void> { const item = source as FormStatus; const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { await ElMessageBox.confirm(`Удалить статус «${item.name}»?`, 'Удаление статуса', { type: 'warning' }); await deleteStatus(props.accessToken, siteID, formID.value, item.id); await load(false) } catch (caught) { if (caught instanceof Error) handleError(caught, 'Не удалось удалить статус.') } }
function editAction(source?: unknown): void { currentAction.value = source ? source as FormAction : null; actionOpen.value = true }
async function saveAction(payload: Pick<FormAction, 'code' | 'name' | 'enabled' | 'trigger' | 'action_type' | 'config' | 'position'>): Promise<void> { const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { if (currentAction.value) await updateAction(props.accessToken, siteID, formID.value, currentAction.value.id, payload); else await createAction(props.accessToken, siteID, formID.value, payload); actionOpen.value = false; ElMessage.success('Действие сохранено'); await load(false) } catch (caught) { handleError(caught, 'Не удалось сохранить действие.') } }
async function removeAction(source: unknown): Promise<void> { const item = source as FormAction; const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { await ElMessageBox.confirm(`Удалить действие «${item.name}»? История исполнений сохранится.`, 'Удаление действия', { type: 'warning' }); await deleteAction(props.accessToken, siteID, formID.value, item.id); await load(false) } catch (caught) { if (caught instanceof Error) handleError(caught, 'Не удалось удалить действие.') } }

async function beforeTabLeave(): Promise<boolean> { return await structureEditor.value?.ensureLeave() ?? true }
let restoringSite = false
watch(() => selected.selectedSite.value, async (_site, previous) => {
  if (restoringSite) { restoringSite = false; return }
  if (!await beforeTabLeave()) { restoringSite = true; selected.setSelected(previous); return }
  detail.value = null
  await load()
})
watch(formID, () => { detail.value = null; void load() })
onMounted(() => void load())
</script>

<template>
  <access-denied-view v-if="!permissions.has('forms.form.read')" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page forms-mail-page form-builder-page" v-loading="loading">
    <header class="page-header"><div><h1>{{ detail?.form.name || 'Форма' }}</h1><p v-if="detail"><code>{{ detail.form.code }}</code> · поля, структура, статусы и асинхронные действия</p></div><el-button @click="router.push({ name: 'forms.list' })">К формам</el-button></header>
    <el-alert v-if="error" type="error" :closable="false" :title="error" show-icon />
    <template v-if="detail">
      <el-card><el-form label-position="top" class="form-settings" :disabled="!canUpdate"><el-form-item label="Название"><el-input v-model="form.name" /></el-form-item><el-form-item label="Код"><el-input v-model="form.code" /></el-form-item><el-form-item label="Описание"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item><el-form-item label="Принимать отправки"><el-switch v-model="form.enabled" /></el-form-item></el-form><el-button v-if="canUpdate" type="primary" :loading="saving" @click="saveForm">Сохранить настройки</el-button></el-card>
      <el-tabs v-model="tab" class="builder-tabs" :before-leave="beforeTabLeave">
        <el-tab-pane label="Структура" name="layout"><form-structure-editor ref="structureEditor" :key="`${detail.form.site_id}:${detail.form.id}`" :detail="detail" :access-token="accessToken" :permissions="permissions" @changed="detail = $event" @unauthorized="emit('unauthorized')" /></el-tab-pane>
        <el-tab-pane label="Статусы" name="statuses"><div class="section-tools"><h2>Статусы результатов</h2><el-button v-if="permissions.has('forms.status.create')" :icon="Plus" @click="editStatus()">Добавить статус</el-button></div><el-table :data="sortedStatuses" stripe><el-table-column prop="name" label="Название" /><el-table-column prop="code" label="Код" /><el-table-column label="Цвет"><template #default="{ row }"><span class="status-dot" :style="{ background: row.color }" />{{ row.color }}</template></el-table-column><el-table-column label="По умолчанию"><template #default="{ row }"><el-tag v-if="row.is_default" type="success">По умолчанию</el-tag></template></el-table-column><el-table-column label="Действия" width="180" align="right"><template #default="{ row }"><el-button v-if="permissions.has('forms.status.update')" text type="primary" @click="editStatus(row)">Изменить</el-button><el-button v-if="permissions.has('forms.status.delete') && !row.is_default" text type="danger" @click="removeStatus(row)">Удалить</el-button></template></el-table-column></el-table></el-tab-pane>
        <el-tab-pane label="Действия" name="actions"><div class="section-tools"><div><h2>Асинхронные действия</h2><p>Результат всегда сохраняется; действия выполняются в фоновом режиме.</p></div><el-button v-if="permissions.has('forms.action.create')" :icon="Plus" @click="editAction()">Добавить действие</el-button></div><el-table :data="sortedActions" stripe><el-table-column prop="name" label="Название" /><el-table-column prop="code" label="Код" /><el-table-column prop="action_type" label="Тип" /><el-table-column label="Событие"><template #default="{ row }">{{ row.trigger.type === 'submitted' ? 'Отправка' : `Смена статуса ${row.trigger.from_status || '*'} → ${row.trigger.to_status || '*'}` }}</template></el-table-column><el-table-column label="Состояние"><template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? 'Включено' : 'Выключено' }}</el-tag></template></el-table-column><el-table-column label="Действия" width="180" align="right"><template #default="{ row }"><el-button v-if="permissions.has('forms.action.update')" text type="primary" @click="editAction(row)">Изменить</el-button><el-button v-if="permissions.has('forms.action.delete')" text type="danger" @click="removeAction(row)">Удалить</el-button></template></el-table-column></el-table></el-tab-pane>
      </el-tabs>
      <form-status-dialog v-model="statusOpen" :status="currentStatus" :next-position="detail.statuses.length" @save="saveStatus" />
      <form-action-dialog v-model="actionOpen" :action="currentAction" :action-types="detail.available_action_types" :fields="detail.fields" :statuses="detail.statuses" :access-token="accessToken" :site-i-d="selected.selectedSite.value?.id ?? 0" :permissions="permissions" :next-position="detail.actions.length" @save="saveAction" />
    </template>
  </section>
</template>

<style scoped>.form-builder-page{display:grid;gap:16px}.form-settings{display:grid;grid-template-columns:1fr 1fr;gap:0 16px}.builder-tabs{padding:0 16px 16px;border:1px solid var(--el-border-color);border-radius:8px}.section-tools{display:flex;align-items:center;justify-content:space-between;gap:16px;margin-bottom:12px}.section-tools h2,.section-tools p{margin:0}.section-tools p{color:var(--el-text-color-secondary)}.builder-tabs :deep(.el-select){width:100%}.status-dot{display:inline-block;width:14px;height:14px;border-radius:50%;margin-right:8px;vertical-align:middle}@media(max-width:760px){.form-settings{grid-template-columns:1fr}.section-tools{align-items:flex-start;flex-direction:column}}</style>
