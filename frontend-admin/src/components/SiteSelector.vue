<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElButton, ElOption, ElPagination, ElSelect } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'

import { adminRequest } from '../api/admin-api'
import { useSelectedSite } from '../composables/use-selected-site'
import type { SiteOption, SiteOptionsResponse } from '../types/admin'

const props = defineProps<{ accessToken: string; canCreate: boolean }>()
const emit = defineEmits<{ error: [error: unknown] }>()
const router = useRouter()
const selected = useSelectedSite()
const options = ref<SiteOption[]>([])
const selectedID = ref<number | null>(selected.selectedSite.value?.id ?? null)
const loading = ref(false)
const search = ref('')
const page = ref(1)
const total = ref(0)
let debounceTimer: ReturnType<typeof setTimeout> | null = null
let controller: AbortController | null = null
let requestSequence = 0

async function load(): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  const sequence = ++requestSequence
  loading.value = true
  const query = new URLSearchParams({
    search: search.value,
    page: String(page.value),
    per_page: '10',
  })
  try {
    const response = await adminRequest<SiteOptionsResponse>(
      `/api/admin/sites/options?${query}`,
      props.accessToken,
      { signal: controller.signal },
    )
    if (sequence !== requestSequence) return
    options.value = response.items
    total.value = response.pagination.total
    const current = selected.selectedSite.value
    if (current && !options.value.some((item) => item.id === current.id)) {
      options.value.unshift(current)
    }
  } catch (error) {
    if (!(error instanceof DOMException && error.name === 'AbortError')) emit('error', error)
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

function handleVisible(visible: boolean): void {
  if (visible) void load()
}

function choose(value: number | null): void {
  if (value === null) {
    selected.clearSelected()
    return
  }
  const option = options.value.find((item) => item.id === value)
  if (option) selected.setSelected(option)
}

watch(selected.selectedSite, (value) => {
  selectedID.value = value?.id ?? null
  if (value && !options.value.some((item) => item.id === value.id)) options.value.unshift(value)
}, { immediate: true })
watch(selected.selectorRevision, () => void load())

onMounted(() => void load())

onBeforeUnmount(() => {
  controller?.abort()
  if (debounceTimer !== null) clearTimeout(debounceTimer)
})
</script>

<template>
  <div class="site-selector">
    <el-select
      v-model="selectedID"
      class="site-select"
      placeholder="Выберите сайт"
      filterable
      remote
      clearable
      :remote-method="remoteSearch"
      :loading="loading"
      no-data-text="Сайты не найдены"
      @visible-change="handleVisible"
      @change="choose"
    >
      <el-option v-for="item in options" :key="item.id" :label="item.domain" :value="item.id" />
      <template #footer>
        <el-pagination
          v-model:current-page="page"
          small
          layout="prev, pager, next"
          :page-size="10"
          :total="total"
          @current-change="load"
        />
      </template>
    </el-select>
    <el-button
      v-if="canCreate"
      circle
      :icon="Plus"
      aria-label="Создать сайт"
      @click="router.push('/admin/sites/create')"
    />
  </div>
</template>
