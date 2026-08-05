// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { adminRequest } from '../api/admin-api'
import { useSelectedSite } from '../composables/use-selected-site'
import ResourceTree from './ResourceTree.vue'

vi.mock('../api/admin-api', () => ({ adminRequest: vi.fn() }))

const requestMock = vi.mocked(adminRequest)

describe('ResourceTree', () => {
  beforeEach(() => {
    requestMock.mockReset()
    useSelectedSite().reset()
    useSelectedSite().setSelected({ id: 7, domain: 'example.com' })
  })

  it('loads only one level and exposes an inline retry row for a failed child request', async () => {
    requestMock.mockResolvedValueOnce({
      items: [{
        id: 3,
        parent_id: null,
        template_code: 'page',
        icon: 'document',
        title: 'Home',
        menu_title: '',
        display_title: 'Home',
        has_children: false,
        can_create_child: true,
      }],
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
      '/api/admin/sites/7/resources',
      'token',
    )
    expect(rootResolve).toHaveBeenCalledWith([
      expect.objectContaining({ id: 3, isLeaf: true }),
    ])

    requestMock.mockRejectedValueOnce(new Error('network down'))
    const childResolve = vi.fn()
    await load({ level: 1, data: { id: 3 } }, childResolve)

    expect(requestMock).toHaveBeenLastCalledWith(
      '/api/admin/sites/7/resources?parent_id=3',
      'token',
    )
    expect(childResolve).toHaveBeenCalledWith([
      expect.objectContaining({ loadError: true, retryParentId: 3, isLeaf: true }),
    ])
  })

  it('fully remounts the lazy tree when the selected site changes', async () => {
    const wrapper = shallowMount(ResourceTree, {
      props: { accessToken: 'token', canCreate: false },
    })
    expect(wrapper.findComponent({ name: 'ElButton' }).exists()).toBe(false)
    const initialKey = wrapper.findComponent({ name: 'ElTree' }).vm.$.vnode.key

    useSelectedSite().setSelected({ id: 8, domain: 'next.example.com' })
    await nextTick()
    await flushPromises()

    expect(wrapper.findComponent({ name: 'ElTree' }).vm.$.vnode.key).not.toBe(initialKey)
    expect(wrapper.findComponent({ name: 'ElTree' }).vm.$.vnode.key).toContain('8-')

    await wrapper.setProps({ canCreate: true })
    expect(wrapper.findComponent({ name: 'ElButton' }).exists()).toBe(true)
  })
})
