// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import { AdminAPIError, adminRequest } from '../../api/admin-api'
import type { ResourceExtensionMetadata } from '../../types/admin'
import ResourceExtensionEditor from './ResourceExtensionEditor.vue'

vi.mock('../../api/admin-api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/admin-api')>()),
  adminRequest: vi.fn(),
}))

const requestMock = vi.mocked(adminRequest)
const metadata: ResourceExtensionMetadata = {
  code: 'seo',
  title: 'SEO',
  applies_to: ['page'],
  fields: [
    { key: 'title_template', label: 'Title', control: 'text' },
    { key: 'robots_index', label: 'Index', control: 'switch' },
    { key: 'og_title', label: 'OpenGraph title', control: 'text' },
  ],
  variables: [{ code: '{{ resource.title }}', label: 'resource.title' }],
}
const settings = {
  title_template: '{{ resource.title }}',
  robots_index: true,
  og_title: '',
}

function mountEditor(canUpdate = true) {
  return shallowMount(ResourceExtensionEditor, {
    props: {
      metadata,
      siteId: 7,
      resourceId: 9,
      accessToken: 'token',
      canUpdate,
    },
    global: { renderStubDefaultSlot: true },
  })
}

describe('ResourceExtensionEditor', () => {
  it('opens SEO on the main tab and keeps OpenGraph in its own tab', async () => {
    requestMock.mockReset()
    requestMock.mockResolvedValueOnce(settings)
    const wrapper = mountEditor()
    await flushPromises()

    expect(wrapper.findComponent({ name: 'ElTabs' }).props('modelValue')).toBe('general')
    expect(wrapper.findAllComponents({ name: 'ElTabPane' }).map((tab) => ({
      label: tab.props('label'),
      name: tab.props('name'),
    }))).toEqual([
      { label: 'Основные', name: 'general' },
      { label: 'OpenGraph', name: 'opengraph' },
    ])
  })

  it('loads, previews, and saves through separate extension requests', async () => {
    requestMock.mockReset()
    requestMock
      .mockResolvedValueOnce(settings)
      .mockResolvedValueOnce({
        title: 'Page',
        description: '',
        keywords: [],
        canonical_url: '',
        robots: { index: false, follow: true },
        open_graph: { title: 'Page', description: '' },
        warnings: [],
        title_characters: 4,
        description_characters: 0,
      })
      .mockResolvedValueOnce(settings)
    const wrapper = mountEditor()
    await flushPromises()

    expect(requestMock).toHaveBeenNthCalledWith(
      1,
      '/api/admin/sites/7/resources/9/extensions/seo',
      'token',
    )
    const buttons = wrapper.findAllComponents({ name: 'ElButton' })
    await buttons[0]?.trigger('click')
    await flushPromises()
    expect(requestMock).toHaveBeenNthCalledWith(
      2,
      '/api/admin/sites/7/resources/9/extensions/seo/preview',
      'token',
      expect.objectContaining({ method: 'POST' }),
    )
    expect(wrapper.findComponent({ name: 'ElDescriptions' }).exists()).toBe(true)

    await buttons[1]?.trigger('click')
    await flushPromises()
    expect(requestMock).toHaveBeenNthCalledWith(
      3,
      '/api/admin/sites/7/resources/9/extensions/seo',
      'token',
      expect.objectContaining({ method: 'PATCH' }),
    )
  })

  it('keeps inputs and save read-only while allowing preview', async () => {
    requestMock.mockReset()
    requestMock.mockResolvedValueOnce(settings)
    const wrapper = mountEditor(false)
    await flushPromises()

    const buttons = wrapper.findAllComponents({ name: 'ElButton' })
    expect(buttons[0]?.props('disabled')).not.toBe(true)
    expect(buttons[1]?.props('disabled')).toBe(true)
    expect(wrapper.findComponent({ name: 'ElInput' }).props('disabled')).toBe(true)
    expect(wrapper.findComponent({ name: 'ElSwitch' }).props('disabled')).toBe(true)
  })

  it('shows backend template validation on the matching field', async () => {
    requestMock.mockReset()
    requestMock
      .mockResolvedValueOnce(settings)
      .mockRejectedValueOnce(new AdminAPIError(
        422,
        'validation_failed',
        'SEO template validation failed',
        [{ key: 'title_template', rule: 'extension', param: 'unknown variable' }],
      ))
    const wrapper = mountEditor()
    await flushPromises()

    await wrapper.findAllComponents({ name: 'ElButton' })[1]?.trigger('click')
    await flushPromises()
    const titleField = wrapper.findAllComponents({ name: 'ElFormItem' })
      .find((field) => field.props('label') === 'Title')
    expect(titleField?.props('error')).toBe('unknown variable')
    expect(wrapper.findComponent({ name: 'ElAlert' }).props('title')).toBe(
      'SEO template validation failed',
    )
  })
})
