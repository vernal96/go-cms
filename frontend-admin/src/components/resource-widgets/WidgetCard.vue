<script setup lang="ts">
import { computed } from 'vue'
import { Delete, Edit, Rank } from '@element-plus/icons-vue'
import { ElButton, ElCard, ElIcon, ElTag } from 'element-plus'
import type { ResourceWidget, WidgetDefinition } from '../../types/admin'

const props = defineProps<{
  widget: ResourceWidget
  definition: WidgetDefinition
  disabled?: boolean
}>()
const emit = defineEmits<{
  edit: []
  delete: []
  dragstart: [event: DragEvent]
}>()

const summaries = computed(() => props.definition.summary_fields
  .filter((key) => Object.hasOwn(props.widget.params, key))
  .map((key) => ({
    key,
    label: props.definition.fields.find((field) => field.key === key)?.label ?? key,
    value: summaryValue(props.widget.params[key]),
  })))

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
    shadow="never"
    :draggable="!disabled"
    @dragstart="emit('dragstart', $event)"
  >
    <div class="widget-card-header">
      <el-icon class="widget-drag-handle"><Rank /></el-icon>
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
      {{ widget.columns }}/12 · {{ widget.view === 'default' ? 'Default' : widget.view }} ·
      отступы {{ widget.margin_top }}/{{ widget.margin_bottom }}
    </div>
  </el-card>
</template>

<style scoped>
.widget-card { margin-bottom: 10px; }
.widget-card-header { display: flex; align-items: center; gap: 10px; }
.widget-drag-handle { cursor: grab; color: var(--el-text-color-secondary); }
.widget-card-title { display: flex; flex: 1; flex-direction: column; min-width: 0; }
.widget-card-title small, .widget-card-presentation { color: var(--el-text-color-secondary); }
.widget-card-summary { display: flex; flex-wrap: wrap; gap: 8px 18px; margin: 10px 0; }
.widget-card-presentation { margin-top: 8px; font-size: 12px; }
</style>

