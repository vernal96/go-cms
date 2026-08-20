<script setup lang="ts">
import { computed } from 'vue'
import { Delete, Edit, Rank } from '@element-plus/icons-vue'
import { ElButton, ElCard, ElIcon, ElTag } from 'element-plus'
import type { ResourceWidget, WidgetDefinition } from '../../types/admin'

const props = defineProps<{
  widget: ResourceWidget
  definition: WidgetDefinition
  disabled?: boolean
  dragging?: boolean
}>()
const emit = defineEmits<{
  edit: []
  delete: []
  dragstart: [event: DragEvent]
  dragend: []
}>()

const summaries = computed(() => (props.definition.summary_fields ?? [])
  .filter((key) => Object.hasOwn(props.widget.params, key))
  .map((key) => ({
    key,
    label: (props.definition.fields ?? []).find((field) => field.key === key)?.label ?? key,
    value: summaryValue(props.widget.params[key]),
  })))
const viewLabel = computed(() => props.widget.view === 'default'
  ? 'Default'
  : props.definition.views.find((view) => view.code === props.widget.view)?.label ?? props.widget.view)

function summaryValue(value: unknown): string {
  if (Array.isArray(value)) return value.join(', ')
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
</script>

<template>
  <el-card
    class="widget-card"
    :class="{ 'is-dragging': dragging }"
    shadow="never"
  >
    <div class="widget-card-header">
      <el-icon
        class="widget-drag-handle"
        :class="{ 'is-disabled': disabled }"
        :draggable="!disabled"
        title="Перетащить виджет"
        @click.stop
        @dragstart.stop="emit('dragstart', $event)"
        @dragend.stop="emit('dragend')"
      ><Rank /></el-icon>
      <div class="widget-card-title">
        <strong>{{ definition.label }}</strong>
        <small v-if="definition.description">{{ definition.description }}</small>
      </div>
      <el-tag v-if="!widget.enabled" type="info">Выключен</el-tag>
      <el-button text :icon="Edit" :disabled="disabled" aria-label="Редактировать виджет" @click="emit('edit')" />
      <el-button text type="danger" :icon="Delete" :disabled="disabled" aria-label="Удалить виджет" @click="emit('delete')" />
    </div>
    <div v-if="summaries.length" class="widget-card-summary">
      <span v-for="summary in summaries" :key="summary.key">
        <b>{{ summary.label }}:</b> {{ summary.value }}
      </span>
    </div>
    <div class="widget-card-presentation">
      {{ widget.columns }}/12 · {{ viewLabel }} ·
      отступы {{ widget.margin_top }}/{{ widget.margin_bottom }}
    </div>
  </el-card>
</template>

<style scoped>
.widget-card { margin-bottom: 10px; transition: border-color .14s ease, box-shadow .14s ease, opacity .14s ease, transform .14s ease; }
.widget-card.is-dragging { border-color: var(--el-color-primary); opacity: .46; transform: scale(.985); }
.widget-card-header { display: flex; align-items: center; gap: 10px; }
.widget-drag-handle { width: 28px; height: 28px; color: var(--el-text-color-secondary); background: var(--el-fill-color-light); border: 1px solid var(--el-border-color-light); border-radius: 6px; cursor: grab; flex: 0 0 auto; transition: color .14s ease, background-color .14s ease, border-color .14s ease, transform .14s ease; }
.widget-drag-handle:hover:not(.is-disabled) { color: var(--el-color-primary); background: var(--el-color-primary-light-9); border-color: var(--el-color-primary-light-5); transform: scale(1.06); }
.widget-drag-handle:active:not(.is-disabled) { cursor: grabbing; }
.widget-drag-handle.is-disabled { cursor: default; opacity: .45; }
.widget-card-title { display: flex; flex: 1; flex-direction: column; min-width: 0; }
.widget-card-title small, .widget-card-presentation { color: var(--el-text-color-secondary); }
.widget-card-summary { display: flex; flex-wrap: wrap; gap: 8px 18px; margin: 10px 0; }
.widget-card-presentation { margin-top: 8px; font-size: 12px; }
</style>
