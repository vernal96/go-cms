<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  ElButton,
  ElCard,
  ElEmpty,
  ElResult,
  ElSkeleton,
  ElSkeletonItem,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus'
import { useRouter } from 'vue-router'

import { AdminAPIError, adminRequest } from '../api/admin-api'
import type { DashboardResponse } from '../types/admin'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const router = useRouter()
const dashboard = ref<DashboardResponse | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
let controller: AbortController | null = null

const hasContent = computed(() => {
  const value = dashboard.value
  return value !== null && Boolean(value.sites || value.resources || value.users || value.groups)
})
const showResourceCounts = computed(() => dashboard.value?.resources !== undefined)

async function load(): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  error.value = null
  try {
    dashboard.value = await adminRequest<DashboardResponse>(
      '/api/admin/dashboard',
      props.accessToken,
      { signal: controller.signal },
    )
  } catch (caught) {
    if (caught instanceof DOMException && caught.name === 'AbortError') return
    if (caught instanceof AdminAPIError && caught.status === 401) {
      emit('unauthorized')
      return
    }
    error.value = caught instanceof Error
      ? caught.message
      : 'Не удалось загрузить общую информацию.'
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <section class="workspace-page dashboard-page">
    <header class="page-header dashboard-header">
      <div>
        <h1>Главная</h1>
        <p>Общая информация по доступным разделам</p>
      </div>
    </header>

    <div v-if="loading" class="dashboard-metrics dashboard-loading" aria-label="Загрузка дашборда">
      <el-card v-for="index in 4" :key="index" shadow="never" class="dashboard-card">
        <el-skeleton animated>
          <template #template>
            <el-skeleton-item variant="text" class="dashboard-skeleton-title" />
            <el-skeleton-item variant="h1" class="dashboard-skeleton-value" />
            <el-skeleton-item variant="text" />
          </template>
        </el-skeleton>
      </el-card>
    </div>

    <el-result
      v-else-if="error"
      icon="error"
      title="Не удалось загрузить дашборд"
      :sub-title="error"
    >
      <template #extra>
        <el-button type="primary" @click="load">Повторить</el-button>
      </template>
    </el-result>

    <el-empty
      v-else-if="!hasContent"
      description="Нет доступных показателей"
    />

    <template v-else>
      <div class="dashboard-metrics">
        <el-card v-if="dashboard?.sites" data-testid="dashboard-sites-card" shadow="never" class="dashboard-card">
          <div class="dashboard-card-label">Сайты</div>
          <div class="dashboard-card-value">{{ dashboard.sites.total }}</div>
          <div class="dashboard-card-details">
            <span>Публичных: {{ dashboard.sites.public }}</span>
            <span>Закрытых: {{ dashboard.sites.private }}</span>
          </div>
        </el-card>

        <el-card v-if="dashboard?.resources" data-testid="dashboard-resources-card" shadow="never" class="dashboard-card">
          <div class="dashboard-card-label">Ресурсы</div>
          <div class="dashboard-card-value">{{ dashboard.resources.total }}</div>
          <div class="dashboard-card-details">Во всех доступных сайтах</div>
        </el-card>

        <el-card v-if="dashboard?.users" data-testid="dashboard-users-card" shadow="never" class="dashboard-card">
          <div class="dashboard-card-label">Пользователи</div>
          <div class="dashboard-card-value">{{ dashboard.users.total }}</div>
          <div class="dashboard-card-details">
            <span>Активных: {{ dashboard.users.active }}</span>
            <span>Заблокированных: {{ dashboard.users.blocked }}</span>
          </div>
        </el-card>

        <el-card v-if="dashboard?.groups" data-testid="dashboard-groups-card" shadow="never" class="dashboard-card">
          <div class="dashboard-card-label">Группы</div>
          <div class="dashboard-card-value">{{ dashboard.groups.total }}</div>
          <div class="dashboard-card-details">Группы управления доступом</div>
        </el-card>
      </div>

      <section v-if="dashboard?.sites" class="dashboard-sites">
        <header class="dashboard-section-header">
          <div>
            <h2>Сайты и ресурсы</h2>
            <p>Первые 10 сайтов в алфавитном порядке</p>
          </div>
          <el-button
            v-if="permissions.has('core.site.read')"
            type="primary"
            plain
            @click="router.push('/admin/sites')"
          >
            Все сайты
          </el-button>
        </header>

        <el-empty
          v-if="dashboard.sites.items.length === 0"
          description="Сайтов пока нет"
        />
        <el-table v-else :data="dashboard.sites.items" stripe class="dashboard-sites-table">
          <el-table-column prop="domain" label="Домен" min-width="260" />
          <el-table-column label="Статус" width="150">
            <template #default="{ row }">
              <el-tag :type="row.is_public ? 'success' : 'info'" effect="light">
                {{ row.is_public ? 'Публичный' : 'Закрытый' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column
            v-if="showResourceCounts"
            data-testid="resource-count-column"
            label="Ресурсы"
            width="140"
            align="right"
          >
            <template #default="{ row }">{{ row.resource_count ?? 0 }}</template>
          </el-table-column>
        </el-table>
      </section>
    </template>
  </section>
</template>
