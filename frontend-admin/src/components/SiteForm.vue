<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
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
import DynamicFieldsForm from './fields/DynamicFieldsForm.vue'
import {
  createFieldValues,
  fieldErrorMessage,
  unsupportedFieldTypes,
  validateFieldValues,
  type DynamicFieldErrors,
} from './fields/model'
import type {
  SiteFormPayload,
  SiteProfile,
  SiteProfilesResponse,
} from '../types/admin'
import type { FieldValidationError } from '../types/auth'

const props = defineProps<{
  accessToken: string
  initial?: SiteFormPayload | null
  editing?: boolean
  submitting?: boolean
  error?: string | null
  fieldErrors?: FieldValidationError[]
}>()
const emit = defineEmits<{
  submit: [payload: SiteFormPayload]
  cancel: []
  unauthorized: []
}>()

const profiles = ref<SiteProfile[]>([])
const loadingProfiles = ref(true)
const localError = ref<string | null>(null)
const localFieldErrors = ref<DynamicFieldErrors>({})
const form = reactive<SiteFormPayload>({
  domain: '',
  profile_code: '',
  locale: 'ru-RU',
  is_public: false,
  settings: {},
})
const selectedProfile = computed(
  () =>
    profiles.value.find((profile) => profile.code === form.profile_code) ??
    null,
)
const displayedFieldErrors = computed<DynamicFieldErrors>(() => {
  const result = { ...localFieldErrors.value }
  for (const error of props.fieldErrors ?? []) {
    result[error.key] = fieldErrorMessage(error.rule, error.param)
  }
  return result
})

watch(
  () => props.initial,
  (value) => {
    if (value) Object.assign(form, value, { settings: { ...value.settings } })
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
      form.profile_code = response.items[0]?.code ?? ''
    }
    const fields = selectedProfile.value?.fields ?? []
    form.settings = createFieldValues(fields, form.settings)
  } catch (error) {
    if (error instanceof AdminAPIError && error.status === 401)
      emit('unauthorized')
    localError.value =
      error instanceof Error ? error.message : 'Не удалось загрузить профили.'
  } finally {
    loadingProfiles.value = false
  }
})

watch(
  () => form.profile_code,
  (code, previous) => {
    if (!previous || code === previous || profiles.value.length === 0) return
    form.settings = createFieldValues(selectedProfile.value?.fields ?? [])
    localFieldErrors.value = {}
  },
)

function submit(): void {
  localError.value = null
  localFieldErrors.value = {}
  if (!form.domain.trim() || !form.profile_code || !form.locale.trim()) {
    localError.value = 'Заполните домен, профиль и локаль.'
    return
  }
  const fields = selectedProfile.value?.fields ?? []
  if (unsupportedFieldTypes(fields).length > 0) {
    localError.value =
      'Форма содержит неизвестные типы полей и не может быть отправлена.'
    return
  }
  localFieldErrors.value = validateFieldValues(fields, form.settings)
  if (Object.keys(localFieldErrors.value).length > 0) return
  emit('submit', {
    domain: form.domain.trim(),
    profile_code: form.profile_code,
    locale: form.locale.trim(),
    is_public: form.is_public,
    settings: { ...form.settings },
  })
}
</script>

<template>
  <el-skeleton v-if="loadingProfiles" animated :rows="5" />
  <el-form
    v-else
    class="site-form"
    label-position="top"
    :model="form"
    @submit.prevent="submit"
  >
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
        />
      </el-select>
    </el-form-item>
    <el-form-item label="Локаль" required>
      <el-input v-model="form.locale" placeholder="ru-RU" />
    </el-form-item>
    <el-form-item label="Публичный сайт">
      <el-switch v-model="form.is_public" />
    </el-form-item>
    <dynamic-fields-form
      v-if="selectedProfile"
      :fields="selectedProfile.fields"
      :model-value="form.settings"
      :errors="displayedFieldErrors"
      @update:model-value="form.settings = $event"
    />
    <div class="form-actions">
      <el-button @click="emit('cancel')">Отмена</el-button>
      <el-button type="primary" native-type="submit" :loading="submitting">
        {{ editing ? 'Сохранить' : 'Создать' }}
      </el-button>
    </div>
  </el-form>
</template>
