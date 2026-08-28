// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { adminRequest } from '../api/admin-api'
import LibraryItemEditView from './LibraryItemEditView.vue'

const replaceMock = vi.fn()
const routeParams: { siteId: string; resourceId: string; itemId?: string } = { siteId: '7', resourceId: '9' }
vi.mock('../api/admin-api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../api/admin-api')>()),
  adminRequest: vi.fn(),
}))
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: routeParams }),
  useRouter: () => ({ replace: replaceMock }),
}))

const requestMock = vi.mocked(adminRequest)

describe('LibraryItemEditView', () => {
  beforeEach(() => {
    requestMock.mockReset()
    replaceMock.mockReset()
    delete routeParams.itemId
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

    expect(wrapper.getComponent({ name: 'ElTabs' }).props('modelValue')).toBe('main')
    expect(wrapper.findAllComponents({ name: 'ElTabPane' }).map((tab) => tab.props('label'))).toEqual(['Основное', 'Настройки'])
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

  it('uses the same ordered conditional tabs as an ordinary resource editor', async () => {
    routeParams.itemId = '101'
    requestMock
      .mockResolvedValueOnce({
        types: [{ code: 'library', label: 'Библиотека', capabilities: { owns_library_items: true } }],
        templates: [{
          code: 'article', label: 'Article', supports_resource_widgets: true,
          fields: [{ key: 'subtitle', type: 'string', label: 'Subtitle', required: false, rules: [] }],
			editor_tabs: [{ code: 'content', label: 'Контент', fields: ['subtitle'] }],
        }],
        widgets: [],
        extensions: [{ code: 'seo', title: 'SEO', applies_to: ['page'], fields: [], variables: [] }],
      })
      .mockResolvedValueOnce({ items: [{ id: 9, type: 'library', display_title: 'Catalog' }] })
      .mockResolvedValueOnce({ resource: { id: 9, type_settings: {} } })
      .mockResolvedValueOnce({
        item: { id: 101, library_id: 9, version: 3, template_code: 'article', title: 'Item', slug: 'item', annotation: '', content: '', is_public: true, is_searchable: true, published_at: null, unpublished_at: null, fields: {}, widgets: [] },
        permissions: { update: true, history_read: true, history_delete: true },
      })
    const wrapper = shallowMount(LibraryItemEditView, {
      props: { accessToken: 'token' },
      global: { renderStubDefaultSlot: true },
    })
    await flushPromises()

    expect(wrapper.getComponent({ name: 'ElTabs' }).props('modelValue')).toBe('main')
		expect(wrapper.findComponent({ name: 'TabbedDynamicFieldsForm' }).exists()).toBe(false)
		expect(wrapper.findComponent({ name: 'DynamicFieldsForm' }).exists()).toBe(true)
    expect(wrapper.findAllComponents({ name: 'ElTabPane' }).map((tab) => tab.props('label'))).toEqual([
      'Основное', 'Виджеты', 'Настройки', 'Параметры полей', 'История', 'SEO',
    ])
  })

  it('keeps Save and Move independent and updates ownership after Move', async () => {
    routeParams.itemId = '101'
    requestMock
      .mockResolvedValueOnce({
        types: [{ code: 'library', label: 'Библиотека', capabilities: { owns_library_items: true } }],
        templates: [{ code: 'article', label: 'Article', fields: [], supports_resource_widgets: false }],
        widgets: [], extensions: [],
      })
      .mockResolvedValueOnce({ items: [
        { id: 9, type: 'library', display_title: 'Catalog' },
        { id: 11, type: 'library', display_title: 'Archive' },
      ] })
      .mockResolvedValueOnce({ resource: { id: 9, type_settings: {} } })
      .mockResolvedValueOnce({
        item: { id: 101, library_id: 9, version: 3, template_code: 'article', title: 'Item', slug: 'item', annotation: '', content: '', is_public: true, is_searchable: true, published_at: null, unpublished_at: null, fields: {}, widgets: [] },
        permissions: { update: true, history_read: false, history_delete: false },
      })
      .mockResolvedValueOnce({ item: { id: 101, library_id: 9, version: 4, fields: {} } })
      .mockResolvedValueOnce({ item: { id: 101, library_id: 11, version: 5 } })
      .mockResolvedValueOnce({ item: { id: 101, library_id: 11, version: 6, fields: {} } })
    const wrapper = shallowMount(LibraryItemEditView, {
      props: { accessToken: 'token' },
      global: { renderStubDefaultSlot: true },
    })
    await flushPromises()
    const model = wrapper.findComponent({ name: 'ElForm' }).props('model') as Record<string, unknown>
    model.library_id = 11
    const buttons = () => wrapper.findAllComponents({ name: 'ElButton' })
    await buttons().find((button) => button.text() === 'Сохранить')!.trigger('click')
    await flushPromises()
    expect(requestMock.mock.calls.some(([url]) => String(url).endsWith('/move'))).toBe(false)

    await buttons().find((button) => button.text() === 'Переместить')!.trigger('click')
    await flushPromises()
    expect(requestMock).toHaveBeenCalledWith(
      '/api/sites/7/library-items/101/move',
      'token',
      expect.objectContaining({ method: 'POST', body: JSON.stringify({ library_id: 11, expected_version: 4 }) }),
    )
    expect(replaceMock).toHaveBeenCalledWith('/admin/sites/7/resources/11/items/101/edit')

    await buttons().find((button) => button.text() === 'Сохранить')!.trigger('click')
    await flushPromises()
    expect(requestMock.mock.calls.filter(([url]) => String(url).endsWith('/move'))).toHaveLength(1)
  })
})
