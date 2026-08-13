// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it, vi } from 'vitest'

import DashboardView from './DashboardView.vue'

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

async function mountView(payload: unknown, permissions: string[] = [], status = 200) {
  const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) =>
    jsonResponse(payload, status))
  vi.stubGlobal('fetch', fetchMock)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/admin/sites', component: { template: '<div />' } },
    ],
  })
  await router.push('/')
  await router.isReady()
  const wrapper = shallowMount(DashboardView, {
    props: { accessToken: 'token', permissions: new Set(permissions) },
    global: {
      plugins: [router],
      stubs: {
        ElCard: {
          name: 'ElCard',
          template: '<div><slot /></div>',
        },
        ElEmpty: {
          name: 'ElEmpty',
          props: ['description'],
          template: '<div>{{ description }}</div>',
        },
        ElTable: {
          name: 'ElTable',
          props: ['data'],
          template: '<div><slot /></div>',
        },
        ElTableColumn: {
          name: 'ElTableColumn',
          template: '<div />',
        },
      },
    },
  })
  await flushPromises()
  return { wrapper, fetchMock, router }
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('DashboardView', () => {
  it('renders available cards, site rows and resource counts', async () => {
    const { wrapper, fetchMock, router } = await mountView({
      sites: {
        total: 2,
        public: 1,
        private: 1,
        items: [{ id: 1, domain: 'example.test', is_public: true, resource_count: 7 }],
      },
      resources: { total: 7 },
      users: { total: 5, active: 4, blocked: 1 },
      groups: { total: 3 },
    }, ['core.site.read'])

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/admin/dashboard')
    const headers = fetchMock.mock.calls[0]?.[1]?.headers as Headers
    expect(headers.get('Authorization')).toBe('Bearer token')
    expect(wrapper.find('[data-testid="dashboard-sites-card"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="dashboard-resources-card"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="dashboard-users-card"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="dashboard-groups-card"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="resource-count-column"]').exists()).toBe(true)
    const table = wrapper.findComponent({ name: 'ElTable' })
    expect(table.props('data')).toEqual([
      { id: 1, domain: 'example.test', is_public: true, resource_count: 7 },
    ])
    expect(wrapper.text()).toContain('Активных: 4')
    await wrapper.findComponent({ name: 'ElButton' }).trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/admin/sites')
  })

  it('omits unavailable cards and resource count column', async () => {
    const { wrapper } = await mountView({
      sites: {
        total: 1,
        public: 0,
        private: 1,
        items: [{ id: 1, domain: 'private.test', is_public: false }],
      },
    }, ['core.site.read'])

    expect(wrapper.find('[data-testid="dashboard-sites-card"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="dashboard-resources-card"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="dashboard-users-card"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="dashboard-groups-card"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="resource-count-column"]').exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'ElTable' }).props('data')).toEqual([
      { id: 1, domain: 'private.test', is_public: false },
    ])
  })

  it('shows an empty state when the API returns no permitted sections', async () => {
    const { wrapper } = await mountView({})
    expect(wrapper.text()).toContain('Нет доступных показателей')
  })

  it('shows an error state when loading fails', async () => {
    const { wrapper } = await mountView({
      error: { code: 'internal_error', message: 'operation failed' },
    }, [], 500)
    expect(wrapper.findComponent({ name: 'ElResult' }).exists()).toBe(true)
  })
})
