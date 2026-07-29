<script setup lang="ts">
import { reactive, ref } from 'vue'
import {
  ElAlert,
  ElButton,
  ElCard,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  type FormInstance,
  type FormRules,
} from 'element-plus'
import { Lock, Platform, User } from '@element-plus/icons-vue'

import type { LoginCredentials } from '../types/auth'

defineProps<{
  loading: boolean
  errorMessage: string | null
}>()

const emit = defineEmits<{
  submit: [credentials: LoginCredentials]
}>()

const formRef = ref<FormInstance>()
const credentials = reactive<LoginCredentials>({
  identifier: '',
  password: '',
})
const rules: FormRules<LoginCredentials> = {
  identifier: [
    {
      required: true,
      message: 'Введите логин или email',
      trigger: 'blur',
    },
  ],
  password: [
    {
      required: true,
      message: 'Введите пароль',
      trigger: 'blur',
    },
  ],
}

async function submit(): Promise<void> {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) {
    return
  }
  emit('submit', {
    identifier: credentials.identifier.trim(),
    password: credentials.password,
  })
}
</script>

<template>
  <section class="app-state auth-page">
    <el-card class="login-card" shadow="never">
      <div class="login-heading">
        <span class="brand-mark login-brand" aria-hidden="true">
          <el-icon :size="26">
            <Platform />
          </el-icon>
        </span>
        <div>
          <h1>Вход в Go CMS</h1>
          <p>Используйте учётную запись администратора</p>
        </div>
      </div>

      <el-alert
        v-if="errorMessage"
        class="login-alert"
        :title="errorMessage"
        type="error"
        show-icon
        :closable="false"
      />

      <el-form
        ref="formRef"
        class="login-form"
        :model="credentials"
        :rules="rules"
        label-position="top"
        @submit.prevent="submit"
      >
        <el-form-item label="Логин или email" prop="identifier">
          <el-input
            v-model="credentials.identifier"
            :prefix-icon="User"
            autocomplete="username"
            placeholder="admin"
            size="large"
            :disabled="loading"
          />
        </el-form-item>

        <el-form-item label="Пароль" prop="password">
          <el-input
            v-model="credentials.password"
            :prefix-icon="Lock"
            autocomplete="current-password"
            placeholder="Введите пароль"
            type="password"
            show-password
            size="large"
            :disabled="loading"
          />
        </el-form-item>

        <el-button
          class="login-submit"
          type="primary"
          size="large"
          native-type="submit"
          :loading="loading"
        >
          Войти
        </el-button>
      </el-form>
    </el-card>
  </section>
</template>
