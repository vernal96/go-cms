<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'

import { AdminAPIError, adminRequest } from '../api/admin-api'
import SiteForm from '../components/SiteForm.vue'
import { useSelectedSite } from '../composables/use-selected-site'
import type { SiteDetailsResponse, SiteFormPayload } from '../types/admin'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const router = useRouter()
const selected = useSelectedSite()
const submitting = ref(false)
const error = ref<string | null>(null)

async function submit(payload: SiteFormPayload): Promise<void> {
  submitting.value = true
  error.value = null
  try {
    const response = await adminRequest<SiteDetailsResponse>('/api/admin/sites', props.accessToken, {
      method: 'POST',
      body: JSON.stringify({ ...payload, settings: {} }),
    })
    ElMessage.success('Сайт создан')
    selected.refreshSelector()
    await router.push(`/admin/sites/${response.site.id}/edit`)
  } catch (caught) {
    if (caught instanceof AdminAPIError && caught.status === 401) emit('unauthorized')
    else error.value = caught instanceof Error ? caught.message : 'Не удалось создать сайт.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section class="workspace-page narrow-page">
    <header class="page-header"><div><h1>Новый сайт</h1><p>Settings будут созданы пустыми</p></div></header>
    <site-form
      :access-token="accessToken"
      :submitting="submitting"
      :error="error"
      @submit="submit"
      @cancel="router.push('/admin/sites')"
      @unauthorized="emit('unauthorized')"
    />
  </section>
</template>
