import type { StoredSession } from '../types/auth'

export const adminSessionStorageKey = 'go-cms.admin.session'

export interface AdminSessionStore {
  read(): StoredSession | null
  write(session: StoredSession): void
  clear(): void
}

export const browserSessionStore: AdminSessionStore = {
  read(): StoredSession | null {
    const value = window.sessionStorage.getItem(adminSessionStorageKey)
    if (value === null) {
      return null
    }

    try {
      const parsed = JSON.parse(value) as unknown
      if (!isStoredSession(parsed)) {
        window.sessionStorage.removeItem(adminSessionStorageKey)
        return null
      }
      return parsed
    } catch {
      window.sessionStorage.removeItem(adminSessionStorageKey)
      return null
    }
  },

  write(session: StoredSession): void {
    window.sessionStorage.setItem(
      adminSessionStorageKey,
      JSON.stringify(session),
    )
  },

  clear(): void {
    window.sessionStorage.removeItem(adminSessionStorageKey)
  },
}

export function isSessionExpired(
  session: StoredSession,
  now = Date.now(),
): boolean {
  const expiresAt = Date.parse(session.expiresAt)
  return !Number.isFinite(expiresAt) || expiresAt <= now
}

function isStoredSession(value: unknown): value is StoredSession {
  if (typeof value !== 'object' || value === null) {
    return false
  }

  const candidate = value as Partial<StoredSession>
  return (
    typeof candidate.accessToken === 'string' &&
    candidate.accessToken.length > 0 &&
    typeof candidate.expiresAt === 'string' &&
    candidate.expiresAt.length > 0 &&
    Number.isFinite(Date.parse(candidate.expiresAt))
  )
}
