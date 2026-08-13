// @vitest-environment jsdom

import { shallowMount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it, vi } from 'vitest'

import UsersListView from './UsersListView.vue'

describe('UsersListView', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('shows access denied and does not request data without core.user.read', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
    await router.push('/')
    await router.isReady()
    const wrapper = shallowMount(UsersListView, {
      props: { accessToken: 'token', permissions: new Set<string>() },
      global: { plugins: [router] },
    })

    expect(wrapper.findComponent({ name: 'AccessDeniedView' }).exists()).toBe(true)
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
