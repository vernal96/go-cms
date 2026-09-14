<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElAlert, ElInput, ElOption, ElPagination, ElSelect } from 'element-plus'
import { getForm, listForms } from './api'
import type { FormRecord } from './types'

const props = defineProps<{ siteId?: number; accessToken?: string }>()
const model = defineModel<number | null>()
const items = ref<FormRecord[]>([])
const selected = ref<FormRecord | null>(null)
const search = ref('')
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const error = ref('')
let generation = 0
let selectedGeneration = 0
let timer: ReturnType<typeof setTimeout> | undefined
const choices = computed(() => selected.value && !items.value.some((item) => item.id === selected.value?.id)
  ? [selected.value, ...items.value] : items.value)

async function load(): Promise<void> {
  const current = ++generation
  items.value = []
  total.value = 0
  error.value = ''
  if (!props.siteId || !props.accessToken) { loading.value = false; return }
  loading.value = true
  try {
    const response = await listForms(props.accessToken, props.siteId, page.value, 20, search.value)
    if (current !== generation) return
    items.value = response.items
    total.value = response.pagination.total
  } catch (cause) {
    if (current === generation) error.value = cause instanceof Error ? cause.message : 'Не удалось загрузить формы'
  } finally {
    if (current === generation) loading.value = false
  }
}

watch(() => [props.siteId, props.accessToken] as const, (value, previous) => {
  clearTimeout(timer)
  generation++
  selectedGeneration++
  selected.value = null
  search.value = ''
  page.value = 1
  if (previous && value[0] !== previous[0]) model.value = null
  void load()
}, { immediate: true })

watch(() => [model.value, props.siteId, props.accessToken] as const, async ([id, siteID, token]) => {
  const current = ++selectedGeneration
  selected.value = null
  if (!id || !siteID || !token) return
  try {
    const item = await getForm(token, siteID, id)
    if (current === selectedGeneration) selected.value = item
  } catch (cause) {
    if (current === selectedGeneration) error.value = cause instanceof Error ? cause.message : 'Выбранная форма недоступна'
  }
}, { immediate: true })

function searchChanged(): void {
  clearTimeout(timer)
  generation++
  timer = setTimeout(() => { page.value = 1; void load() }, 250)
}
function changePage(value: number): void { page.value = value; void load() }
onBeforeUnmount(() => { clearTimeout(timer); generation++; selectedGeneration++ })
</script>

<template>
  <div class="form-picker">
    <el-input v-model="search" placeholder="Поиск формы" clearable @update:model-value="searchChanged" />
    <el-alert v-if="error" type="error" :closable="false" :title="error" />
    <el-select v-model="model" :loading="loading" placeholder="Выберите форму" clearable @visible-change="(open: boolean) => { if (open) load() }">
      <el-option v-for="item in choices" :key="item.id" :value="item.id" :label="`${item.name} (${item.code})${item.enabled ? '' : ' — отключена'}`" />
    </el-select>
    <el-pagination v-if="total > 20" :current-page="page" :page-size="20" :total="total" layout="prev, pager, next" @current-change="changePage" />
  </div>
</template>

<style scoped>
.form-picker { display: grid; gap: 8px; width: 100%; }
</style>
