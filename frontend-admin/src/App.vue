<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { ElCard, ElIcon } from 'element-plus'
import { Loading, Platform } from '@element-plus/icons-vue'

import AccessDeniedView from './components/AccessDeniedView.vue'
import AdminDashboard from './components/AdminDashboard.vue'
import LoginView from './components/LoginView.vue'
import { useAdminAuth } from './auth/use-admin-auth'
import { setAdminUnauthorizedHandler } from './api/admin-api'
import type { LoginCredentials } from './types/auth'

const {
  status,
  user,
  accessToken,
  permissions,
  errorMessage,
  isSubmitting,
  bootstrap,
  signIn,
  logout,
  dispose,
} = useAdminAuth()

function handleSignIn(credentials: LoginCredentials): void {
  void signIn(credentials)
}

onMounted(() => {
  setAdminUnauthorizedHandler(logout)
  void bootstrap()
})
onBeforeUnmount(() => {
  setAdminUnauthorizedHandler(null)
  dispose()
})
</script>

<template>
  <section v-if="status === 'checking'" class="app-state auth-page">
    <el-card class="status-card" shadow="never">
      <div class="status-brand">
        <span class="brand-mark" aria-hidden="true">
          <el-icon :size="24">
            <Platform />
          </el-icon>
        </span>
        <span>Go CMS</span>
      </div>
      <el-icon class="status-spinner is-loading" :size="30">
        <Loading />
      </el-icon>
      <p class="status-copy">Проверяем сессию…</p>
    </el-card>
  </section>

  <login-view
    v-else-if="status === 'anonymous' || status === 'authenticating'"
    :loading="isSubmitting"
    :error-message="errorMessage"
    @submit="handleSignIn"
  />

  <access-denied-view
    v-else-if="status === 'forbidden'"
    @switch-user="logout"
  />

  <admin-dashboard
    v-else-if="status === 'authorized' && user && accessToken"
    :display-name="user.display_name"
    :access-token="accessToken"
    :permissions="permissions"
    @logout="logout"
  />
</template>
