// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { adminRequest } from '../api/admin-api'
import type { Resource, ResourceTreeItem } from '../types/admin'
import ResourceSiteTransferDialog from './ResourceSiteTransferDialog.vue'

vi.mock('../api/admin-api', () => ({ adminRequest: vi.fn() }))

const requestMock = vi.mocked(adminRequest)
const source = {
  id: 9,
  version: 3,
  display_title: 'Раздел',
} as ResourceTreeItem
const transferred = {
  id: 9,
  site_id: 8,
  version: 4,
} as Resource

describe('ResourceSiteTransferDialog', () => {
  beforeEach(() => requestMock.mockReset())

  it('excludes the current site and sends target id with the expected version', async () => {
    requestMock
      .mockResolvedValueOnce({
        items: [{ id: 8, domain: 'target.example' }],
        pagination: { page: 1, per_page: 10, total: 1 },
      })
      .mockResolvedValueOnce(transferred)
    const wrapper = shallowMount(ResourceSiteTransferDialog, {
      props: { accessToken: 'token', sourceSiteId: 7 },
    })

    ;(wrapper.vm as unknown as { open: (item: ResourceTreeItem) => void }).open(source)
    await flushPromises()
    expect(requestMock).toHaveBeenNthCalledWith(
      1,
      '/api/sites/options?search=&page=1&per_page=10&exclude_id=7',
      'token',
      { signal: expect.any(AbortSignal) },
    )

    const state = (wrapper.vm.$ as unknown as { setupState: unknown }).setupState as {
      targetID: number | null
      transfer: () => Promise<void>
    }
    state.targetID = 8
    await state.transfer()

    expect(requestMock).toHaveBeenNthCalledWith(
      2,
      '/api/sites/7/resources/9/transfer',
      'token',
      {
        method: 'POST',
        body: JSON.stringify({ target_site_id: 8, expected_version: 3 }),
      },
    )
    expect(wrapper.emitted('transferred')?.[0]?.[0]).toEqual({
      resource: transferred,
      source,
      target: { id: 8, domain: 'target.example' },
    })
  })

  it('keeps the dialog open and shows an API error', async () => {
    requestMock
      .mockResolvedValueOnce({
        items: [{ id: 8, domain: 'target.example' }],
        pagination: { page: 1, per_page: 10, total: 1 },
      })
      .mockRejectedValueOnce(new Error('Есть ссылки между сайтами'))
    const wrapper = shallowMount(ResourceSiteTransferDialog, {
      props: { accessToken: 'token', sourceSiteId: 7 },
    })
    ;(wrapper.vm as unknown as { open: (item: ResourceTreeItem) => void }).open(source)
    await flushPromises()
    const state = (wrapper.vm.$ as unknown as { setupState: unknown }).setupState as {
      targetID: number | null
      visible: boolean
      errorMessage: string
      transfer: () => Promise<void>
    }
    state.targetID = 8
    await state.transfer()

    expect(state.visible).toBe(true)
    expect(state.errorMessage).toBe('Есть ссылки между сайтами')
  })
})
