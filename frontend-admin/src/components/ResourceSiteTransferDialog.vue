<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { ElAlert, ElButton, ElDialog, ElOption, ElPagination, ElSelect } from 'element-plus'

import { adminRequest } from '../api/admin-api'
import type { Resource, ResourceTreeItem, SiteOption, SiteOptionsResponse } from '../types/admin'

const props = defineProps<{ accessToken: string; sourceSiteId: number }>()
const emit = defineEmits<{
  transferred: [payload: { resource: Resource; source: ResourceTreeItem; target: SiteOption }]
  error: [error: unknown]
}>()

const visible = ref(false)
const source = ref<ResourceTreeItem | null>(null)
const options = ref<SiteOption[]>([])
const targetID = ref<number | null>(null)
const search = ref('')
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const submitting = ref(false)
const errorMessage = ref('')
let debounceTimer: ReturnType<typeof setTimeout> | null = null
let controller: AbortController | null = null
let requestSequence = 0

function open(item: ResourceTreeItem): void {
  source.value = item
  targetID.value = null
  search.value = ''
  page.value = 1
  total.value = 0
  errorMessage.value = ''
  visible.value = true
  void load()
}

async function load(): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  const sequence = ++requestSequence
  loading.value = true
  const query = new URLSearchParams({
    search: search.value,
    page: String(page.value),
    per_page: '10',
    exclude_id: String(props.sourceSiteId),
  })
  try {
    const response = await adminRequest<SiteOptionsResponse>(
      `/api/sites/options?${query}`,
      props.accessToken,
      { signal: controller.signal },
    )
    if (sequence !== requestSequence) return
    options.value = response.items
    total.value = response.pagination.total
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') return
    errorMessage.value = error instanceof Error ? error.message : 'Не удалось загрузить сайты.'
    emit('error', error)
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}

function remoteSearch(value: string): void {
  search.value = value.trim()
  page.value = 1
  if (debounceTimer !== null) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => void load(), 300)
}

async function transfer(): Promise<void> {
  if (!source.value || targetID.value === null) return
  const target = options.value.find((item) => item.id === targetID.value)
  if (!target) return
  submitting.value = true
  errorMessage.value = ''
  try {
    const resource = await adminRequest<Resource>(
      `/api/sites/${props.sourceSiteId}/resources/${source.value.id}/transfer`,
      props.accessToken,
      {
        method: 'POST',
        body: JSON.stringify({
          target_site_id: target.id,
          expected_version: source.value.version,
        }),
      },
    )
    visible.value = false
    emit('transferred', { resource, source: source.value, target })
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Не удалось перенести ресурс.'
    emit('error', error)
  } finally {
    submitting.value = false
  }
}

onBeforeUnmount(() => {
  controller?.abort()
  if (debounceTimer !== null) clearTimeout(debounceTimer)
})

defineExpose({ open })
</script>

<template>
  <el-dialog v-model="visible" title="Перенести на сайт" width="520px" destroy-on-close>
    <p v-if="source" class="move-dialog-copy">
      Выберите сайт для ресурса <strong>«{{ source.display_title }}»</strong> и всего его поддерева.
      Ресурс станет последним элементом в корне целевого сайта.
    </p>
    <el-alert v-if="errorMessage" type="error" :closable="false" :title="errorMessage" />
    <el-select
      v-model="targetID"
      class="full-width"
      filterable
      remote
      clearable
      placeholder="Выберите сайт"
      no-data-text="Доступных сайтов нет"
      :remote-method="remoteSearch"
      :loading="loading"
    >
      <el-option v-for="item in options" :key="item.id" :label="item.domain" :value="item.id" />
      <template #footer>
        <el-pagination
          v-model:current-page="page"
          size="small"
          layout="prev, pager, next"
          :page-size="10"
          :total="total"
          @current-change="load"
        />
      </template>
    </el-select>
    <template #footer>
      <el-button @click="visible = false">Отмена</el-button>
      <el-button type="primary" :loading="submitting" :disabled="targetID === null" @click="transfer">
        Перенести
      </el-button>
    </template>
  </el-dialog>
</template>
