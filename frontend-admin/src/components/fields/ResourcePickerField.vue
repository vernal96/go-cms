<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElOption, ElSelect } from 'element-plus'
import { adminRequest } from '../../api/admin-api'
import type { ResourceLookupResponse, ResourceOption } from '../../types/admin'

const props = withDefaults(defineProps<{ siteId?: number; accessToken?: string; multiple?: boolean }>(), { multiple: false })
const model = defineModel<number | number[] | undefined>()
const options = ref<ResourceOption[]>([])
const loading = ref(false)

function label(item: ResourceOption): string { return item.path ? `${item.display_title} (${item.path})` : item.display_title }
async function load(search = ''): Promise<void> {
  if (!props.siteId || !props.accessToken) return
  loading.value = true
  try {
    const query = new URLSearchParams({ page: '1', per_page: '30' })
    if (search) query.set('search', search)
    const result = await adminRequest<ResourceLookupResponse>(`/api/sites/${props.siteId}/resources/lookup?${query}`, props.accessToken)
    options.value = result.items
  } finally { loading.value = false }
}
onMounted(() => { void load() })
</script>

<template>
  <el-select v-model="model" class="full-width" filterable remote clearable :multiple="multiple" :remote-method="load" :loading="loading">
    <el-option v-for="item in options" :key="item.id" :label="label(item)" :value="item.id" />
  </el-select>
</template>

<style scoped>.full-width { width: 100%; }</style>
