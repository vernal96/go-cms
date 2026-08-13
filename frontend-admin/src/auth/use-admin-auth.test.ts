import { describe, expect, it, vi } from 'vitest'

import { AdminAPIError } from '../api/admin-api'
import type {
  AdminSessionResponse,
  LoginResponse,
  StoredSession,
} from '../types/auth'
import type { AdminSessionStore } from './session-storage'
import {
  useAdminAuth,
  type AdminAuthDependencies,
} from './use-admin-auth'

const now = Date.parse('2026-07-29T12:00:00Z')
const validSession: StoredSession = {
  accessToken: 'signed',
  expiresAt: '2026-07-29T12:15:00Z',
}
const loginResponse: LoginResponse = {
  access_token: validSession.accessToken,
  token_type: 'Bearer',
  expires_at: validSession.expiresAt,
}
const adminSession: AdminSessionResponse = {
  user: {
    id: 1,
    login: 'admin',
    email: 'admin@example.test',
    display_name: 'Администратор',
    color_scheme: 'system',
    accent_color: 'blue',
    has_avatar: false,
    avatar_updated_at: null,
  },
  permissions: ['admin.panel.read'],
}

function dependencies(
  stored: StoredSession | null,
  loadSession: AdminAuthDependencies['loadSession'] = vi
    .fn()
    .mockResolvedValue(adminSession),
): {
  auth: ReturnType<typeof useAdminAuth>
  storage: AdminSessionStore
  login: AdminAuthDependencies['login']
  loadSession: AdminAuthDependencies['loadSession']
} {
  let current = stored
  const storage: AdminSessionStore = {
    read: vi.fn(() => current),
    write: vi.fn((session) => {
      current = session
    }),
    clear: vi.fn(() => {
      current = null
    }),
  }
  const login = vi.fn().mockResolvedValue(loginResponse)
  const authDependencies: AdminAuthDependencies = {
    login,
    loadSession,
    storage,
    now: () => now,
    setTimer: vi.fn(
      () => 1 as unknown as ReturnType<typeof setTimeout>,
    ),
    clearTimer: vi.fn(),
  }
  return {
    auth: useAdminAuth(authDependencies),
    storage,
    login,
    loadSession,
  }
}

describe('admin authorization state', () => {
  it('restores a valid session after a page reload', async () => {
    const fixture = dependencies(validSession)

    await fixture.auth.bootstrap()

    expect(fixture.auth.status.value).toBe('authorized')
    expect(fixture.auth.user.value).toEqual(adminSession.user)
    expect(fixture.loadSession).toHaveBeenCalledWith('signed')
  })

  it('clears an expired session without calling the API', async () => {
    const fixture = dependencies({
      accessToken: 'expired',
      expiresAt: '2026-07-29T11:59:59Z',
    })

    await fixture.auth.bootstrap()

    expect(fixture.auth.status.value).toBe('anonymous')
    expect(fixture.storage.clear).toHaveBeenCalled()
    expect(fixture.loadSession).not.toHaveBeenCalled()
  })

  it('clears a session rejected with 401', async () => {
    const fixture = dependencies(
      validSession,
      vi.fn().mockRejectedValue(
        new AdminAPIError(401, 'unauthenticated', 'expired'),
      ),
    )

    await fixture.auth.bootstrap()

    expect(fixture.auth.status.value).toBe('anonymous')
    expect(fixture.storage.clear).toHaveBeenCalled()
  })

  it('shows access denied for a session rejected with 403', async () => {
    const fixture = dependencies(
      validSession,
      vi
        .fn()
        .mockRejectedValue(new AdminAPIError(403, 'forbidden', 'denied')),
    )

    await fixture.auth.bootstrap()

    expect(fixture.auth.status.value).toBe('forbidden')
    expect(fixture.storage.clear).not.toHaveBeenCalled()
  })

  it('stores a new token and verifies admin access after login', async () => {
    const fixture = dependencies(null)

    await fixture.auth.signIn({
      identifier: 'admin',
      password: 'admin-dev-only-2026',
    })

    expect(fixture.login).toHaveBeenCalled()
    expect(fixture.storage.write).toHaveBeenCalledWith(validSession)
    expect(fixture.auth.status.value).toBe('authorized')
  })
})
