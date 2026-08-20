<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElButton, ElEmpty, ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { AdminAPIError, adminRequest, adminRequestVoid } from '../../api/admin-api'
import type {
  ResourceTemplate,
  ResourceWidget,
  WidgetArea,
  WidgetDefinition,
} from '../../types/admin'
import { moveWidget, normalizeWidgetPositions, widgetOrder, type WidgetSettingsValue } from './model'
import WidgetCard from './WidgetCard.vue'
import WidgetPickerDialog from './WidgetPickerDialog.vue'
import WidgetSettingsDialog from './WidgetSettingsDialog.vue'

const props = defineProps<{
  accessToken: string
  siteId: number
  resourceId: number
  template: ResourceTemplate
  definitions: WidgetDefinition[]
  modelValue: ResourceWidget[]
  canUpdate: boolean
}>()
const emit = defineEmits<{
  'update:modelValue': [value: ResourceWidget[]]
  unauthorized: []
}>()

const pickerOpen = ref(false)
const settingsOpen = ref(false)
const saving = ref(false)
const pendingArea = ref<WidgetArea>('body')
const selectedDefinition = ref<WidgetDefinition | null>(null)
const editingWidget = ref<ResourceWidget | null>(null)
const draggingID = ref<number | null>(null)
const activeTarget = ref<{ area: WidgetArea; index: number } | null>(null)
const reordering = ref(false)

const widgetDragType = 'application/x-go-cms-widget'

const allowedAreas = computed(() => props.template.widget_areas)
const body = computed(() => widgetsIn('body'))
const sidebar = computed(() => widgetsIn('sidebar'))

function widgetsIn(area: WidgetArea): ResourceWidget[] {
  return props.modelValue
    .filter((widget) => widget.area === area)
    .sort((left, right) => left.position - right.position)
}

function definition(code: string): WidgetDefinition {
  return props.definitions.find((item) => item.code === code) ?? {
    code, module_code: '', module_label: 'Недоступный модуль', module_description: '',
    label: code, description: 'Определение виджета недоступно текущему профилю.', fields: [],
    editor_tabs: [], summary_fields: [], views: [],
  }
}

function add(area: WidgetArea): void {
  pendingArea.value = area
  pickerOpen.value = true
}

function selectWidget(value: WidgetDefinition): void {
  selectedDefinition.value = value
  editingWidget.value = null
  pickerOpen.value = false
  settingsOpen.value = true
}

function edit(value: ResourceWidget): void {
  selectedDefinition.value = definition(value.code)
  editingWidget.value = value
  settingsOpen.value = true
}

async function save(value: WidgetSettingsValue): Promise<void> {
  if (!selectedDefinition.value) return
  saving.value = true
  try {
    const path = `/api/admin/sites/${props.siteId}/resources/${props.resourceId}/widgets`
    const saved = editingWidget.value
      ? await adminRequest<ResourceWidget>(`${path}/${editingWidget.value.id}`, props.accessToken, {
          method: 'PATCH', body: JSON.stringify(value),
        })
      : await adminRequest<ResourceWidget>(path, props.accessToken, {
          method: 'POST', body: JSON.stringify({
            code: selectedDefinition.value.code,
            area: pendingArea.value,
            ...value,
          }),
        })
    const current = editingWidget.value
      ? props.modelValue.map((widget) => widget.id === saved.id ? saved : widget)
      : [...props.modelValue, saved]
    emit('update:modelValue', normalizeWidgetPositions(current))
    settingsOpen.value = false
    ElMessage.success(editingWidget.value ? 'Виджет обновлён' : 'Виджет добавлен')
  } catch (error) {
    handleError(error, 'Не удалось сохранить виджет.')
  } finally {
    saving.value = false
  }
}

async function remove(value: ResourceWidget): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `Удалить виджет «${definition(value.code).label}»?`,
      'Удалить виджет?',
      { type: 'warning', confirmButtonText: 'Удалить', cancelButtonText: 'Отмена' },
    )
  } catch { return }
  try {
    await adminRequestVoid(
      `/api/admin/sites/${props.siteId}/resources/${props.resourceId}/widgets/${value.id}`,
      props.accessToken,
      { method: 'DELETE' },
    )
    emit('update:modelValue', normalizeWidgetPositions(props.modelValue.filter((widget) => widget.id !== value.id)))
    ElMessage.success('Виджет удалён')
  } catch (error) {
    handleError(error, 'Не удалось удалить виджет.')
  }
}

function startDrag(value: ResourceWidget, event: DragEvent): void {
  if (!props.canUpdate || reordering.value || !event.dataTransfer) return
  draggingID.value = value.id
  activeTarget.value = null
  event.dataTransfer.setData(widgetDragType, String(value.id))
  event.dataTransfer.effectAllowed = 'move'
}

function activateDropTarget(area: WidgetArea, index: number, event: DragEvent): void {
  if (!canDropAt(area, index)) {
    if (event.dataTransfer) event.dataTransfer.dropEffect = 'none'
    if (isActiveTarget(area, index)) activeTarget.value = null
    return
  }
  event.preventDefault()
  activeTarget.value = { area, index }
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'
}

function leaveDropTarget(area: WidgetArea, index: number, event: DragEvent): void {
  const current = event.currentTarget as HTMLElement | null
  const related = event.relatedTarget as Node | null
  if (current && related && current.contains(related)) return
  if (isActiveTarget(area, index)) activeTarget.value = null
}

async function drop(area: WidgetArea, index: number, event: DragEvent): Promise<void> {
  const id = draggingID.value ?? Number(event.dataTransfer?.getData(widgetDragType))
  if (!props.canUpdate || reordering.value || !Number.isInteger(id) || id <= 0 || !canDropAt(area, index, id)) {
    finishDrag()
    return
  }
  event.preventDefault()
  const previous = normalizeWidgetPositions(props.modelValue)
  const moved = moveWidget(previous, id, area, index)
  finishDrag()
  reordering.value = true
  emit('update:modelValue', moved)
  try {
    const response = await adminRequest<{ items: ResourceWidget[] }>(
      `/api/admin/sites/${props.siteId}/resources/${props.resourceId}/widgets/order`,
      props.accessToken,
      { method: 'PUT', body: JSON.stringify({ items: widgetOrder(moved) }) },
    )
    emit('update:modelValue', normalizeWidgetPositions(response.items))
  } catch (error) {
    emit('update:modelValue', previous)
    handleError(error, 'Не удалось изменить порядок виджетов.')
  } finally {
    reordering.value = false
  }
}

function canDropAt(area: WidgetArea, index: number, id = draggingID.value): boolean {
  if (!props.canUpdate || reordering.value || id === null) return false
  const previous = normalizeWidgetPositions(props.modelValue)
  const moved = moveWidget(previous, id, area, index)
  const before = widgetOrder(previous)
  const after = widgetOrder(moved)
  return before.some((item, position) => {
    const next = after[position]
    return !next || item.id !== next.id || item.area !== next.area || item.position !== next.position
  })
}

function isActiveTarget(area: WidgetArea, index: number): boolean {
  return activeTarget.value?.area === area && activeTarget.value.index === index
}

function areaCanDrop(area: WidgetArea): boolean {
  const count = area === 'body' ? body.value.length : sidebar.value.length
  for (let index = 0; index <= count; index++) {
    if (canDropAt(area, index)) return true
  }
  return false
}

function finishDrag(): void {
  draggingID.value = null
  activeTarget.value = null
}

function handleError(error: unknown, fallback: string): void {
  if (error instanceof AdminAPIError && error.status === 401) {
    emit('unauthorized')
    return
  }
  ElMessage.error(error instanceof Error ? error.message : fallback)
}
</script>

<template>
  <div class="resource-widgets-editor">
    <section
      v-if="allowedAreas.includes('body')"
      class="widget-area"
      :class="{
        'is-drag-available': areaCanDrop('body'),
        'is-drop-area': activeTarget?.area === 'body',
        'is-reordering': reordering,
      }"
    >
      <header>
        <div><h3>Body</h3><p>Основная область страницы</p></div>
        <el-button :icon="Plus" :disabled="!canUpdate || reordering" @click="add('body')">Добавить виджет</el-button>
      </header>
      <div
        v-if="!body.length"
        class="widget-empty-drop-target"
        :class="{ 'is-available': canDropAt('body', 0), 'is-active': isActiveTarget('body', 0) }"
        @dragenter.stop="activateDropTarget('body', 0, $event)"
        @dragover.stop="activateDropTarget('body', 0, $event)"
        @dragleave.stop="leaveDropTarget('body', 0, $event)"
        @drop.stop="drop('body', 0, $event)"
      ><el-empty description="В основной области нет виджетов" :image-size="70" /></div>
      <template v-for="(item, index) in body" :key="item.id">
        <div
          class="widget-drop-target"
          :class="{ 'is-available': canDropAt('body', index), 'is-active': isActiveTarget('body', index) }"
          @dragenter.stop="activateDropTarget('body', index, $event)"
          @dragover.stop="activateDropTarget('body', index, $event)"
          @dragleave.stop="leaveDropTarget('body', index, $event)"
          @drop.stop="drop('body', index, $event)"
        />
        <widget-card
          :widget="item"
          :definition="definition(item.code)"
          :disabled="!canUpdate || reordering"
          :dragging="draggingID === item.id"
          @dragstart="startDrag(item, $event)"
          @dragend="finishDrag"
          @edit="edit(item)"
          @delete="remove(item)"
        />
      </template>
      <div
        v-if="body.length"
        class="widget-drop-target"
        :class="{ 'is-available': canDropAt('body', body.length), 'is-active': isActiveTarget('body', body.length) }"
        @dragenter.stop="activateDropTarget('body', body.length, $event)"
        @dragover.stop="activateDropTarget('body', body.length, $event)"
        @dragleave.stop="leaveDropTarget('body', body.length, $event)"
        @drop.stop="drop('body', body.length, $event)"
      />
    </section>

    <section
      v-if="allowedAreas.includes('sidebar')"
      class="widget-area"
      :class="{
        'is-drag-available': areaCanDrop('sidebar'),
        'is-drop-area': activeTarget?.area === 'sidebar',
        'is-reordering': reordering,
      }"
    >
      <header>
        <div><h3>Sidebar</h3><p>Боковая область страницы</p></div>
        <el-button :icon="Plus" :disabled="!canUpdate || reordering" @click="add('sidebar')">Добавить виджет</el-button>
      </header>
      <div
        v-if="!sidebar.length"
        class="widget-empty-drop-target"
        :class="{ 'is-available': canDropAt('sidebar', 0), 'is-active': isActiveTarget('sidebar', 0) }"
        @dragenter.stop="activateDropTarget('sidebar', 0, $event)"
        @dragover.stop="activateDropTarget('sidebar', 0, $event)"
        @dragleave.stop="leaveDropTarget('sidebar', 0, $event)"
        @drop.stop="drop('sidebar', 0, $event)"
      ><el-empty description="В боковой области нет виджетов" :image-size="70" /></div>
      <template v-for="(item, index) in sidebar" :key="item.id">
        <div
          class="widget-drop-target"
          :class="{ 'is-available': canDropAt('sidebar', index), 'is-active': isActiveTarget('sidebar', index) }"
          @dragenter.stop="activateDropTarget('sidebar', index, $event)"
          @dragover.stop="activateDropTarget('sidebar', index, $event)"
          @dragleave.stop="leaveDropTarget('sidebar', index, $event)"
          @drop.stop="drop('sidebar', index, $event)"
        />
        <widget-card
          :widget="item"
          :definition="definition(item.code)"
          :disabled="!canUpdate || reordering"
          :dragging="draggingID === item.id"
          @dragstart="startDrag(item, $event)"
          @dragend="finishDrag"
          @edit="edit(item)"
          @delete="remove(item)"
        />
      </template>
      <div
        v-if="sidebar.length"
        class="widget-drop-target"
        :class="{ 'is-available': canDropAt('sidebar', sidebar.length), 'is-active': isActiveTarget('sidebar', sidebar.length) }"
        @dragenter.stop="activateDropTarget('sidebar', sidebar.length, $event)"
        @dragover.stop="activateDropTarget('sidebar', sidebar.length, $event)"
        @dragleave.stop="leaveDropTarget('sidebar', sidebar.length, $event)"
        @drop.stop="drop('sidebar', sidebar.length, $event)"
      />
    </section>

    <widget-picker-dialog v-model="pickerOpen" :widgets="definitions" @select="selectWidget" />
    <widget-settings-dialog
      v-model="settingsOpen"
      :definition="selectedDefinition"
      :widget="editingWidget"
      :saving="saving"
      @save="save"
    />
  </div>
</template>

<style scoped>
.resource-widgets-editor { display: grid; grid-template-columns: minmax(0, 2fr) minmax(0, 1fr); gap: 22px; }
.widget-area { min-height: 240px; padding: 16px; border: 1px solid var(--el-border-color); border-radius: 8px; background: var(--el-fill-color-extra-light); transition: background-color .14s ease, border-color .14s ease, box-shadow .14s ease, opacity .14s ease; }
.widget-area.is-drag-available { border-color: var(--el-color-primary-light-5); background: color-mix(in srgb, var(--el-color-primary) 3%, var(--el-fill-color-extra-light)); box-shadow: inset 0 0 0 1px var(--el-color-primary-light-7); }
.widget-area.is-drop-area { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); box-shadow: inset 0 0 0 1px var(--el-color-primary); }
.widget-area.is-reordering { cursor: progress; opacity: .78; }
.widget-area header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-bottom: 10px; }
.widget-area h3 { margin: 0; }
.widget-area p { margin: 4px 0 0; color: var(--el-text-color-secondary); font-size: 13px; }
.widget-drop-target { display: flex; height: 8px; align-items: center; border-radius: 4px; transition: height .14s ease, background-color .14s ease; }
.widget-drop-target::after { width: 100%; height: 3px; border-radius: 3px; background: transparent; content: ''; transition: background-color .14s ease, box-shadow .14s ease; }
.widget-drop-target.is-available { height: 18px; }
.widget-drop-target.is-available::after { background: var(--el-color-primary-light-7); }
.widget-drop-target.is-active::after { background: var(--el-color-primary); box-shadow: 0 0 0 2px color-mix(in srgb, var(--el-color-primary) 18%, transparent); }
.widget-empty-drop-target { min-height: 158px; border: 1px solid transparent; border-radius: 7px; transition: background-color .14s ease, border-color .14s ease, box-shadow .14s ease; }
.widget-empty-drop-target.is-available { border-color: var(--el-color-primary-light-5); background: color-mix(in srgb, var(--el-color-primary) 4%, transparent); }
.widget-empty-drop-target.is-active { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); box-shadow: inset 0 0 0 1px var(--el-color-primary); }
@media (max-width: 900px) { .resource-widgets-editor { grid-template-columns: 1fr; } }
</style>
