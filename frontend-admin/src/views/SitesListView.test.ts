// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedSite } from '../composables/use-selected-site'
import SitesListView from './SitesListView.vue'

function jsonResponse(payload: unknown): Response {
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

function listResponse(page: number, items: Array<Record<string, unknown>>) {
  return {
    items,
    pagination: { page, per_page: 10, total: page === 2 ? 11 : items.length },
    permissions: { read: true, create: true, update: true, delete: true },
  }
}

describe('SitesListView', () => {
  beforeEach(() => {
    useSelectedSite().reset()
    useSelectedSite().setSelected({ id: 7, domain: 'selected.example.com' })
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('moves back one page and clears selection after deleting its last selected row', async () => {
    vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as never)
    vi.spyOn(ElMessage, 'success').mockImplementation(() => ({ close: () => undefined }))
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (init?.method === 'DELETE') return new Response(null, { status: 204 })
      if (url.includes('page=2')) {
        return jsonResponse(listResponse(2, [{
          id: 7,
          domain: 'selected.example.com',
          profile_code: 'dev',
          locale: 'ru-RU',
          settings: {},
          is_public: false,
		  capabilities: { view: true, edit: true, delete: true },
        }]))
      }
      return jsonResponse(listResponse(1, [{
        id: 1,
        domain: 'first.example.com',
        profile_code: 'dev',
        locale: 'ru-RU',
        settings: {},
        is_public: false,
		capabilities: { view: true, edit: true, delete: true },
      }]))
    })
    vi.stubGlobal('fetch', fetchMock)
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }],
    })
    await router.push('/')
    await router.isReady()
    const wrapper = shallowMount(SitesListView, {
      props: {
        accessToken: 'token',
        permissions: new Set(['core.site.create']),
      },
      global: { plugins: [router] },
    })
    await flushPromises()
    const table = wrapper.findComponent({ name: 'AdminDataTable' })

    table.vm.$emit('page-change', 2)
    await flushPromises()
    table.vm.$emit('action', 'delete', {
      id: 7,
      domain: 'selected.example.com',
      profile_code: 'dev',
      locale: 'ru-RU',
      settings: {},
      is_public: false,
	  capabilities: { view: true, edit: true, delete: true },
    })
    await flushPromises()

    expect(useSelectedSite().selectedSite.value).toBeNull()
    const calls = fetchMock.mock.calls.map(([input, init]) => ({ url: String(input), method: init?.method }))
    expect(calls).toContainEqual({ url: '/api/sites/7', method: 'DELETE' })
    expect(calls[calls.length - 1]?.url).toContain('page=1')
  })

  it('uses per-site capabilities to hide edit and delete actions', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(listResponse(1, [{
	  id: 3,
	  domain: 'view-only.example.com',
	  profile_code: 'dev',
	  locale: 'ru-RU',
	  settings: {},
	  is_public: false,
	  capabilities: { view: true, edit: false, delete: false },
	}]))))
	const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
	await router.push('/')
	await router.isReady()
	const wrapper = shallowMount(SitesListView, {
	  props: { accessToken: 'token', permissions: new Set(['core.site.update', 'core.site.delete']) },
	  global: { plugins: [router] },
	})
	await flushPromises()
	const actions = wrapper.findComponent({ name: 'AdminDataTable' }).props('actions') as Array<{ visible?: (row: Record<string, unknown>) => boolean }>
	const row = { capabilities: { view: true, edit: false, delete: false } }
	expect(actions.map((action) => action.visible?.(row))).toEqual([false, false])
  })
})
