// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedSite } from '../composables/use-selected-site'
import AdminDashboard from './AdminDashboard.vue'

async function mountDashboard(permissions: string[]) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/admin/dashboard', component: { template: '<div />' } }],
  })
  await router.push('/admin/dashboard')
  await router.isReady()
  const wrapper = shallowMount(AdminDashboard, {
    props: {
      displayName: 'Администратор',
      accessToken: 'token',
      permissions: new Set(permissions),
    },
    global: { plugins: [router], renderStubDefaultSlot: true },
  })
  await flushPromises()
  return wrapper
}

describe('AdminDashboard', () => {
  beforeEach(() => {
    localStorage.clear()
    useSelectedSite().reset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('does not restore a stored site or mount resource navigation without site read access', async () => {
    localStorage.setItem('go-cms.admin.selected-site', JSON.stringify({ id: 7, domain: 'stored.test' }))
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = await mountDashboard(['admin.panel.read', 'core.resource.read'])

    expect(fetchMock).not.toHaveBeenCalled()
    expect(wrapper.findComponent({ name: 'SiteSelector' }).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'ResourceTree' }).exists()).toBe(false)
    expect(wrapper.text()).toContain('Нет доступа к сайтам')
    const home = wrapper.findAllComponents({ name: 'RouterLink' })
      .find((link) => link.props('to') === '/admin/dashboard')
    expect(home?.classes()).toContain('brand-mark')
    expect(home?.attributes('aria-label')).toBe('На главную')
    expect(wrapper.text()).not.toContain('Главная')
  })

  it('keeps the site selector and resource tree on the dashboard when permitted', async () => {
    const wrapper = await mountDashboard([
      'admin.panel.read',
      'core.site.read',
      'core.resource.read',
    ])

    expect(wrapper.findComponent({ name: 'SiteSelector' }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'ResourceTree' }).exists()).toBe(true)
  })
})
