// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { adminRequest } from '../api/admin-api'
import { useSelectedSite } from '../composables/use-selected-site'
import SiteSelector from './SiteSelector.vue'

vi.mock('../api/admin-api', () => ({ adminRequest: vi.fn() }))

const requestMock = vi.mocked(adminRequest)

function response(domain: string) {
  return {
    items: [{ id: domain === 'new.example.com' ? 2 : 1, domain }],
    pagination: { page: 1, per_page: 10, total: 1 },
  }
}

describe('SiteSelector', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    requestMock.mockReset()
    useSelectedSite().reset()
    useSelectedSite().clearSelected()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('debounces remote search and ignores a stale response', async () => {
    const pending: Array<(value: ReturnType<typeof response>) => void> = []
    requestMock.mockImplementation(() => new Promise((resolve) => pending.push(resolve)))
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }],
    })
    await router.push('/')
    await router.isReady()
    const wrapper = shallowMount(SiteSelector, {
      props: { accessToken: 'token', canCreate: false },
      global: { plugins: [router], renderStubDefaultSlot: true },
    })
    const select = wrapper.findComponent({ name: 'ElSelect' })
    const remoteSearch = select.props('remoteMethod') as (value: string) => void

    remoteSearch('old')
    vi.advanceTimersByTime(299)
    expect(requestMock).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(requestMock).toHaveBeenCalledTimes(1)

    remoteSearch('new')
    vi.advanceTimersByTime(300)
    expect(requestMock).toHaveBeenCalledTimes(2)
    expect(requestMock.mock.calls[1]?.[0]).toContain('search=new')

    pending[1]?.(response('new.example.com'))
    await flushPromises()
    pending[0]?.(response('old.example.com'))
    await flushPromises()

    const labels = wrapper.findAllComponents({ name: 'ElOption' }).map((option) => option.props('label'))
    expect(labels).toEqual(['new.example.com'])
  })
})
