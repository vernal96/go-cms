<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { onBeforeRouteLeave, onBeforeRouteUpdate } from 'vue-router'
import { Plus, MoreFilled } from '@element-plus/icons-vue'
import { ElAlert, ElButton, ElDialog, ElDropdown, ElDropdownItem, ElDropdownMenu, ElEmpty, ElForm, ElFormItem, ElInput, ElMessage, ElMessageBox, ElOption, ElSelect, ElTag, ElTree } from 'element-plus'
import type { AllowDropType, NodeDropType } from 'element-plus/es/components/tree/src/tree.type'
import type TreeNode from 'element-plus/es/components/tree/src/model/node'
import { AdminAPIError } from '../../api/admin-api'
import { createContainer, createElement, createField, deleteContainer, deleteElement, deleteField, getFormEditor, replaceLayout, updateElement, updateField } from './api'
import FormFieldEditor from './FormFieldEditor.vue'
import FormElementEditor from './FormElementEditor.vue'
import { canPlace, dropNode, moveNode, siblings, treeNodes } from './layout-tree'
import type { ContainerType, ElementType, FormEditorResponse, FormElement, FormField, FormFieldPayload, FormsFieldType, LayoutKind, LayoutNode } from './types'

function clone<T>(value: T): T { return JSON.parse(JSON.stringify(value)) as T }

const props = defineProps<{ detail: FormEditorResponse; accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ changed: [detail: FormEditorResponse]; unauthorized: [] }>()
const canUpdate = computed(() => props.permissions.has('forms.form.update'))
const nodes = ref<LayoutNode[]>([])
const expanded = ref<number[]>([])
const tree = ref<InstanceType<typeof ElTree>>()
const treeKey = ref(0)
const selectedID = ref<number | null>(null)
const creating = ref(false)
const parentID = ref<number | null>(null)
const kind = ref<LayoutKind | ''>('')
const type = ref('')
const label = ref('')
const initialLabel = ref('')
const field = ref<FormField | null>(null)
const element = ref<FormElement | null>(null)
const fieldEditor = ref<{ payload: () => FormFieldPayload }>()
const elementEditor = ref<{ payload: () => Pick<FormElement, 'code' | 'type' | 'config'> }>()
const editorKey = ref(0)
const editorDirty = ref(false)
const busy = ref(false)
const error = ref('')
const unsynced = ref(false)
const leaveOpen = ref(false)
let leaveResolve: ((answer: boolean) => void) | undefined
const dirty = computed(() => canUpdate.value && (creating.value || editorDirty.value || label.value !== initialLabel.value))
const selectedNode = computed(() => nodes.value.find(node => node.id === selectedID.value))
const data = computed(() => treeNodes(nodes.value))
const typeOptions = computed(() => {
  if (kind.value === 'field') return props.detail.available_field_types.map(code => ({ code, label: fieldTypeLabels[code] ?? code }))
  if (kind.value === 'element') return props.detail.available_element_types.filter(item => item.code !== 'submit_button').map(item => ({ code: item.code, label: item.label }))
  if (kind.value === 'container') return props.detail.available_container_types
  return []
})
const fieldTypeLabels: Record<string, string> = { string: 'Строка', integer: 'Целое число', float: 'Число', checkbox: 'Флаг', radio: 'Один вариант', select: 'Список', textarea: 'Многострочный текст', email: 'Email', phone: 'Телефон', json: 'JSON', 'forms.captcha': 'CAPTCHA', 'forms.consent': 'Согласие', 'forms.upload': 'Загрузка файлов' }
const parentOptions = computed(() => nodes.value.filter(node => node.kind === 'container' && selectedID.value !== null && canPlace(nodes.value, selectedID.value, node.id)))
watch(() => props.detail.layout, value => { nodes.value = clone(value) }, { immediate: true })

function nodeLabel(node: LayoutNode): string {
  if (node.kind === 'field') return props.detail.fields.find(item => item.id === node.field_id)?.label ?? 'Поле'
  if (node.kind === 'element') { const item = props.detail.elements.find(item => item.id === node.element_id); return String(item?.config.label || item?.config.text || item?.code || 'Элемент') }
  return String(node.config?.label || props.detail.available_container_types.find(item => item.code === node.container_type)?.label || node.container_type)
}
function nodeType(node: LayoutNode): string {
  if (node.kind === 'field') { const code = props.detail.fields.find(item => item.id === node.field_id)?.type ?? ''; return `Поле · ${fieldTypeLabels[code] ?? code}` }
  if (node.kind === 'element') { const code = props.detail.elements.find(item => item.id === node.element_id)?.type; return `Элемент · ${props.detail.available_element_types.find(item => item.code === code)?.label ?? code}` }
  return `Контейнер · ${props.detail.available_container_types.find(item => item.code === node.container_type)?.label ?? node.container_type}`
}
function mandatory(node: LayoutNode): boolean {
  return props.detail.fields.some(item => item.id === node.field_id && ['privacy_consent', 'captcha'].includes(item.code)) || props.detail.elements.some(item => item.id === node.element_id && item.type === 'submit_button')
}
function showNode(id: number | null): void {
  const node = nodes.value.find(item => item.id === id)
  selectedID.value = node?.id ?? null; creating.value = false; kind.value = node?.kind ?? ''; type.value = ''
  field.value = clone(props.detail.fields.find(item => item.id === node?.field_id) ?? null)
  element.value = clone(props.detail.elements.find(item => item.id === node?.element_id) ?? null)
  label.value = initialLabel.value = String(node?.config?.label ?? '')
  editorDirty.value = false; editorKey.value++
  void nextTick(() => tree.value?.setCurrentKey(selectedID.value ?? undefined))
}
function report(caught: unknown): void {
  if (caught instanceof AdminAPIError && caught.status === 401) emit('unauthorized')
  error.value = caught instanceof Error ? caught.message : 'Не удалось сохранить изменения.'
}
async function refresh(): Promise<void> {
  const value = await getFormEditor(props.accessToken, props.detail.form.site_id, props.detail.form.id)
  emit('changed', value)
  await nextTick()
  unsynced.value = false
  treeKey.value++
}
async function selectNode(node: LayoutNode): Promise<void> {
  if (busy.value || node.id === selectedID.value && !creating.value) return
  if (await ensureLeave()) showNode(node.id)
  else tree.value?.setCurrentKey(selectedID.value ?? undefined)
}
async function addNode(parent: number | null): Promise<void> {
  if (!canUpdate.value || busy.value || unsynced.value || !await ensureLeave()) return
  showNode(null); parentID.value = parent; creating.value = true
  if (parent !== null && !expanded.value.includes(parent)) expanded.value.push(parent)
}
function changeKind(): void { type.value = ''; editorKey.value++; editorDirty.value = false }
function changeType(): void { editorKey.value++; editorDirty.value = false }
async function save(): Promise<boolean> {
  if (!canUpdate.value || busy.value || unsynced.value) return false
  error.value = ''; busy.value = true
  let savedID = selectedID.value
  let committed = false
  try {
    const { site_id: site, id: form } = props.detail.form
    const placement = { parent_id: parentID.value, position: siblings(nodes.value, parentID.value).length }
    if (kind.value === 'field') {
      const payload = fieldEditor.value?.payload()
      if (!payload || !payload.code || !payload.label) throw new Error('Заполните код и подпись поля.')
      if (creating.value) savedID = (await createField(props.accessToken, site, form, { ...payload, ...placement })).layout_node.id
      else if (field.value) await updateField(props.accessToken, site, form, field.value.id, payload)
    } else if (kind.value === 'element') {
      const payload = elementEditor.value?.payload()
      if (!payload || !payload.code) throw new Error('Заполните код элемента.')
      if (creating.value) savedID = (await createElement(props.accessToken, site, form, { ...payload, ...placement })).layout_node.id
      else if (element.value) await updateElement(props.accessToken, site, form, element.value.id, payload)
    } else if (kind.value === 'container') {
      if (creating.value) {
        if (!type.value) throw new Error('Выберите тип контейнера.')
        savedID = (await createContainer(props.accessToken, site, form, { ...placement, container_type: type.value as ContainerType, config: { label: label.value.trim() } })).id
      } else {
        await replaceLayout(props.accessToken, site, form, nodes.value.map(node => node.id === selectedID.value ? { ...node, config: { ...node.config, label: label.value.trim() } } : node))
      }
    } else throw new Error('Выберите категорию и тип узла.')
    committed = true
    // Retain the created ID if refreshing fails, preventing duplicate creation on retry.
    selectedID.value = savedID; creating.value = false; editorDirty.value = false; initialLabel.value = label.value
    await refresh(); showNode(savedID); ElMessage.success('Узел сохранён')
    return true
  } catch (caught) { if (committed) unsynced.value = true; report(caught); return false }
  finally { busy.value = false }
}
async function ensureLeave(): Promise<boolean> {
  if (busy.value) return false
  if (!dirty.value) return true
  if (leaveResolve) return false
  leaveOpen.value = true
  return new Promise(resolve => { leaveResolve = resolve })
}
function finishLeave(answer: boolean): void {
  const resolve = leaveResolve; leaveResolve = undefined; leaveOpen.value = false; resolve?.(answer)
}
async function leaveSave(): Promise<void> { if (await save()) finishLeave(true) }
function leaveDiscard(): void { showNode(selectedID.value); finishLeave(true) }
function beforeUnload(event: BeforeUnloadEvent): void { if (dirty.value || busy.value) { event.preventDefault(); event.returnValue = '' } }
onMounted(() => window.addEventListener('beforeunload', beforeUnload))
onBeforeUnmount(() => { window.removeEventListener('beforeunload', beforeUnload); finishLeave(false) })
onBeforeRouteLeave(ensureLeave)
onBeforeRouteUpdate(ensureLeave)

function allowDrop(drag: TreeNode, target: TreeNode, position: AllowDropType): boolean {
  if (!canUpdate.value || busy.value || unsynced.value || drag.data.id === target.data.id) return false
  return canPlace(nodes.value, drag.data.id, position === 'inner' ? target.data.id : target.data.parent_id ?? null)
}
async function persistMove(draft: LayoutNode[]): Promise<void> {
  if (!canUpdate.value || busy.value || unsynced.value) return
  const previous = nodes.value; nodes.value = draft; busy.value = true; error.value = ''
  try {
    const result = await replaceLayout(props.accessToken, props.detail.form.site_id, props.detail.form.id, draft)
    nodes.value = result.nodes
    emit('changed', { ...props.detail, layout: result.nodes })
  } catch (caught) {
    nodes.value = previous; report(caught)
    try { await refresh() } catch (refreshError) { unsynced.value = true; report(refreshError) }
  } finally { busy.value = false; treeKey.value++; await nextTick(); tree.value?.setCurrentKey(selectedID.value ?? undefined) }
}
function onDrop(drag: TreeNode, target: TreeNode, position: NodeDropType): void {
  if (position === 'none') return
  try { void persistMove(dropNode(nodes.value, drag.data.id, target.data.id, position === 'inner' ? 'inner' : position === 'before' ? 'before' : 'after')) }
  catch (caught) { treeKey.value++; report(caught) }
}
async function command(action: string, node: LayoutNode): Promise<void> {
  if (busy.value || unsynced.value || !canUpdate.value) return
  if (action === 'add') { await addNode(node.id); return }
  if (action === 'delete') { await remove(node); return }
  if (action === 'parent') { if (await ensureLeave()) { showNode(node.id); moveParent.value = node.parent_id ?? 0; moveOpen.value = true }; return }
  const children = siblings(nodes.value, node.parent_id ?? null)
  const index = children.findIndex(item => item.id === node.id)
  const next = index + (action === 'up' ? -1 : 1)
  if (next >= 0 && next < children.length) await persistMove(moveNode(nodes.value, node.id, node.parent_id ?? null, next))
}
const moveOpen = ref(false)
const moveParent = ref(0)
async function moveToParent(): Promise<void> {
  const id = selectedID.value; if (id === null) return
  await persistMove(moveNode(nodes.value, id, moveParent.value || null, siblings(nodes.value, moveParent.value || null).filter(item => item.id !== id).length))
  moveOpen.value = false
}
async function remove(node: LayoutNode): Promise<void> {
  if (mandatory(node) || !await ensureLeave()) return
  try { await ElMessageBox.confirm(node.kind === 'container' ? 'Удалить контейнер? Его содержимое займёт его место, порядок сохранится.' : `Удалить «${nodeLabel(node)}»?`, 'Удаление узла', { type: 'warning', confirmButtonText: 'Удалить', cancelButtonText: 'Отмена' }) } catch { return }
  busy.value = true; error.value = ''
  let committed = false
  try {
    const { site_id: site, id: form } = props.detail.form
    if (node.kind === 'container') await deleteContainer(props.accessToken, site, form, node.id)
    else if (node.kind === 'field' && node.field_id) await deleteField(props.accessToken, site, form, node.field_id)
    else if (node.element_id) await deleteElement(props.accessToken, site, form, node.element_id)
    committed = true
    await refresh(); showNode(selectedID.value === node.id ? null : selectedID.value)
  } catch (caught) { if (committed) unsynced.value = true; report(caught) }
  finally { busy.value = false }
}
async function retryRefresh(): Promise<void> {
  busy.value = true
  try { await refresh(); showNode(selectedID.value); error.value = '' } catch (caught) { report(caught) } finally { busy.value = false }
}
defineExpose({ ensureLeave })
</script>

<template>
  <div class="structure-editor">
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
    <el-button v-if="unsynced" :loading="busy" @click="retryRefresh">Обновить структуру</el-button>
    <div class="structure-columns">
      <div class="structure-tree">
        <div class="structure-toolbar"><h2>Структура</h2><el-button v-if="canUpdate" :icon="Plus" :disabled="busy || unsynced" @click="addNode(null)">Добавить узел</el-button></div>
        <p class="structure-hint">{{ canUpdate ? 'Перетаскивайте узлы для изменения порядка и вложенности.' : 'Выберите узел для просмотра настроек.' }}</p>
        <el-tree ref="tree" :key="treeKey" :data="data" node-key="id" :props="{ children: 'children', label: (value: Record<string, unknown>) => nodeLabel(value as unknown as LayoutNode) }" :current-node-key="selectedID ?? undefined" :default-expanded-keys="expanded" :expand-on-click-node="false" highlight-current :draggable="canUpdate && !busy && !unsynced" :allow-drop="allowDrop" :allow-drag="() => canUpdate && !busy && !unsynced" @node-click="selectNode" @node-drop="onDrop" @node-expand="node => { if (!expanded.includes(node.id)) expanded.push(node.id) }" @node-collapse="node => expanded = expanded.filter(id => id !== node.id)">
          <template #default="{ data: node }">
            <div class="structure-node" :data-node-id="node.id">
              <span class="structure-node-text" :title="nodeLabel(node)"><strong>{{ nodeLabel(node) }}</strong><small>{{ nodeType(node) }}</small></span>
              <el-tag v-if="mandatory(node)" size="small" type="info">Обязательный</el-tag>
              <el-button v-if="canUpdate && node.kind === 'container'" :icon="Plus" text :disabled="busy || unsynced" :aria-label="`Добавить потомка: ${nodeLabel(node)}`" @click.stop="addNode(node.id)" />
              <el-dropdown v-if="canUpdate" trigger="click" :disabled="busy || unsynced" @command="value => command(value, node)">
                <el-button text :icon="MoreFilled" :disabled="busy || unsynced" :aria-label="`Действия: ${nodeLabel(node)}`" @click.stop />
                <template #dropdown><el-dropdown-menu>
                  <el-dropdown-item v-if="node.kind === 'container'" command="add">Добавить потомка</el-dropdown-item>
                  <el-dropdown-item command="up" :disabled="siblings(nodes, node.parent_id ?? null)[0]?.id === node.id">Выше</el-dropdown-item>
                  <el-dropdown-item command="down" :disabled="siblings(nodes, node.parent_id ?? null).at(-1)?.id === node.id">Ниже</el-dropdown-item>
                  <el-dropdown-item command="parent">Переместить…</el-dropdown-item>
                  <el-dropdown-item command="delete" divided :disabled="mandatory(node)">Удалить</el-dropdown-item>
                </el-dropdown-menu></template>
              </el-dropdown>
            </div>
          </template>
        </el-tree>
        <span v-if="busy" role="status" class="structure-hint">Сохранение…</span>
      </div>
      <div class="structure-panel">
        <template v-if="creating || selectedNode">
          <h2>{{ creating ? 'Новый узел' : nodeLabel(selectedNode!) }}</h2>
          <p v-if="creating" class="structure-hint">{{ parentID === null ? 'В корне формы' : `В контейнере «${nodeLabel(nodes.find(node => node.id === parentID)!)}»` }}</p>
          <el-form v-if="creating" label-position="top" :disabled="busy || unsynced">
            <el-form-item label="Категория" required><el-select v-model="kind" placeholder="Выберите категорию" @change="changeKind"><el-option label="Элемент" value="element" /><el-option label="Поле" value="field" /><el-option label="Контейнер" value="container" /></el-select></el-form-item>
            <el-form-item v-if="kind" label="Тип" required><el-select v-model="type" placeholder="Выберите тип" filterable @change="changeType"><el-option v-for="option in typeOptions" :key="option.code" :label="option.label" :value="option.code" /></el-select></el-form-item>
          </el-form>
          <form-field-editor v-if="kind === 'field' && (field || type)" ref="fieldEditor" :key="editorKey" :field="field" :fields="detail.fields" :available-types="detail.available_field_types" :initial-type="type as FormsFieldType" :disabled="!canUpdate || busy || unsynced" @dirty="editorDirty = $event" />
          <form-element-editor v-if="kind === 'element' && (element || type)" ref="elementEditor" :key="editorKey" :element="element" :available-types="detail.available_element_types" :initial-type="type as ElementType" :access-token="accessToken" :permissions="permissions" :disabled="!canUpdate || busy || unsynced" @dirty="editorDirty = $event" />
          <el-form v-if="kind === 'container' && (selectedNode || type)" label-position="top" :disabled="!canUpdate || busy || unsynced">
            <el-form-item v-if="selectedNode" label="Тип"><span>{{ nodeType(selectedNode) }}</span></el-form-item>
            <el-form-item label="Название"><el-input v-model="label" placeholder="Необязательно" /></el-form-item>
          </el-form>
          <div v-if="canUpdate" class="structure-actions"><el-button type="primary" :loading="busy" :disabled="unsynced || (creating ? !type : !dirty)" @click="save">{{ creating ? 'Создать' : 'Сохранить' }}</el-button><el-button :disabled="busy" @click="showNode(selectedID)">Отмена</el-button></div>
        </template>
        <el-empty v-else description="Выберите узел в дереве или добавьте новый" :image-size="72" />
      </div>
    </div>
    <el-dialog v-model="leaveOpen" title="Несохранённые изменения" class="forms-mail-dialog" width="min(520px, 94vw)" :close-on-click-modal="false" :close-on-press-escape="!busy" :show-close="!busy" @closed="finishLeave(false)">
      <p>Сохранить настройки узла перед переходом?</p>
      <el-alert v-if="error" :title="error" type="error" :closable="false" />
      <template #footer><el-button :disabled="busy" @click="finishLeave(false)">Остаться</el-button><el-button :disabled="busy" @click="leaveDiscard">Отбросить</el-button><el-button type="primary" :loading="busy" @click="leaveSave">Сохранить</el-button></template>
    </el-dialog>
    <el-dialog v-model="moveOpen" title="Переместить узел" class="forms-mail-dialog" width="min(480px, 94vw)">
      <el-form label-position="top"><el-form-item label="Родитель"><el-select v-model="moveParent"><el-option :value="0" label="Корень" /><el-option v-for="node in parentOptions" :key="node.id" :value="node.id" :label="nodeLabel(node)" /></el-select></el-form-item></el-form>
      <template #footer><el-button @click="moveOpen = false">Отмена</el-button><el-button type="primary" :loading="busy" @click="moveToParent">Переместить</el-button></template>
    </el-dialog>
  </div>
</template>

<style scoped>
.structure-editor{display:grid;gap:16px;min-width:0}.structure-columns{display:grid;grid-template-columns:minmax(300px,0.9fr) minmax(0,1.1fr);gap:20px;align-items:start}.structure-tree,.structure-panel{min-width:0}.structure-panel{border-left:1px solid var(--el-border-color);padding-left:20px}.structure-toolbar{display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap}.structure-editor h2{font-size:18px;margin:0 0 16px}.structure-toolbar h2{margin:0}.structure-hint{color:var(--el-text-color-secondary);font-size:13px;line-height:1.5;margin:12px 0}.structure-node{display:flex;align-items:center;gap:6px;flex:1;min-width:0;padding:6px 0}.structure-node-text{display:grid;gap:3px;flex:1;min-width:0}.structure-node-text strong{font-weight:500;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.structure-node-text small{color:var(--el-text-color-secondary);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.structure-tree :deep(.el-tree-node__content){height:auto;min-height:52px}.structure-tree :deep(.el-button+.el-button){margin-left:0}.structure-actions{display:flex;flex-wrap:wrap;gap:8px;margin-top:16px}.structure-actions .el-button{margin:0}.structure-panel :deep(.el-select){width:100%}
@media(max-width:1100px){.structure-columns{grid-template-columns:1fr}.structure-panel{border-left:0;border-top:1px solid var(--el-border-color);padding:20px 0 0}}
</style>
