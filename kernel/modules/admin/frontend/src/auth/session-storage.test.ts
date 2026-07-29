import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { StoredSession } from '../types/auth'
import {
  adminSessionStorageKey,
  browserSessionStore,
  isSessionExpired,
} from './session-storage'

function memoryStorage(): Storage {
  const values = new Map<string, string>()
  return {
    get length() {
      return values.size
    },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => [...values.keys()][index] ?? null,
    removeItem: (key) => {
      values.delete(key)
    },
    setItem: (key, value) => {
      values.set(key, value)
    },
  }
}

describe('admin session storage', () => {
  beforeEach(() => {
    vi.stubGlobal('window', { sessionStorage: memoryStorage() })
  })

  it('stores and restores one typed session', () => {
    const session: StoredSession = {
      accessToken: 'signed',
      expiresAt: '2026-07-29T12:15:00Z',
    }

    browserSessionStore.write(session)

    expect(browserSessionStore.read()).toEqual(session)
  })

  it('removes malformed values instead of restoring them', () => {
    window.sessionStorage.setItem(adminSessionStorageKey, '{"token":"old"}')

    expect(browserSessionStore.read()).toBeNull()
    expect(window.sessionStorage.getItem(adminSessionStorageKey)).toBeNull()
  })

  it('detects expired and invalid expiry timestamps', () => {
    expect(
      isSessionExpired(
        {
          accessToken: 'signed',
          expiresAt: '2026-07-29T12:15:00Z',
        },
        Date.parse('2026-07-29T12:15:00Z'),
      ),
    ).toBe(true)
    expect(
      isSessionExpired({
        accessToken: 'signed',
        expiresAt: 'not-a-date',
      }),
    ).toBe(true)
  })
})
