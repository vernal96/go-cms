<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import {
  ElAlert,
  ElButton,
  ElForm,
  ElFormItem,
  ElInput,
  ElOption,
  ElSelect,
  ElSkeleton,
  ElSwitch,
} from 'element-plus'

import { AdminAPIError, adminRequest } from '../api/admin-api'
import type { SiteFormPayload, SiteProfile, SiteProfilesResponse } from '../types/admin'

const props = defineProps<{
  accessToken: string
  initial?: SiteFormPayload | null
  editing?: boolean
  submitting?: boolean
  error?: string | null
}>()
const emit = defineEmits<{
  submit: [payload: SiteFormPayload]
  cancel: []
  unauthorized: []
}>()

const profiles = ref<SiteProfile[]>([])
const loadingProfiles = ref(true)
const localError = ref<string | null>(null)
const form = reactive<SiteFormPayload>({
  domain: '', profile_code: '', locale: 'ru-RU', is_public: false,
})

watch(
  () => props.initial,
  (value) => {
    if (value) Object.assign(form, value)
  },
  { immediate: true },
)

onMounted(async () => {
  try {
    const response = await adminRequest<SiteProfilesResponse>(
      '/api/admin/site-profiles',
      props.accessToken,
    )
    profiles.value = response.items
    if (!form.profile_code) {
      form.profile_code = response.items.find((item) => item.creatable)?.code ?? ''
    }
  } catch (error) {
    if (error instanceof AdminAPIError && error.status === 401) emit('unauthorized')
    localError.value = error instanceof Error ? error.message : 'Не удалось загрузить профили.'
  } finally {
    loadingProfiles.value = false
  }
})

function submit(): void {
  localError.value = null
  if (!form.domain.trim() || !form.profile_code || !form.locale.trim()) {
    localError.value = 'Заполните домен, профиль и локаль.'
    return
  }
  emit('submit', {
    domain: form.domain.trim(),
    profile_code: form.profile_code,
    locale: form.locale.trim(),
    is_public: form.is_public,
  })
}
</script>

<template>
  <el-skeleton v-if="loadingProfiles" animated :rows="5" />
  <el-form v-else class="site-form" label-position="top" :model="form" @submit.prevent="submit">
    <el-alert
      v-if="error || localError"
      class="form-alert"
      type="error"
      :closable="false"
      :title="error || localError || ''"
    />
    <el-form-item label="Домен" required>
      <el-input v-model="form.domain" placeholder="example.com" />
    </el-form-item>
    <el-form-item label="Профиль" required>
      <el-select v-model="form.profile_code" class="full-width">
        <el-option
          v-for="profile in profiles"
          :key="profile.code"
          :label="profile.name"
          :value="profile.code"
          :disabled="!editing && !profile.creatable"
        />
      </el-select>
    </el-form-item>
    <el-form-item label="Локаль" required>
      <el-input v-model="form.locale" placeholder="ru-RU" />
    </el-form-item>
    <el-form-item label="Публичный сайт">
      <el-switch v-model="form.is_public" />
    </el-form-item>
    <div class="form-actions">
      <el-button @click="emit('cancel')">Отмена</el-button>
      <el-button type="primary" native-type="submit" :loading="submitting">
        {{ editing ? 'Сохранить' : 'Создать' }}
      </el-button>
    </div>
  </el-form>
</template>
