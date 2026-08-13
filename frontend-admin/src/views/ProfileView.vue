<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import {
  ElAlert,
  ElAvatar,
  ElButton,
  ElCard,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElSkeleton,
  ElTabPane,
  ElTabs,
} from 'element-plus'
import { Camera, Delete, FolderOpened, UserFilled } from '@element-plus/icons-vue'

import {
  adminBlob,
  adminRequest,
  adminRequestVoid,
  adminUpload,
} from '../api/admin-api'
import FilePickerDialog from '../components/files/FilePickerDialog.vue'
import type { FilesystemItem, ProfileResponse, ProfileUser } from '../types/admin'

const props = defineProps<{
  accessToken: string
  permissions: ReadonlySet<string>
}>()
const emit = defineEmits<{ updated: [] }>()

const loading = ref(true)
const savingInfo = ref(false)
const savingPassword = ref(false)
const error = ref('')
const activeTab = ref('info')
const profile = ref<ProfileUser | null>(null)
const avatarURL = ref('')
const uploadInput = ref<HTMLInputElement | null>(null)
const pickerVisible = ref(false)
const info = reactive({ login: '', email: '', name: '', last_name: '', middle_name: '', phone: '' })
const password = reactive({ current: '', next: '', confirm: '' })

onMounted(() => void load())
onBeforeUnmount(revokeAvatar)

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    applyResponse(await adminRequest<ProfileResponse>('/api/admin/profile', props.accessToken))
  } catch (caught) {
    error.value = message(caught, 'Не удалось загрузить профиль.')
  } finally {
    loading.value = false
  }
}

function applyResponse(response: ProfileResponse): void {
  profile.value = response.user
  Object.assign(info, {
    login: response.user.login,
    email: response.user.email,
    name: response.user.name,
    last_name: response.user.last_name ?? '',
    middle_name: response.user.middle_name ?? '',
    phone: response.user.phone ?? '',
  })
  void loadAvatar()
}

async function saveInfo(): Promise<void> {
  if (!info.name.trim()) {
    ElMessage.warning('Введите имя.')
    return
  }
  savingInfo.value = true
  try {
    const response = await adminRequest<ProfileResponse>('/api/admin/profile', props.accessToken, {
      method: 'PATCH',
      body: JSON.stringify({
        name: info.name.trim(),
        last_name: nullable(info.last_name),
        middle_name: nullable(info.middle_name),
        phone: nullable(info.phone),
      }),
    })
    applyResponse(response)
    emit('updated')
    ElMessage.success('Профиль сохранён.')
  } catch (caught) {
    ElMessage.error(message(caught, 'Не удалось сохранить профиль.'))
  } finally {
    savingInfo.value = false
  }
}

async function changePassword(): Promise<void> {
  if (!password.current || password.next.length < 12) {
    ElMessage.warning('Укажите текущий пароль и новый пароль не короче 12 символов.')
    return
  }
  if (password.next !== password.confirm) {
    ElMessage.warning('Подтверждение пароля не совпадает.')
    return
  }
  savingPassword.value = true
  try {
    await adminRequestVoid('/api/admin/profile/password', props.accessToken, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ current_password: password.current, new_password: password.next }),
    })
    password.current = ''
    password.next = ''
    password.confirm = ''
    ElMessage.success('Пароль изменён.')
  } catch (caught) {
    ElMessage.error(message(caught, 'Не удалось изменить пароль.'))
  } finally {
    savingPassword.value = false
  }
}

async function uploadAvatar(files: FileList | null): Promise<void> {
  const uploaded = files?.[0]
  if (!uploaded) return
  if (uploaded.size > 5 * 1024 * 1024) {
    ElMessage.warning('Размер аватара не должен превышать 5 МБ.')
    return
  }
  const data = new FormData()
  data.set('file', uploaded, uploaded.name)
  try {
    applyResponse(await adminUpload<ProfileResponse>('/api/admin/profile/avatar/upload', props.accessToken, data))
    emit('updated')
    ElMessage.success('Аватар обновлён.')
  } catch (caught) {
    ElMessage.error(message(caught, 'Не удалось загрузить аватар.'))
  } finally {
    if (uploadInput.value) uploadInput.value.value = ''
  }
}

async function selectAvatar(item: FilesystemItem): Promise<void> {
  try {
    applyResponse(await adminRequest<ProfileResponse>('/api/admin/profile/avatar', props.accessToken, {
      method: 'PUT', body: JSON.stringify({ file_id: item.id }),
    }))
    emit('updated')
    ElMessage.success('Аватар обновлён.')
  } catch (caught) {
    ElMessage.error(message(caught, 'Не удалось выбрать аватар.'))
  }
}

async function removeAvatar(): Promise<void> {
  try {
    applyResponse(await adminRequest<ProfileResponse>('/api/admin/profile/avatar', props.accessToken, { method: 'DELETE' }))
    emit('updated')
    ElMessage.success('Аватар удалён.')
  } catch (caught) {
    ElMessage.error(message(caught, 'Не удалось удалить аватар.'))
  }
}

async function loadAvatar(): Promise<void> {
  revokeAvatar()
  if (!profile.value?.avatar) return
  try {
    const blob = await adminBlob('/api/admin/profile/avatar/preview', props.accessToken)
    avatarURL.value = URL.createObjectURL(blob)
  } catch (caught) {
    ElMessage.error(message(caught, 'Не удалось открыть аватар.'))
  }
}

function revokeAvatar(): void {
  if (avatarURL.value) URL.revokeObjectURL(avatarURL.value)
  avatarURL.value = ''
}

function nullable(value: string): string | null {
  return value.trim() || null
}

function message(value: unknown, fallback: string): string {
  return value instanceof Error ? value.message : fallback
}
</script>

<template>
  <section class="workspace-page profile-page">
    <header class="page-header">
      <div><h1>Профиль</h1><p>Личные данные и безопасность учётной записи</p></div>
    </header>
    <el-alert v-if="error" type="error" :closable="false" :title="error" />
    <el-skeleton v-else-if="loading" :rows="8" animated />
    <el-card v-else-if="profile" class="editor-card" shadow="never">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="Личные данные" name="info">
          <div class="profile-avatar-row">
            <el-avatar :size="104" :src="avatarURL" :icon="avatarURL ? undefined : UserFilled" />
            <div class="profile-avatar-actions">
              <el-button :icon="Camera" @click="uploadInput?.click()">Загрузить</el-button>
              <el-button v-if="permissions.has('core.file.read')" :icon="FolderOpened" @click="pickerVisible = true">Выбрать файл</el-button>
              <el-button v-if="profile.avatar" type="danger" plain :icon="Delete" @click="removeAvatar">Удалить</el-button>
              <small>JPEG, PNG, WebP или GIF, до 5 МБ</small>
            </div>
            <input ref="uploadInput" hidden type="file" accept="image/jpeg,image/png,image/webp,image/gif" @change="uploadAvatar(($event.target as HTMLInputElement).files)" />
          </div>
          <el-form label-position="top" class="identity-form" @submit.prevent="saveInfo">
            <div class="form-grid">
              <el-form-item label="Логин"><el-input v-model="info.login" disabled /></el-form-item>
              <el-form-item label="Email"><el-input v-model="info.email" disabled /></el-form-item>
              <el-form-item label="Имя" required><el-input v-model="info.name" /></el-form-item>
              <el-form-item label="Фамилия"><el-input v-model="info.last_name" /></el-form-item>
              <el-form-item label="Отчество"><el-input v-model="info.middle_name" /></el-form-item>
              <el-form-item label="Телефон"><el-input v-model="info.phone" /></el-form-item>
            </div>
            <el-button type="primary" native-type="submit" :loading="savingInfo">Сохранить</el-button>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="Безопасность" name="security">
          <el-form label-position="top" class="password-form" @submit.prevent="changePassword">
            <el-form-item label="Текущий пароль" required><el-input v-model="password.current" type="password" show-password /></el-form-item>
            <el-form-item label="Новый пароль" required><el-input v-model="password.next" type="password" show-password /></el-form-item>
            <el-form-item label="Подтверждение" required><el-input v-model="password.confirm" type="password" show-password /></el-form-item>
            <el-button type="primary" native-type="submit" :loading="savingPassword">Изменить пароль</el-button>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
    <file-picker-dialog
      v-if="permissions.has('core.file.read')"
      v-model="pickerVisible"
      :access-token="accessToken"
      :permissions="permissions"
      :mime-types="['image/jpeg', 'image/png', 'image/webp', 'image/gif']"
      @select="selectAvatar"
    />
  </section>
</template>
