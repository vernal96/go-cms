// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { adminRequest, adminRequestVoid } from '../api/admin-api'
import { useSelectedSite } from '../composables/use-selected-site'
import ResourceTree from './ResourceTree.vue'

vi.mock('../api/admin-api', () => ({ adminRequest: vi.fn(), adminRequestVoid: vi.fn() }))
vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
    afterEach: vi.fn(() => vi.fn()),
    currentRoute: { value: { params: {} } },
  }),
}))

const requestMock = vi.mocked(adminRequest)
const requestVoidMock = vi.mocked(adminRequestVoid)

describe('ResourceTree', () => {
  beforeEach(() => {
    requestMock.mockReset()
    requestVoidMock.mockReset()
    useSelectedSite().reset()
    useSelectedSite().setSelected({ id: 7, domain: 'example.com' })
  })

  it('loads only one level and exposes an inline retry row for a failed child request', async () => {
    requestMock.mockResolvedValueOnce({
      items: [
        {
          id: 3,
          parent_id: null,
          template_code: 'page',
          icon: 'document',
          title: 'Home',
          menu_title: '',
          display_title: 'Home',
          sort: 0,
          deleted: false,
          published: true,
          deleted_at: null,
          has_children: false,
          can_create_child: true,
        },
      ],
      permissions: { create_root: true },
    })
    const wrapper = shallowMount(ResourceTree, {
      props: { accessToken: 'token', canCreate: true },
    })
    const load = wrapper.findComponent({ name: 'ElTree' }).props('load') as (
      node: { level: number; data: Record<string, unknown> },
      resolve: (rows: Array<Record<string, unknown>>) => void,
    ) => Promise<void>
    const rootResolve = vi.fn()

    await load({ level: 0, data: {} }, rootResolve)

    expect(requestMock).toHaveBeenCalledWith(
      '/api/sites/7/resources',
      'token',
    )
    expect(rootResolve).toHaveBeenCalledWith([
      expect.objectContaining({ id: 3, isLeaf: true }),
    ])

    requestMock.mockRejectedValueOnce(new Error('network down'))
    const childResolve = vi.fn()
    await load({ level: 1, data: { id: 3 } }, childResolve)

    expect(requestMock).toHaveBeenLastCalledWith(
      '/api/sites/7/resources?parent_id=3',
      'token',
    )
    expect(childResolve).toHaveBeenCalledWith([
      expect.objectContaining({
        loadError: true,
        retryParentId: 3,
        isLeaf: true,
      }),
    ])
  })

  it('fully remounts the lazy tree when the selected site changes', async () => {
    const wrapper = shallowMount(ResourceTree, {
      props: { accessToken: 'token', canCreate: false },
    })
    expect(wrapper.find('.resource-panel-header .resource-search').exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'ElButton' }).exists()).toBe(false)
    const initialKey = wrapper.findComponent({ name: 'ElTree' }).vm.$.vnode.key

    useSelectedSite().setSelected({ id: 8, domain: 'next.example.com' })
    await nextTick()
    await flushPromises()

    expect(wrapper.findComponent({ name: 'ElTree' }).vm.$.vnode.key).not.toBe(
      initialKey,
    )
    expect(wrapper.findComponent({ name: 'ElTree' }).vm.$.vnode.key).toContain(
      '8-',
    )

    await wrapper.setProps({ canCreate: true })
    expect(wrapper.findComponent({ name: 'ElButton' }).exists()).toBe(true)
    expect(wrapper.find('.resource-heading-add').exists()).toBe(true)
  })

  it('moves a resource into a folder as its last child', async () => {
    requestMock.mockResolvedValue({})
    const wrapper = shallowMount(ResourceTree, {
      props: { accessToken: 'token', canCreate: false, canUpdate: true },
    })
    const tree = wrapper.findComponent({ name: 'ElTree' })

    tree.vm.$emit(
      'node-drop',
      { data: { id: 3, deleted: false, loadError: false } },
      { data: { id: 8, deleted: false, loadError: false } },
      'inner',
    )
    await flushPromises()

    expect(requestMock).toHaveBeenCalledWith(
      '/api/sites/7/resources/3/move',
      'token',
      {
        method: 'POST',
        body: JSON.stringify({ parent_id: 8, position: 2_147_483_647 }),
      },
    )
  })

  it('shows restore actions and permanent delete for a deleted resource', async () => {
    const wrapper = shallowMount(ResourceTree, {
      props: { accessToken: 'token', canCreate: false, canDelete: true },
    })
    const tree = wrapper.findComponent({ name: 'ElTree' })
    const event = new MouseEvent('contextmenu', { clientX: 100, clientY: 100 })

    tree.vm.$emit('node-contextmenu', event, {
      id: 12,
      display_title: 'Удалённый ресурс',
      deleted: true,
      has_children: true,
      loadError: false,
    })
    await nextTick()

    const menu = wrapper.find('.resource-context-menu')
    expect(menu.exists()).toBe(true)
    expect(menu.text()).toContain('Восстановить')
    expect(menu.text()).toContain('Восстановить с потомками')
    expect(menu.text()).toContain('Удалить окончательно')
    expect(menu.find('.is-danger').exists()).toBe(true)
  })
})
