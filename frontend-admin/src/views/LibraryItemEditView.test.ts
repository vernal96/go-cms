// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { adminRequest } from '../api/admin-api'
import LibraryItemEditView from './LibraryItemEditView.vue'

const replaceMock = vi.fn()
vi.mock('../api/admin-api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../api/admin-api')>()),
  adminRequest: vi.fn(),
}))
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { siteId: '7', resourceId: '9' } }),
  useRouter: () => ({ replace: replaceMock }),
}))

const requestMock = vi.mocked(adminRequest)

describe('LibraryItemEditView', () => {
  beforeEach(() => {
    requestMock.mockReset()
    replaceMock.mockReset()
  })

  it('uses the Library default template and exposes no tree parent control', async () => {
    requestMock
      .mockResolvedValueOnce({
        types: [{ code: 'library', label: 'Библиотека', capabilities: { owns_library_items: true } }],
        templates: [{ code: 'article', label: 'Article', fields: [], supports_resource_widgets: false }],
        widgets: [], extensions: [],
      })
      .mockResolvedValueOnce({
        items: [
          { id: 9, type: 'library', display_title: 'Catalog' },
          { id: 10, type: 'page', display_title: 'Page' },
        ],
      })
      .mockResolvedValueOnce({
        resource: { id: 9, type_settings: { default_item_template: 'article' } },
      })
      .mockResolvedValueOnce({ item: { id: 101, library_id: 9 } })
    const wrapper = shallowMount(LibraryItemEditView, {
      props: { accessToken: 'token' },
      global: { renderStubDefaultSlot: true },
    })
    await flushPromises()

    const model = wrapper.findComponent({ name: 'ElForm' }).props('model') as Record<string, unknown>
    expect(model.template_code).toBe('article')
    expect(wrapper.text()).not.toContain('Родительский ресурс')
    Object.assign(model, { title: ' First item ', slug: 'first-item' })
    await wrapper.findComponent({ name: 'ElButton' }).trigger('click')
    await flushPromises()

    expect(requestMock).toHaveBeenNthCalledWith(
      4,
      '/api/sites/7/resources/9/items',
      'token',
      expect.objectContaining({ method: 'POST' }),
    )
    expect(replaceMock).toHaveBeenCalledWith('/admin/sites/7/resources/9/items/101/edit')
  })
})
