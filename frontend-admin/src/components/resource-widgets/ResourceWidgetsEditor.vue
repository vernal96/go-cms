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
  if (!props.canUpdate) return
  draggingID.value = value.id
  event.dataTransfer?.setData('text/plain', String(value.id))
  if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move'
}

async function drop(area: WidgetArea, index: number, event: DragEvent): Promise<void> {
  event.preventDefault()
  const id = draggingID.value ?? Number(event.dataTransfer?.getData('text/plain'))
  draggingID.value = null
  if (!props.canUpdate || !Number.isInteger(id) || id <= 0) return
  const previous = normalizeWidgetPositions(props.modelValue)
  const moved = moveWidget(previous, id, area, index)
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
  }
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
    <section v-if="allowedAreas.includes('body')" class="widget-area" @dragover.prevent @drop="drop('body', body.length, $event)">
      <header>
        <div><h3>Body</h3><p>Основная область страницы</p></div>
        <el-button :icon="Plus" :disabled="!canUpdate" @click="add('body')">Добавить виджет</el-button>
      </header>
      <el-empty v-if="!body.length" description="В основной области нет виджетов" :image-size="70" />
      <template v-for="(item, index) in body" :key="item.id">
        <div class="widget-drop-target" @dragover.prevent @drop.stop="drop('body', index, $event)" />
        <widget-card
          :widget="item"
          :definition="definition(item.code)"
          :disabled="!canUpdate"
          @dragstart="startDrag(item, $event)"
          @edit="edit(item)"
          @delete="remove(item)"
        />
      </template>
      <div class="widget-drop-target" @dragover.prevent @drop.stop="drop('body', body.length, $event)" />
    </section>

    <section v-if="allowedAreas.includes('sidebar')" class="widget-area" @dragover.prevent @drop="drop('sidebar', sidebar.length, $event)">
      <header>
        <div><h3>Sidebar</h3><p>Боковая область страницы</p></div>
        <el-button :icon="Plus" :disabled="!canUpdate" @click="add('sidebar')">Добавить виджет</el-button>
      </header>
      <el-empty v-if="!sidebar.length" description="В боковой области нет виджетов" :image-size="70" />
      <template v-for="(item, index) in sidebar" :key="item.id">
        <div class="widget-drop-target" @dragover.prevent @drop.stop="drop('sidebar', index, $event)" />
        <widget-card
          :widget="item"
          :definition="definition(item.code)"
          :disabled="!canUpdate"
          @dragstart="startDrag(item, $event)"
          @edit="edit(item)"
          @delete="remove(item)"
        />
      </template>
      <div class="widget-drop-target" @dragover.prevent @drop.stop="drop('sidebar', sidebar.length, $event)" />
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
.resource-widgets-editor { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 22px; }
.widget-area { min-height: 240px; padding: 16px; border: 1px solid var(--el-border-color); border-radius: 8px; background: var(--el-fill-color-extra-light); }
.widget-area header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-bottom: 10px; }
.widget-area h3 { margin: 0; }
.widget-area p { margin: 4px 0 0; color: var(--el-text-color-secondary); font-size: 13px; }
.widget-drop-target { height: 8px; border-radius: 4px; }
.widget-drop-target:hover { background: var(--el-color-primary-light-7); }
@media (max-width: 900px) { .resource-widgets-editor { grid-template-columns: 1fr; } }
</style>

