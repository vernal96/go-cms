<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElButton, ElDialog, ElEmpty, ElInput, ElTabPane, ElTabs } from 'element-plus'
import type { WidgetDefinition } from '../../types/admin'

const props = defineProps<{
  modelValue: boolean
  widgets: WidgetDefinition[]
}>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  select: [definition: WidgetDefinition]
}>()

const search = ref('')
const activeModule = ref('all')
const modules = computed(() => {
  const values = new Map<string, string>()
  for (const widget of props.widgets) values.set(widget.module_code, widget.module_label)
  return [...values].map(([code, label]) => ({ code, label }))
})
const filtered = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  return props.widgets.filter((widget) =>
    (activeModule.value === 'all' || widget.module_code === activeModule.value) &&
    (!query || `${widget.label} ${widget.description} ${widget.module_label}`.toLocaleLowerCase().includes(query)),
  )
})

watch(() => props.modelValue, (open) => {
  if (open) {
    search.value = ''
    activeModule.value = 'all'
  }
})
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    title="Добавить виджет"
    width="min(920px, calc(100vw - 32px))"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-input v-model="search" clearable placeholder="Поиск виджетов" class="widget-search" />
    <el-tabs v-model="activeModule">
      <el-tab-pane label="Все" name="all" />
      <el-tab-pane v-for="module in modules" :key="module.code" :label="module.label" :name="module.code" />
    </el-tabs>
    <el-empty v-if="!filtered.length" description="Подходящие виджеты не найдены" />
    <div v-else class="widget-picker-grid">
      <el-button
        v-for="widget in filtered"
        :key="widget.code"
        class="widget-picker-item"
        @click="emit('select', widget)"
      >
        <strong>{{ widget.label }}</strong>
        <span>{{ widget.description }}</span>
        <small>{{ widget.module_label }}</small>
      </el-button>
    </div>
  </el-dialog>
</template>

<style scoped>
.widget-search { margin-bottom: 8px; }
.widget-picker-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.widget-picker-item { height: auto; min-height: 100px; margin: 0; padding: 16px; white-space: normal; }
.widget-picker-item :deep(span) { display: flex; width: 100%; flex-direction: column; align-items: flex-start; gap: 6px; text-align: left; }
.widget-picker-item small { color: var(--el-text-color-secondary); }
@media (max-width: 700px) { .widget-picker-grid { grid-template-columns: 1fr; } }
</style>
