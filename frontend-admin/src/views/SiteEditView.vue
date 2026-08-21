<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElMessage, ElSkeleton } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'

import { AdminAPIError, adminRequest } from '../api/admin-api'
import SiteForm from '../components/SiteForm.vue'
import { useSelectedSite } from '../composables/use-selected-site'
import type { SiteDetailsResponse, SiteFormPayload } from '../types/admin'

const props = defineProps<{
  accessToken: string
  permissions: ReadonlySet<string>
}>()
const emit = defineEmits<{ unauthorized: [] }>()
const route = useRoute()
const router = useRouter()
const selected = useSelectedSite()
const loading = ref(true)
const submitting = ref(false)
const error = ref<string | null>(null)
const fieldErrors = ref<AdminAPIError['fieldErrors']>([])
const initial = ref<SiteFormPayload | null>(null)

async function load(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const response = await adminRequest<SiteDetailsResponse>(
      `/api/sites/${route.params.siteId}`,
      props.accessToken,
    )
    initial.value = {
      domain: response.site.domain,
      profile_code: response.site.profile_code,
      locale: response.site.locale,
      is_public: response.site.is_public,
      settings: response.site.settings,
    }
  } catch (caught) {
    handleError(caught)
  } finally {
    loading.value = false
  }
}

async function submit(payload: SiteFormPayload): Promise<void> {
  submitting.value = true
  error.value = null
  fieldErrors.value = []
  try {
    const response = await adminRequest<SiteDetailsResponse>(
      `/api/sites/${route.params.siteId}`,
      props.accessToken,
      { method: 'PATCH', body: JSON.stringify(payload) },
    )
    if (selected.selectedSite.value?.id === response.site.id) {
      selected.setSelected({
        id: response.site.id,
        domain: response.site.domain,
      })
    }
    selected.refreshSelector()
    initial.value = payload
    ElMessage.success('Изменения сохранены')
  } catch (caught) {
    handleError(caught)
  } finally {
    submitting.value = false
  }
}

function handleError(caught: unknown): void {
  if (caught instanceof AdminAPIError && caught.status === 401)
    emit('unauthorized')
  else {
    if (caught instanceof AdminAPIError) fieldErrors.value = caught.fieldErrors
    error.value =
      caught instanceof Error ? caught.message : 'Не удалось загрузить сайт.'
  }
}

onMounted(() => void load())
watch(
  () => route.params.siteId,
  () => void load(),
)
</script>

<template>
  <section class="workspace-page narrow-page">
    <header class="page-header">
      <div>
        <h1>Настройки сайта</h1>
        <p>Параметры профиля загружаются с backend</p>
      </div>
    </header>
    <el-skeleton v-if="loading" animated :rows="5" />
    <site-form
      v-else-if="initial"
      :access-token="accessToken"
      :initial="initial"
      editing
      :submitting="submitting"
      :error="error"
      :field-errors="fieldErrors"
      @submit="submit"
      @cancel="router.push('/admin/sites')"
      @unauthorized="emit('unauthorized')"
    />
    <div v-else class="form-load-error">{{ error }}</div>
  </section>
</template>
