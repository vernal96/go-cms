// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { adminRequest } from '../api/admin-api'
import ResourceHistoryTab from './ResourceHistoryTab.vue'

vi.mock('../api/admin-api', async () => {
  const actual = await vi.importActual<typeof import('../api/admin-api')>('../api/admin-api')
  return { ...actual, adminRequest: vi.fn() }
})

const requestMock = vi.mocked(adminRequest)

describe('ResourceHistoryTab', () => {
  beforeEach(() => requestMock.mockReset())

  it('loads bounded metadata and hides destructive actions without permission', async () => {
    requestMock.mockResolvedValueOnce({ items: [{ id: 1, resource_id: 9, site_id: 7, version: 3, kind: 'updated', created_at: '2026-01-01T00:00:00Z', created_by_name: 'Editor' }], page: 1, per_page: 20, total: 1 })
    const wrapper = shallowMount(ResourceHistoryTab, { props: { accessToken: 'token', siteId: 7, resourceId: 9, resourceVersion: 3, canRestore: false, canDelete: false } })
    await flushPromises()
    expect(requestMock).toHaveBeenCalledWith('/api/sites/7/resources/9/revisions?page=1&per_page=20', 'token')
    expect(wrapper.findAllComponents({ name: 'ElButton' }).some((button) => button.text() === 'Очистить историю')).toBe(false)
  })
})
