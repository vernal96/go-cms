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
        {
          code: 'dev',
          name: 'Development',
          fields: [
            {
              key: 'title',
              type: 'string',
              label: 'Title',
              required: true,
              rules: ['min=2'],
            },
          ],
			editor_tabs: [
				{ code: 'main', label: 'Main', fields: ['title'] },
			],
        },
      ],
    })
  })

  it('initializes and submits settings from profile metadata', async () => {
    const wrapper = shallowMount(SiteForm, {
      props: { accessToken: 'token' },
      global: { renderStubDefaultSlot: true },
    })
    await flushPromises()

    const formComponent = wrapper.findComponent({ name: 'ElForm' })
    const model = formComponent.props('model') as Record<string, unknown>
    Object.assign(model, {
      domain: ' example.com ',
      profile_code: 'dev',
      locale: ' ru-RU ',
      is_public: true,
      settings: { title: 'Demo' },
    })
    formComponent.vm.$emit('submit', new Event('submit'))
    await flushPromises()

    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({
      domain: 'example.com',
      profile_code: 'dev',
      locale: 'ru-RU',
      is_public: true,
      settings: { title: 'Demo' },
    })
  })

	it('uses tabbed settings only while editing a site', async () => {
		const createWrapper = shallowMount(SiteForm, {
			props: { accessToken: 'token' },
			global: { renderStubDefaultSlot: true },
		})
		await flushPromises()
		expect(createWrapper.findComponent({ name: 'TabbedDynamicFieldsForm' }).exists()).toBe(false)
		expect(createWrapper.findComponent({ name: 'DynamicFieldsForm' }).exists()).toBe(true)

		const editWrapper = shallowMount(SiteForm, {
			props: { accessToken: 'token', editing: true },
			global: { renderStubDefaultSlot: true },
		})
		await flushPromises()
		expect(editWrapper.findComponent({ name: 'TabbedDynamicFieldsForm' }).exists()).toBe(true)
		expect(editWrapper.findComponent({ name: 'DynamicFieldsForm' }).exists()).toBe(false)
	})
})
