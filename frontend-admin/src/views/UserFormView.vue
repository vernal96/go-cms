<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  ElAlert,
  ElButton,
  ElCard,
  ElCheckbox,
  ElCheckboxGroup,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElSkeleton,
  ElTabPane,
  ElTabs,
} from 'element-plus'
import { CopyDocument, RefreshRight } from '@element-plus/icons-vue'
import { useRoute, useRouter } from 'vue-router'

import { AdminAPIError, adminRequest, adminRequestVoid } from '../api/admin-api'
import { generatePassword as createPassword } from '../auth/password-generator'
import AccessDeniedView from '../components/AccessDeniedView.vue'
import type { AdminGroup, GroupOptionsResponse, UserDetailsResponse, UserInfoPayload } from '../types/admin'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const route = useRoute()
const router = useRouter()
const isCreate = computed(() => route.params.userId === undefined)
const userId = computed(() => Number(route.params.userId))
const hasAccess = computed(() => isCreate.value ? props.permissions.has('core.user.create') : props.permissions.has('core.user.read'))
const activeTab = ref('info')
const loading = ref(!isCreate.value)
const saving = ref(false)
const error = ref<string | null>(null)
const showPassword = ref(false)
const groups = ref<AdminGroup[]>([])
const groupIDs = ref<number[]>([])
const info = reactive({ login: '', email: '', name: '', last_name: '', middle_name: '', phone: '' })
const password = reactive({ value: '', confirmation: '' })

onMounted(() => {
  if (!hasAccess.value) return
  void initialize()
})

async function initialize(): Promise<void> {
  loading.value = !isCreate.value
  error.value = null
  try {
    const tasks: Promise<unknown>[] = []
    if (props.permissions.has('core.group.read')) tasks.push(loadGroupOptions())
    if (!isCreate.value) tasks.push(loadUser())
    await Promise.all(tasks)
  } catch (caught) {
    handleError(caught)
  } finally {
    loading.value = false
  }
}

async function loadUser(): Promise<void> {
  const response = await adminRequest<UserDetailsResponse>(`/api/admin/users/${userId.value}`, props.accessToken)
  Object.assign(info, {
    login: response.user.login,
    email: response.user.email,
    name: response.user.name,
    last_name: response.user.last_name ?? '',
    middle_name: response.user.middle_name ?? '',
    phone: response.user.phone ?? '',
  })
  if (props.permissions.has('core.group.read')) {
    const selected = await adminRequest<GroupOptionsResponse>(`/api/admin/users/${userId.value}/groups`, props.accessToken)
    groupIDs.value = selected.items.map((item) => item.id)
  }
}

async function loadGroupOptions(): Promise<void> {
  const response = await adminRequest<GroupOptionsResponse>('/api/admin/groups/options?page=1&per_page=100', props.accessToken)
  groups.value = response.items
}

function infoPayload(): UserInfoPayload {
  return {
    login: info.login.trim(), email: info.email.trim(), name: info.name.trim(),
    last_name: optional(info.last_name), middle_name: optional(info.middle_name), phone: optional(info.phone),
  }
}

function optional(value: string): string | null {
  const normalized = value.trim()
  return normalized === '' ? null : normalized
}

function validateInfo(): boolean {
  if (!/^[a-z][a-z0-9._-]{2,63}$/.test(info.login.trim().toLowerCase())) {
    ElMessage.error('Логин должен начинаться с буквы и содержать от 3 до 64 допустимых символов')
    activeTab.value = 'info'
    return false
  }
  if (!info.email.includes('@') || !info.name.trim()) {
    ElMessage.error('Заполните корректный email и имя')
    activeTab.value = 'info'
    return false
  }
  return true
}

function validatePassword(): boolean {
  if (password.value.length < 12) {
    ElMessage.error('Пароль должен содержать не менее 12 символов')
    activeTab.value = 'password'
    return false
  }
  if (password.value !== password.confirmation) {
    ElMessage.error('Пароли не совпадают')
    activeTab.value = 'password'
    return false
  }
  return true
}

async function createUser(): Promise<void> {
  if (!validateInfo() || !validatePassword()) return
  saving.value = true
  try {
    await adminRequest<UserDetailsResponse>('/api/admin/users', props.accessToken, {
      method: 'POST',
      body: JSON.stringify({ ...infoPayload(), password: password.value, group_ids: groupIDs.value }),
    })
    ElMessage.success('Пользователь создан')
    await router.push('/admin/users')
  } catch (caught) {
    handleError(caught)
  } finally {
    saving.value = false
  }
}

async function saveInfo(): Promise<void> {
  if (!validateInfo()) return
  saving.value = true
  try {
    await adminRequest<UserDetailsResponse>(`/api/admin/users/${userId.value}`, props.accessToken, { method: 'PATCH', body: JSON.stringify(infoPayload()) })
    ElMessage.success('Информация сохранена')
  } catch (caught) {
    handleError(caught)
  } finally {
    saving.value = false
  }
}

async function savePassword(): Promise<void> {
  if (!validatePassword()) return
  saving.value = true
  try {
    await adminRequestVoid(`/api/admin/users/${userId.value}/password`, props.accessToken, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ password: password.value }) })
    password.value = ''
    password.confirmation = ''
    ElMessage.success('Пароль изменён')
  } catch (caught) {
    handleError(caught)
  } finally {
    saving.value = false
  }
}

async function saveGroups(): Promise<void> {
  saving.value = true
  try {
    await adminRequestVoid(`/api/admin/users/${userId.value}/groups`, props.accessToken, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ group_ids: groupIDs.value }) })
    ElMessage.success('Группы сохранены')
  } catch (caught) {
    handleError(caught)
  } finally {
    saving.value = false
  }
}

function generatePassword(): void {
  password.value = createPassword()
  password.confirmation = password.value
  showPassword.value = true
}

async function copyPassword(): Promise<void> {
  if (!password.value) return
  await navigator.clipboard.writeText(password.value)
  ElMessage.success('Пароль скопирован')
}

function handleError(caught: unknown): void {
  if (caught instanceof AdminAPIError && caught.status === 401) {
    emit('unauthorized')
    return
  }
  error.value = caught instanceof Error ? caught.message : 'Операция не выполнена.'
  ElMessage.error(error.value)
}
</script>

<template>
  <access-denied-view v-if="!hasAccess" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page form-page">
    <header class="page-header">
      <div><h1>{{ isCreate ? 'Новый пользователь' : 'Редактирование пользователя' }}</h1><p>Информация, пароль и группы доступа</p></div>
      <el-button @click="router.push('/admin/users')">К списку</el-button>
    </header>
    <el-skeleton v-if="loading" animated :rows="8" />
    <el-card v-else shadow="never" class="editor-card">
      <el-alert v-if="error" type="error" :title="error" show-icon :closable="false" />
      <el-tabs v-model="activeTab">
        <el-tab-pane label="Информация" name="info">
          <el-form label-position="top" class="identity-form">
            <div class="form-grid">
              <el-form-item label="Логин" required><el-input v-model="info.login" autocomplete="off" /></el-form-item>
              <el-form-item label="Email" required><el-input v-model="info.email" type="email" /></el-form-item>
              <el-form-item label="Имя" required><el-input v-model="info.name" /></el-form-item>
              <el-form-item label="Фамилия"><el-input v-model="info.last_name" /></el-form-item>
              <el-form-item label="Отчество"><el-input v-model="info.middle_name" /></el-form-item>
              <el-form-item label="Телефон"><el-input v-model="info.phone" /></el-form-item>
            </div>
            <el-button v-if="!isCreate && permissions.has('core.user.update')" type="primary" :loading="saving" @click="saveInfo">Сохранить информацию</el-button>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="Пароль" name="password">
          <div class="password-toolbar">
            <el-button :icon="RefreshRight" @click="generatePassword">Сгенерировать</el-button>
            <el-button :icon="CopyDocument" :disabled="!password.value" @click="copyPassword">Копировать</el-button>
          </div>
          <el-form label-position="top" class="password-form">
            <el-form-item :label="isCreate ? 'Пароль' : 'Новый пароль'" required><el-input v-model="password.value" :type="showPassword ? 'text' : 'password'" show-password autocomplete="new-password" /></el-form-item>
            <el-form-item label="Подтверждение" required><el-input v-model="password.confirmation" :type="showPassword ? 'text' : 'password'" show-password autocomplete="new-password" /></el-form-item>
            <el-button v-if="!isCreate && permissions.has('core.user.update')" type="primary" :loading="saving" @click="savePassword">Сменить пароль</el-button>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="Группы" name="groups">
          <el-alert v-if="!permissions.has('core.group.read')" type="info" title="Нет права на просмотр групп" :closable="false" />
          <template v-else>
            <el-checkbox-group v-model="groupIDs" class="groups-choice" :disabled="!permissions.has('core.group.update')">
              <el-checkbox v-for="item in groups" :key="item.id" :value="item.id"><strong>{{ item.name }}</strong><span class="group-code">{{ item.code }}</span></el-checkbox>
            </el-checkbox-group>
            <el-button v-if="!isCreate && permissions.has('core.group.update')" type="primary" :loading="saving" @click="saveGroups">Сохранить группы</el-button>
          </template>
        </el-tab-pane>
      </el-tabs>
      <div v-if="isCreate" class="form-submit"><el-button type="primary" :loading="saving" @click="createUser">Создать пользователя</el-button></div>
    </el-card>
  </section>
</template>
