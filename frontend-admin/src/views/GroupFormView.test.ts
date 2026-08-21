// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { adminRequest } from '../api/admin-api'
import type { GroupSiteAccess } from '../types/admin'
import GroupFormView from './GroupFormView.vue'

vi.mock('../api/admin-api', async (importOriginal) => {
  const original = await importOriginal<typeof import('../api/admin-api')>()
  return { ...original, adminRequest: vi.fn() }
})

const requestMock = vi.mocked(adminRequest)

describe('GroupFormView', () => {
  beforeEach(() => {
    requestMock.mockImplementation(async (url) => {
      if (url === '/api/admin/permission-catalog') {
        return {
          can_manage: true,
          items: [
            { code: 'core.site.create', module: 'core', entity: 'site', action: 'create' },
            { code: 'core.resource.read', module: 'core', entity: 'resource', action: 'read' },
          ],
        }
      }
      if (url.startsWith('/api/admin/sites?')) {
        return {
          items: [{ id: 7, domain: 'example.test', profile_code: 'dev', locale: 'ru-RU', settings: {}, is_public: false, capabilities: { view: true, edit: true, delete: true } }],
          pagination: { page: 1, per_page: 10, total: 1 },
          permissions: { read: true, create: true, update: true, delete: true },
        }
      }
      throw new Error(`unexpected request ${url}`)
    })
  })

  afterEach(() => vi.clearAllMocks())

  it('renders permission tabs and enforces the site capability hierarchy', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/admin/groups/create', component: GroupFormView },
        { path: '/admin/groups', component: { template: '<div />' } },
      ],
    })
    await router.push('/admin/groups/create')
    await router.isReady()
    const wrapper = shallowMount(GroupFormView, {
      props: { accessToken: 'token', permissions: new Set(['core.group.create']) },
      global: {
		plugins: [router],
		stubs: {
		  ElCard: { template: '<div><slot /></div>' },
		  ElTabs: { template: '<div><slot /></div>' },
		  ElTabPane: { props: ['label', 'name'], template: '<section class="tab-pane-stub" :data-label="label"><slot /></section>' },
		  ElTableColumn: { template: '<div />' },
		},
	  },
    })
    await flushPromises()

    const labels = wrapper.findAll('.tab-pane-stub').map((tab) => tab.attributes('data-label'))
    expect(labels).toEqual(['Общие права', 'Доступ к сайтам'])

    const vm = wrapper.vm as unknown as {
      updateSiteAccess: (siteId: number, capability: 'view' | 'edit' | 'delete', enabled: boolean) => void
      siteGrant: (siteId: number) => GroupSiteAccess
    }
    vm.updateSiteAccess(7, 'delete', true)
    expect(vm.siteGrant(7)).toMatchObject({ can_view: true, can_edit: true, can_delete: true })
    vm.updateSiteAccess(7, 'edit', false)
    expect(vm.siteGrant(7)).toMatchObject({ can_view: true, can_edit: false, can_delete: false })
    vm.updateSiteAccess(7, 'view', false)
    expect(vm.siteGrant(7)).toMatchObject({ can_view: false, can_edit: false, can_delete: false })
  })
})
