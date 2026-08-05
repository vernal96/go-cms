import { computed, ref } from 'vue'

import {
  AdminAPIError,
  loadAdminSession,
  login,
} from '../api/admin-api'
import type {
  AdminSessionResponse,
  AdminUser,
  AuthStatus,
  LoginCredentials,
  LoginResponse,
  StoredSession,
} from '../types/auth'
import {
  browserSessionStore,
  isSessionExpired,
  type AdminSessionStore,
} from './session-storage'

const maximumTimerDelay = 2_147_483_647

export interface AdminAuthDependencies {
  login(credentials: LoginCredentials): Promise<LoginResponse>
  loadSession(accessToken: string): Promise<AdminSessionResponse>
  storage: AdminSessionStore
  now(): number
  setTimer(callback: () => void, delay: number): ReturnType<typeof setTimeout>
  clearTimer(timer: ReturnType<typeof setTimeout>): void
}

const browserDependencies: AdminAuthDependencies = {
  login,
  loadSession: loadAdminSession,
  storage: browserSessionStore,
  now: Date.now,
  setTimer: (callback, delay) => globalThis.setTimeout(callback, delay),
  clearTimer: (timer) => globalThis.clearTimeout(timer),
}

export function useAdminAuth(
  dependencies: AdminAuthDependencies = browserDependencies,
) {
  const status = ref<AuthStatus>('checking')
  const user = ref<AdminUser | null>(null)
  const accessToken = ref<string | null>(null)
  const permissions = ref<ReadonlySet<string>>(new Set())
  const errorMessage = ref<string | null>(null)
  let expirationTimer: ReturnType<typeof setTimeout> | null = null

  const isSubmitting = computed(() => status.value === 'authenticating')

  async function bootstrap(): Promise<void> {
    clearExpirationTimer()
    status.value = 'checking'
    user.value = null
    accessToken.value = null
    permissions.value = new Set()
    errorMessage.value = null

    const stored = dependencies.storage.read()
    if (stored === null) {
      status.value = 'anonymous'
      return
    }
    if (isSessionExpired(stored, dependencies.now())) {
      endSession('Сессия истекла. Войдите снова.')
      return
    }

    await authorize(stored)
  }

  async function signIn(credentials: LoginCredentials): Promise<void> {
    clearExpirationTimer()
    status.value = 'authenticating'
    user.value = null
    errorMessage.value = null

    try {
      const response = await dependencies.login(credentials)
      const session: StoredSession = {
        accessToken: response.access_token,
        expiresAt: response.expires_at,
      }
      dependencies.storage.write(session)
      await authorize(session)
    } catch (error) {
      dependencies.storage.clear()
      status.value = 'anonymous'
      errorMessage.value =
        error instanceof AdminAPIError && error.status === 401
          ? 'Неверный логин или пароль.'
          : 'Не удалось войти. Проверьте доступность сервера.'
    }
  }

  function logout(): void {
    endSession(null)
  }

  function dispose(): void {
    clearExpirationTimer()
  }

  async function authorize(session: StoredSession): Promise<void> {
    if (isSessionExpired(session, dependencies.now())) {
      endSession('Сессия истекла. Войдите снова.')
      return
    }

    try {
      const response = await dependencies.loadSession(session.accessToken)
      user.value = response.user
      accessToken.value = session.accessToken
      permissions.value = new Set(response.permissions)
      status.value = 'authorized'
      errorMessage.value = null
      scheduleExpiration(session)
    } catch (error) {
      user.value = null
      accessToken.value = null
      permissions.value = new Set()
      if (error instanceof AdminAPIError && error.status === 401) {
        endSession('Сессия истекла. Войдите снова.')
        return
      }
      if (error instanceof AdminAPIError && error.status === 403) {
        status.value = 'forbidden'
        errorMessage.value = null
        scheduleExpiration(session)
        return
      }

      status.value = 'anonymous'
      errorMessage.value = 'Не удалось проверить сессию. Попробуйте ещё раз.'
    }
  }

  function scheduleExpiration(session: StoredSession): void {
    clearExpirationTimer()
    const delay = Date.parse(session.expiresAt) - dependencies.now()
    if (delay <= 0) {
      endSession('Сессия истекла. Войдите снова.')
      return
    }
    expirationTimer = dependencies.setTimer(
      () => endSession('Сессия истекла. Войдите снова.'),
      Math.min(delay, maximumTimerDelay),
    )
  }

  function endSession(message: string | null): void {
    clearExpirationTimer()
    dependencies.storage.clear()
    user.value = null
    accessToken.value = null
    permissions.value = new Set()
    status.value = 'anonymous'
    errorMessage.value = message
  }

  function clearExpirationTimer(): void {
    if (expirationTimer === null) {
      return
    }
    dependencies.clearTimer(expirationTimer)
    expirationTimer = null
  }

  return {
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
  }
}
