// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedSite } from './use-selected-site'

describe('useSelectedSite', () => {
  beforeEach(() => {
    localStorage.clear()
    useSelectedSite().reset()
    vi.unstubAllGlobals()
  })

  it('validates and restores a stored site through the API', async () => {
    localStorage.setItem('go-cms.admin.selected-site', JSON.stringify({ id: 7, domain: 'old.test' }))
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      site: { id: 7, domain: 'example.com', profile_code: 'dev', locale: 'ru-RU', settings: {}, is_public: false },
      permissions: { read: true, create: true, update: true, delete: true },
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)

    const selected = useSelectedSite()
    await selected.initialize('token')

    expect(fetchMock).toHaveBeenCalledWith('/api/admin/sites/7', expect.objectContaining({
      headers: expect.any(Headers),
    }))
    expect(selected.selectedSite.value).toEqual({ id: 7, domain: 'example.com' })
  })

  it('clears a stored site that is no longer available', async () => {
    localStorage.setItem('go-cms.admin.selected-site', JSON.stringify({ id: 8, domain: 'missing.test' }))
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'not_found', message: 'not found' },
    }), { status: 404, headers: { 'Content-Type': 'application/json' } })))

    const selected = useSelectedSite()
    await selected.initialize('token')

    expect(selected.selectedSite.value).toBeNull()
    expect(localStorage.getItem('go-cms.admin.selected-site')).toBeNull()
  })
})
