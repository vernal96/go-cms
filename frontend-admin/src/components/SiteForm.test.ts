// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { adminRequest } from '../api/admin-api'
import SiteForm from './SiteForm.vue'

vi.mock('../api/admin-api', () => ({ adminRequest: vi.fn() }))

const requestMock = vi.mocked(adminRequest)

describe('SiteForm', () => {
  beforeEach(() => {
    requestMock.mockReset()
    requestMock.mockResolvedValue({
      items: [
        { code: 'required', name: 'Required settings', creatable: false },
        { code: 'dev', name: 'Development', creatable: true },
      ],
    })
  })

  it('disables non-creatable profiles and submits only the shared site fields', async () => {
    const wrapper = shallowMount(SiteForm, {
      props: { accessToken: 'token' },
      global: { renderStubDefaultSlot: true },
    })
    await flushPromises()

    const options = wrapper.findAllComponents({ name: 'ElOption' })
    expect(options.map((option) => option.props('disabled'))).toEqual([true, false])

    const formComponent = wrapper.findComponent({ name: 'ElForm' })
    const model = formComponent.props('model') as Record<string, unknown>
    Object.assign(model, {
      domain: ' example.com ',
      profile_code: 'dev',
      locale: ' ru-RU ',
      is_public: true,
    })
    formComponent.vm.$emit('submit', new Event('submit'))
    await flushPromises()

    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({
      domain: 'example.com',
      profile_code: 'dev',
      locale: 'ru-RU',
      is_public: true,
    })
    expect(model).not.toHaveProperty('settings')
  })
})
