// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from 'vitest'

import { AdminPluginRegistry } from '../admin-plugins/registry'
import type { AdminNavigationItem } from '../types/admin'
import { resolveAdminNavigation, useAdminNavigation } from './use-admin-navigation'

const view = { template: '<div />' }
const registry = new AdminPluginRegistry([
  {
    code: 'core',
    routes: [{ name: 'core.sites', path: '/admin/sites', component: view }],
  },
  {
    code: 'forms',
    routes: [{ name: 'forms.list', path: '/admin/forms', component: view }],
  },
  {
    code: 'seo',
    routes: [{ name: 'seo.list', path: '/admin/seo', component: view }],
  },
])

const globalItem: AdminNavigationItem = {
  code: 'sites', label: 'Сайты', route: 'core.sites', order: 100, scope: 'global',
}

describe('useAdminNavigation', () => {
  beforeEach(() => vi.unstubAllGlobals())

  it('drops stale site items immediately and refreshes for the new site', async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      const siteItem = url.endsWith('site_id=1')
        ? { code: 'forms', label: 'Формы', route: 'forms.list', order: 400, scope: 'site' }
        : { code: 'seo', label: 'SEO', route: 'seo.list', order: 400, scope: 'site' }
      return Promise.resolve(new Response(JSON.stringify({
        items: [globalItem, siteItem],
      }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    })
    vi.stubGlobal('fetch', fetchMock)
    const navigation = useAdminNavigation(registry)

    await navigation.refresh('token', 1)
    expect(navigation.items.value.map((item) => item.code)).toEqual(['sites', 'forms'])

    const refresh = navigation.refresh('token', 2)
    expect(navigation.items.value.map((item) => item.code)).toEqual(['sites'])
    await refresh

    expect(navigation.items.value.map((item) => item.code)).toEqual(['sites', 'seo'])
    expect(fetchMock.mock.calls.map(([url]) => String(url))).toEqual([
      '/api/admin/navigation?site_id=1',
      '/api/admin/navigation?site_id=2',
    ])
  })

  it('omits unsupported semantic routes with a useful warning', () => {
    const warn = vi.fn()
    const items = resolveAdminNavigation([
      globalItem,
      { code: 'missing', label: 'Missing', route: 'missing.list', order: 200, scope: 'site' },
      {
        code: 'empty-group', label: 'Empty', order: 300, scope: 'site',
        children: [{ code: 'missing-child', label: 'Child', route: 'missing.child', order: 100, scope: 'site' }],
      },
    ], registry, warn)

    expect(items.map((item) => item.code)).toEqual(['sites'])
    expect(warn).toHaveBeenCalledTimes(2)
    expect(warn.mock.calls[0]?.[0]).toContain('missing.list')
  })
})
