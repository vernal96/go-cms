<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ArrowDown, ArrowUp, Plus } from '@element-plus/icons-vue'
import { ElAlert, ElButton, ElCard, ElForm, ElFormItem, ElInput, ElMessage, ElMessageBox, ElOption, ElSelect, ElSwitch, ElTabPane, ElTable, ElTableColumn, ElTabs, ElTag } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { AdminAPIError } from '../../api/admin-api'
import AccessDeniedView from '../../components/AccessDeniedView.vue'
import { useSelectedSite } from '../../composables/use-selected-site'
import {
  createAction, createContainer, createElement, createField, createStatus, deleteAction, deleteElement, deleteField,
  deleteStatus, getFormEditor, replaceLayout, updateAction, updateElement, updateField, updateForm, updateStatus,
} from './api'
import FormActionDialog from './FormActionDialog.vue'
import FormElementDialog from './FormElementDialog.vue'
import FormFieldDialog from './FormFieldDialog.vue'
import FormStatusDialog from './FormStatusDialog.vue'
import type { FormAction, FormEditorResponse, FormElement, FormField, FormFieldPayload, FormPayload, FormStatus, LayoutNode } from './types'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const route = useRoute(); const router = useRouter(); const selected = useSelectedSite()
const formID = computed(() => Number(route.params.formId))
const detail = ref<FormEditorResponse | null>(null)
const loading = ref(false); const saving = ref(false); const error = ref<string | null>(null); const tab = ref('fields')
const form = reactive<FormPayload>({ code: '', name: '', description: '', enabled: false })
const fieldOpen = ref(false); const currentField = ref<FormField | null>(null)
const elementOpen = ref(false); const currentElement = ref<FormElement | null>(null)
const statusOpen = ref(false); const currentStatus = ref<FormStatus | null>(null)
const actionOpen = ref(false); const currentAction = ref<FormAction | null>(null)
const layoutDraft = ref<LayoutNode[]>([])

const canUpdate = computed(() => props.permissions.has('forms.form.update'))
const sortedFields = computed(() => [...(detail.value?.fields ?? [])].sort((a, b) => a.result_position - b.result_position || a.id - b.id))
const sortedStatuses = computed(() => [...(detail.value?.statuses ?? [])].sort((a, b) => a.position - b.position || a.id - b.id))
const sortedActions = computed(() => [...(detail.value?.actions ?? [])].sort((a, b) => a.position - b.position || a.id - b.id))
const containers = computed(() => layoutDraft.value.filter((item) => item.kind === 'container'))
const layoutRows = computed(() => {
  const children = new Map<number, LayoutNode[]>()
  for (const node of layoutDraft.value) { const parent = node.parent_id ?? 0; children.set(parent, [...(children.get(parent) ?? []), node]) }
  for (const items of children.values()) items.sort((a, b) => a.position - b.position || a.id - b.id)
  const result: Array<LayoutNode & { depth: number }> = []
  const visit = (parent: number, depth: number): void => { for (const item of children.get(parent) ?? []) { result.push({ ...item, depth }); if (item.kind === 'container') visit(item.id, depth + 1) } }
  visit(0, 0); return result
})

function handleError(caught: unknown, fallback: string): void {
  if (caught instanceof AdminAPIError && caught.status === 401) { emit('unauthorized'); return }
  error.value = caught instanceof Error ? caught.message : fallback
}
async function load(): Promise<void> {
  const siteID = selected.selectedSite.value?.id
  if (!siteID || !formID.value || !props.permissions.has('forms.form.read')) return
  loading.value = true; error.value = null
  try {
    detail.value = await getFormEditor(props.accessToken, siteID, formID.value)
    Object.assign(form, { code: detail.value.form.code, name: detail.value.form.name, description: detail.value.form.description, enabled: detail.value.form.enabled })
    layoutDraft.value = detail.value.layout.map((item) => ({ ...item, config: { ...(item.config ?? {}) } }))
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
function editField(source?: unknown): void { currentField.value = source ? source as FormField : null; fieldOpen.value = true }
async function saveField(payload: FormFieldPayload): Promise<void> {
  const siteID = selected.selectedSite.value?.id; if (!siteID) return
  try { if (currentField.value) await updateField(props.accessToken, siteID, formID.value, currentField.value.id, payload); else await createField(props.accessToken, siteID, formID.value, payload); fieldOpen.value = false; ElMessage.success('Поле сохранено'); await load() }
  catch (caught) { handleError(caught, 'Не удалось сохранить поле.') }
}
async function removeField(source: unknown): Promise<void> { const item = source as FormField; const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { await ElMessageBox.confirm(`Удалить поле «${item.label}»?`, 'Удаление поля', { type: 'warning' }); await deleteField(props.accessToken, siteID, formID.value, item.id); await load() } catch (caught) { if (caught instanceof Error) handleError(caught, 'Не удалось удалить поле.') } }
function editElement(source?: unknown): void { currentElement.value = source ? source as FormElement : null; elementOpen.value = true }
async function saveElement(payload: Pick<FormElement, 'code' | 'type' | 'config'>): Promise<void> { const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { if (currentElement.value) await updateElement(props.accessToken, siteID, formID.value, currentElement.value.id, payload); else await createElement(props.accessToken, siteID, formID.value, payload); elementOpen.value = false; ElMessage.success('Элемент сохранён'); await load() } catch (caught) { handleError(caught, 'Не удалось сохранить элемент.') } }
async function removeElement(source: unknown): Promise<void> { const item = source as FormElement; const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { await ElMessageBox.confirm(`Удалить элемент «${item.code}»?`, 'Удаление элемента', { type: 'warning' }); await deleteElement(props.accessToken, siteID, formID.value, item.id); await load() } catch (caught) { if (caught instanceof Error) handleError(caught, 'Не удалось удалить элемент.') } }
function editStatus(source?: unknown): void { currentStatus.value = source ? source as FormStatus : null; statusOpen.value = true }
async function saveStatus(payload: Pick<FormStatus, 'code' | 'name' | 'color' | 'position' | 'is_default'>): Promise<void> { const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { if (currentStatus.value) await updateStatus(props.accessToken, siteID, formID.value, currentStatus.value.id, payload); else await createStatus(props.accessToken, siteID, formID.value, payload); statusOpen.value = false; ElMessage.success('Статус сохранён'); await load() } catch (caught) { handleError(caught, 'Не удалось сохранить статус.') } }
async function removeStatus(source: unknown): Promise<void> { const item = source as FormStatus; const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { await ElMessageBox.confirm(`Удалить статус «${item.name}»?`, 'Удаление статуса', { type: 'warning' }); await deleteStatus(props.accessToken, siteID, formID.value, item.id); await load() } catch (caught) { if (caught instanceof Error) handleError(caught, 'Не удалось удалить статус.') } }
function editAction(source?: unknown): void { currentAction.value = source ? source as FormAction : null; actionOpen.value = true }
async function saveAction(payload: Pick<FormAction, 'code' | 'name' | 'enabled' | 'trigger' | 'action_type' | 'config' | 'position'>): Promise<void> { const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { if (currentAction.value) await updateAction(props.accessToken, siteID, formID.value, currentAction.value.id, payload); else await createAction(props.accessToken, siteID, formID.value, payload); actionOpen.value = false; ElMessage.success('Действие сохранено'); await load() } catch (caught) { handleError(caught, 'Не удалось сохранить действие.') } }
async function removeAction(source: unknown): Promise<void> { const item = source as FormAction; const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { await ElMessageBox.confirm(`Удалить действие «${item.name}»? История исполнений сохранится.`, 'Удаление действия', { type: 'warning' }); await deleteAction(props.accessToken, siteID, formID.value, item.id); await load() } catch (caught) { if (caught instanceof Error) handleError(caught, 'Не удалось удалить действие.') } }

function nodeLabel(source: unknown): string {
	const node = source as LayoutNode
  if (node.kind === 'field') { const item = detail.value?.fields.find((field) => field.id === node.field_id); return item ? `Поле: ${item.label} (${item.code})` : 'Неизвестное поле' }
  if (node.kind === 'element') { const item = detail.value?.elements.find((element) => element.id === node.element_id); return item ? `Элемент: ${item.code}` : 'Неизвестный элемент' }
  return node.container_type === 'slide' ? 'Слайд' : 'Группа'
}
function descendants(nodeID: number): Set<number> { const result = new Set<number>(); const visit = (id: number): void => { for (const child of layoutDraft.value.filter((item) => item.parent_id === id)) { result.add(child.id); visit(child.id) } }; visit(nodeID); return result }
function allowedParents(source: unknown): LayoutNode[] { const node = source as LayoutNode; const denied = descendants(node.id); denied.add(node.id); return containers.value.filter((item) => !denied.has(item.id)) }
function normalizeSiblings(parentID?: number | null): void { layoutDraft.value.filter((item) => (item.parent_id ?? null) === (parentID ?? null)).sort((a, b) => a.position - b.position || a.id - b.id).forEach((item, index) => { item.position = index }) }
function changeParent(node: LayoutNode, value: number | null): void { const original = node.parent_id ?? null; node.parent_id = value || null; normalizeSiblings(original); normalizeSiblings(node.parent_id) }
function move(node: LayoutNode, direction: -1 | 1): void { const siblings = layoutDraft.value.filter((item) => (item.parent_id ?? null) === (node.parent_id ?? null)).sort((a, b) => a.position - b.position || a.id - b.id); const index = siblings.findIndex((item) => item.id === node.id); const target = siblings[index + direction]; if (!target) return; const position = node.position; node.position = target.position; target.position = position }
async function addContainer(type: 'group' | 'slide'): Promise<void> { const siteID = selected.selectedSite.value?.id; if (!siteID) return; try { await createContainer(props.accessToken, siteID, formID.value, { container_type: type, position: layoutDraft.value.filter((item) => !item.parent_id).length, config: {} }); await load() } catch (caught) { handleError(caught, 'Не удалось создать контейнер.') } }
async function saveLayout(): Promise<void> { const siteID = selected.selectedSite.value?.id; if (!siteID) return; saving.value = true; try { for (const parent of [null, ...containers.value.map((item) => item.id)]) normalizeSiblings(parent); const response = await replaceLayout(props.accessToken, siteID, formID.value, layoutDraft.value); layoutDraft.value = response.nodes; ElMessage.success('Порядок сохранён') } catch (caught) { handleError(caught, 'Не удалось сохранить структуру.') } finally { saving.value = false } }

watch(() => selected.selectedSite.value?.id, () => void load())
onMounted(() => void load())
</script>

<template>
  <access-denied-view v-if="!permissions.has('forms.form.read')" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page form-builder-page" v-loading="loading">
    <header class="page-header"><div><h1>{{ detail?.form.name || 'Форма' }}</h1><p v-if="detail"><code>{{ detail.form.code }}</code> · поля, структура, статусы и асинхронные действия</p></div><el-button @click="router.push({ name: 'forms.list' })">К формам</el-button></header>
    <el-alert v-if="error" type="error" :closable="false" :title="error" show-icon />
    <template v-if="detail">
      <el-card><el-form label-position="top" class="form-settings"><el-form-item label="Название"><el-input v-model="form.name" /></el-form-item><el-form-item label="Код"><el-input v-model="form.code" /></el-form-item><el-form-item label="Описание"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item><el-form-item label="Принимать отправки"><el-switch v-model="form.enabled" /></el-form-item></el-form><el-button v-if="canUpdate" type="primary" :loading="saving" @click="saveForm">Сохранить настройки</el-button></el-card>
      <el-tabs v-model="tab" class="builder-tabs">
        <el-tab-pane label="Поля" name="fields"><div class="section-tools"><h2>Поля</h2><el-button v-if="canUpdate" :icon="Plus" @click="editField()">Добавить поле</el-button></div><el-table :data="sortedFields" stripe><el-table-column prop="label" label="Подпись" min-width="220" /><el-table-column prop="code" label="Код" min-width="160" /><el-table-column prop="type" label="Тип" min-width="150" /><el-table-column label="Свойства" min-width="200"><template #default="{ row }"><el-tag v-if="row.required" type="warning">Обязательное</el-tag> <el-tag v-if="row.show_in_results" type="success">В таблице: {{ row.result_label || row.label }}</el-tag> <el-tag v-if="row.code === 'privacy_consent' || row.code === 'captcha'">Системное</el-tag></template></el-table-column><el-table-column label="Действия" width="180" align="right"><template #default="{ row }"><el-button v-if="canUpdate" text type="primary" @click="editField(row)">Изменить</el-button><el-button v-if="canUpdate && row.code !== 'privacy_consent' && row.code !== 'captcha'" text type="danger" @click="removeField(row)">Удалить</el-button></template></el-table-column></el-table></el-tab-pane>
        <el-tab-pane label="Элементы" name="elements"><div class="section-tools"><h2>Элементы</h2><el-button v-if="canUpdate" :icon="Plus" @click="editElement()">Добавить элемент</el-button></div><el-table :data="detail.elements" stripe><el-table-column prop="code" label="Код" /><el-table-column prop="type" label="Тип" /><el-table-column label="Свойства"><template #default="{ row }"><el-tag v-if="row.type === 'submit_button'">Обязательная кнопка</el-tag><span v-else>{{ row.type === 'image' ? `Файл #${row.config.file_id}` : (row.config.text || row.config.content || '—') }}</span></template></el-table-column><el-table-column label="Действия" width="180" align="right"><template #default="{ row }"><el-button v-if="canUpdate" text type="primary" @click="editElement(row)">Изменить</el-button><el-button v-if="canUpdate && row.type !== 'submit_button'" text type="danger" @click="removeElement(row)">Удалить</el-button></template></el-table-column></el-table></el-tab-pane>
        <el-tab-pane label="Структура" name="layout"><div class="section-tools"><div><h2>Структура</h2><p>Корень, группы и слайды сохраняют порядок для будущего многошагового рендера.</p></div><span><el-button v-if="canUpdate" @click="addContainer('group')">Добавить группу</el-button> <el-button v-if="canUpdate" @click="addContainer('slide')">Добавить слайд</el-button></span></div><el-table :data="layoutRows" row-key="id" stripe><el-table-column label="Узел" min-width="280"><template #default="{ row }"><span :style="{ paddingLeft: `${row.depth * 24}px` }">{{ nodeLabel(row) }}</span></template></el-table-column><el-table-column label="Родитель" min-width="220"><template #default="{ row }"><el-select :model-value="row.parent_id ?? 0" :disabled="!canUpdate" @update:model-value="changeParent(layoutDraft.find(item => item.id === row.id)!, $event)"><el-option :value="0" label="Корень" /><el-option v-for="parent in allowedParents(row)" :key="parent.id" :value="parent.id" :label="nodeLabel(parent)" /></el-select></template></el-table-column><el-table-column label="Порядок" width="150"><template #default="{ row }"><el-button :icon="ArrowUp" circle plain aria-label="Выше" :disabled="!canUpdate" @click="move(layoutDraft.find(item => item.id === row.id)!, -1)" /><el-button :icon="ArrowDown" circle plain aria-label="Ниже" :disabled="!canUpdate" @click="move(layoutDraft.find(item => item.id === row.id)!, 1)" /></template></el-table-column></el-table><el-button v-if="canUpdate" type="primary" :loading="saving" @click="saveLayout">Сохранить структуру</el-button></el-tab-pane>
        <el-tab-pane label="Статусы" name="statuses"><div class="section-tools"><h2>Статусы результатов</h2><el-button v-if="permissions.has('forms.status.create')" :icon="Plus" @click="editStatus()">Добавить статус</el-button></div><el-table :data="sortedStatuses" stripe><el-table-column prop="name" label="Название" /><el-table-column prop="code" label="Код" /><el-table-column label="Цвет"><template #default="{ row }"><span class="status-dot" :style="{ background: row.color }" />{{ row.color }}</template></el-table-column><el-table-column label="По умолчанию"><template #default="{ row }"><el-tag v-if="row.is_default" type="success">По умолчанию</el-tag></template></el-table-column><el-table-column label="Действия" width="180" align="right"><template #default="{ row }"><el-button v-if="permissions.has('forms.status.update')" text type="primary" @click="editStatus(row)">Изменить</el-button><el-button v-if="permissions.has('forms.status.delete') && !row.is_default" text type="danger" @click="removeStatus(row)">Удалить</el-button></template></el-table-column></el-table></el-tab-pane>
        <el-tab-pane label="Действия" name="actions"><div class="section-tools"><div><h2>Асинхронные действия</h2><p>Результат всегда сохраняется; действия выполняются через очередь после commit.</p></div><el-button v-if="permissions.has('forms.action.create')" :icon="Plus" @click="editAction()">Добавить действие</el-button></div><el-table :data="sortedActions" stripe><el-table-column prop="name" label="Название" /><el-table-column prop="code" label="Код" /><el-table-column prop="action_type" label="Тип" /><el-table-column label="Событие"><template #default="{ row }">{{ row.trigger.type === 'submitted' ? 'Отправка' : `Смена статуса ${row.trigger.from_status || '*'} → ${row.trigger.to_status || '*'}` }}</template></el-table-column><el-table-column label="Состояние"><template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? 'Включено' : 'Выключено' }}</el-tag></template></el-table-column><el-table-column label="Действия" width="180" align="right"><template #default="{ row }"><el-button v-if="permissions.has('forms.action.update')" text type="primary" @click="editAction(row)">Изменить</el-button><el-button v-if="permissions.has('forms.action.delete')" text type="danger" @click="removeAction(row)">Удалить</el-button></template></el-table-column></el-table></el-tab-pane>
      </el-tabs>
      <form-field-dialog v-model="fieldOpen" :field="currentField" :fields="detail.fields" :available-types="detail.available_field_types" @save="saveField" />
      <form-element-dialog v-model="elementOpen" :element="currentElement" :available-types="detail.available_element_types" :access-token="accessToken" :permissions="permissions" @save="saveElement" />
      <form-status-dialog v-model="statusOpen" :status="currentStatus" :next-position="detail.statuses.length" @save="saveStatus" />
      <form-action-dialog v-model="actionOpen" :action="currentAction" :action-types="detail.available_action_types" :fields="detail.fields" :statuses="detail.statuses" :access-token="accessToken" :site-i-d="selected.selectedSite.value?.id ?? 0" :permissions="permissions" :next-position="detail.actions.length" @save="saveAction" />
    </template>
  </section>
</template>

<style scoped>.form-builder-page{display:grid;gap:16px}.form-settings{display:grid;grid-template-columns:1fr 1fr;gap:0 16px}.builder-tabs{padding:0 16px 16px;border:1px solid var(--el-border-color);border-radius:8px}.section-tools{display:flex;align-items:center;justify-content:space-between;gap:16px;margin-bottom:12px}.section-tools h2,.section-tools p{margin:0}.section-tools p{color:var(--el-text-color-secondary)}.builder-tabs :deep(.el-select){width:100%}.status-dot{display:inline-block;width:14px;height:14px;border-radius:50%;margin-right:8px;vertical-align:middle}@media(max-width:760px){.form-settings{grid-template-columns:1fr}.section-tools{align-items:flex-start;flex-direction:column}}</style>
