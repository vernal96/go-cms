<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElButton, ElInput, ElMessage, ElTable, ElTableColumn } from 'element-plus'
import { useRouter } from 'vue-router'

import { AdminAPIError, adminRequest } from '../api/admin-api'
import type { LibraryItemsResponse } from '../types/admin'

const props = defineProps<{ accessToken: string; siteId: number; libraryId: number }>()
const router = useRouter()
const loading = ref(false)
const search = ref('')
const items = ref<LibraryItemsResponse['items']>([])
const nextCursor = ref('')
const cursor = ref('')
const history = ref<string[]>([])

async function load(reset = false): Promise<void> {
  if (reset) { cursor.value = ''; history.value = [] }
  loading.value = true
  const query = new URLSearchParams({ limit: '25' })
  if (cursor.value) query.set('cursor', cursor.value)
  if (search.value.trim()) query.set('search', search.value.trim())
  try {
    const response = await adminRequest<LibraryItemsResponse>(
      `/api/sites/${props.siteId}/resources/${props.libraryId}/items?${query}`,
      props.accessToken,
    )
    items.value = response.items
    nextCursor.value = response.next_cursor
  } catch (error) {
    ElMessage.error(error instanceof AdminAPIError ? error.message : 'Не удалось загрузить ресурсы библиотеки.')
  } finally { loading.value = false }
}

function add(): void { void router.push(`/admin/sites/${props.siteId}/resources/${props.libraryId}/items/new`) }
function edit(id: number): void { void router.push(`/admin/sites/${props.siteId}/resources/${props.libraryId}/items/${id}/edit`) }
function next(): void { if (!nextCursor.value) return; history.value.push(cursor.value); cursor.value = nextCursor.value; void load() }
function previous(): void { if (!history.value.length) return; cursor.value = history.value.pop() ?? ''; void load() }

onMounted(() => void load())
watch(() => [props.siteId, props.libraryId], () => void load(true))
</script>

<template>
  <div class="library-items-tab">
    <div class="library-items-toolbar">
      <el-button type="primary" @click="add">Добавить ресурс</el-button>
      <el-input v-model="search" clearable placeholder="Поиск по названию или коду" @keyup.enter="load(true)" @clear="load(true)" />
      <el-button @click="load(true)">Найти</el-button>
    </div>
    <el-table :data="items" v-loading="loading" row-key="id">
      <el-table-column prop="id" label="ID" width="100" />
      <el-table-column prop="title" label="Название" min-width="260" />
      <el-table-column prop="slug" label="Код" min-width="180" />
      <el-table-column label="Активность" width="130">
        <template #default="scope">{{ scope.row.deleted ? 'Удалён' : (scope.row.is_public ? 'Активен' : 'Скрыт') }}</template>
      </el-table-column>
      <el-table-column label="Редактировать" width="150">
        <template #default="scope"><el-button link type="primary" @click="edit(scope.row.id)">Открыть</el-button></template>
      </el-table-column>
    </el-table>
    <div class="library-items-pagination">
      <el-button :disabled="!history.length || loading" @click="previous">Назад</el-button>
      <el-button :disabled="!nextCursor || loading" @click="next">Далее</el-button>
    </div>
  </div>
</template>

<style scoped>
.library-items-toolbar { display: grid; grid-template-columns: auto minmax(220px, 1fr) auto; gap: 12px; margin-bottom: 16px; }
.library-items-pagination { display: flex; justify-content: flex-end; gap: 8px; margin-top: 16px; }
</style>
