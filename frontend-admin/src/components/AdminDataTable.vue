<script setup lang="ts">
import {
  ElButton,
  ElEmpty,
  ElInput,
  ElPagination,
  ElResult,
  ElSkeleton,
  ElSkeletonItem,
  ElTable,
  ElTableColumn,
} from 'element-plus'
import { Search } from '@element-plus/icons-vue'

import type { AdminTableAction, AdminTableColumn } from '../types/admin'

defineProps<{
  columns: AdminTableColumn[]
  rows: Record<string, unknown>[]
  actions?: AdminTableAction[]
  loading: boolean
  error?: string | null
  page: number
  perPage: number
  total: number
  search: string
  emptyText?: string
}>()

const emit = defineEmits<{
  'update:search': [value: string]
  'page-change': [page: number]
  retry: []
  action: [key: string, row: Record<string, unknown>]
}>()

function cell(column: AdminTableColumn, row: Record<string, unknown>): string {
  if (column.formatter) return column.formatter(row)
  const value = row[column.prop]
  return value === null || value === undefined ? '—' : String(value)
}
</script>

<template>
  <section class="admin-table">
    <div class="admin-table-toolbar">
      <el-input
        :model-value="search"
        :prefix-icon="Search"
        clearable
        placeholder="Поиск"
        aria-label="Поиск"
        @update:model-value="emit('update:search', String($event))"
      />
      <slot name="toolbar" />
    </div>

    <el-skeleton v-if="loading" animated :rows="6" class="table-skeleton">
      <template #template>
        <el-skeleton-item v-for="index in 6" :key="index" variant="rect" class="table-skeleton-row" />
      </template>
    </el-skeleton>
    <el-result v-else-if="error" icon="error" title="Не удалось загрузить данные" :sub-title="error">
      <template #extra><el-button type="primary" @click="emit('retry')">Повторить</el-button></template>
    </el-result>
    <el-empty v-else-if="rows.length === 0" :description="emptyText ?? 'Данных пока нет'" />
    <el-table v-else :data="rows" stripe class="admin-table-grid">
      <el-table-column
        v-for="column in columns"
        :key="column.prop"
        :label="column.label"
        :width="column.width"
        min-width="140"
      >
        <template #default="{ row }">{{ cell(column, row) }}</template>
      </el-table-column>
      <el-table-column v-if="actions?.length" label="Действия" width="210" align="right">
        <template #default="{ row }">
          <el-button
            v-for="action in actions.filter((item) => !item.visible || item.visible(row))"
            :key="action.key"
            text
            size="small"
            :type="action.danger ? 'danger' : 'primary'"
            @click="emit('action', action.key, row)"
          >{{ action.label }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-if="total > perPage"
      class="admin-table-pagination"
      background
      layout="total, prev, pager, next"
      :current-page="page"
      :page-size="perPage"
      :total="total"
      @current-change="emit('page-change', $event)"
    />
  </section>
</template>
