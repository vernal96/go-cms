// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AdminAPIError, adminRequest } from '../api/admin-api'
import AdministrationView from './AdministrationView.vue'

vi.mock('../api/admin-api', async () => {
  const actual = await vi.importActual<typeof import('../api/admin-api')>('../api/admin-api')
  return { ...actual, adminRequest: vi.fn() }
})

const requestMock = vi.mocked(adminRequest)

describe('AdministrationView', () => {
  beforeEach(() => requestMock.mockReset())

  it('requires DELETE and refreshes the count after purge', async () => {
    requestMock.mockResolvedValueOnce({ count: 12 }).mockResolvedValueOnce({ count: 12 }).mockResolvedValueOnce({ count: 0 })
    const wrapper = mount(AdministrationView, { props: { accessToken: 'token' } })
    await flushPromises()
    expect(wrapper.text()).toContain('Всего ревизий: 12')
    const button = wrapper.findComponent({ name: 'ElButton' })
    expect(button.props('disabled')).toBe(true)
    await wrapper.find('input').setValue('DELETE')
    await button.trigger('click')
    await flushPromises()
    expect(requestMock).toHaveBeenNthCalledWith(2, '/api/administration/resource-revisions', 'token', { method: 'DELETE' })
    expect(requestMock).toHaveBeenCalledTimes(3)
    expect(wrapper.text()).toContain('Всего ревизий: 0')
  })

  it('shows the protected-boundary message on direct forbidden access', async () => {
    requestMock.mockRejectedValueOnce(new AdminAPIError(403, 'forbidden', 'Forbidden'))
    const wrapper = mount(AdministrationView, { props: { accessToken: 'token' } })
    await flushPromises()
    expect(wrapper.text()).toContain('только участникам встроенной группы администраторов')
  })
})
