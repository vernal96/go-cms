// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { ElMessageBox } from 'element-plus'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { adminRequest } from '../api/admin-api'
import ResourceEditView from './ResourceEditView.vue'

vi.mock('../api/admin-api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../api/admin-api')>()),
  adminRequest: vi.fn(),
}))
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { siteId: '7', resourceId: '9' } }),
  useRouter: () => ({ push: vi.fn() }),
}))

const requestMock = vi.mocked(adminRequest)

describe('ResourceEditView schema transitions', () => {
  beforeEach(() => {
    requestMock.mockReset()
    requestMock
      .mockResolvedValueOnce({
        resource: {
          id: 9,
          site_id: 7,
          parent_id: null,
          type: 'page',
          template_code: 'page',
          title: 'Page',
          menu_title: '',
          slug: '',
          path: '/',
          annotation: '',
          content_type: 'html',
          content: '',
          target_resource_id: null,
          external_url: null,
          is_public: true,
          is_searchable: true,
          in_menu: true,
          in_sitemap: true,
          sort: 0,
          published_at: null,
          unpublished_at: null,
          deleted: false,
          deleted_at: null,
			fields: { page_title: 'Old title' }, type_settings: { custom_option: { nested: true } },
					widgets: [],
        },
        permissions: { update: true, delete: true, restore: true },
      })
      .mockResolvedValueOnce({
        types: [
			{ code: 'page', label: 'Страница', capabilities: { supports_template: true, supports_content: true, supports_widgets: true, supports_fields: true, mutable_type: true }, settings_fields: [{ key: 'custom_option', type: 'json', label: 'Custom option', required: false, rules: [] }], settings_defaults: {}, content_types: [{ code: 'html', label: 'HTML', editor: 'html' }] },
			{ code: 'link', label: 'Ссылка', capabilities: { mutable_type: true }, settings_fields: [], settings_defaults: {}, content_types: [] },
        ],
        templates: [
          {
            code: 'page',
            label: 'Страница',
            icon: 'document',
				editor_tabs: [{ code: 'content', label: 'Контент', fields: ['page_title'] }],
				supports_resource_widgets: true,
					widget_areas: ['body', 'sidebar'],
            fields: [
              {
                key: 'page_title',
                type: 'string',
                label: 'Page title',
                required: true,
                rules: [],
              },
            ],
          },
          {
            code: 'landing',
            label: 'Лендинг',
            icon: 'document',
				editor_tabs: [
					{ code: 'content', label: 'Контент', fields: ['hero_title'] },
					{ code: 'layout', label: 'Макет', fields: ['columns'] },
				],
				supports_resource_widgets: false,
					widget_areas: [],
            fields: [
              {
                key: 'hero_title',
                type: 'string',
                label: 'Hero title',
                required: true,
                rules: [],
              },
              {
                key: 'columns',
                type: 'int',
                label: 'Columns',
                required: true,
                rules: [],
              },
            ],
          },
        ],
			widgets: [],
			extensions: [],
      })
      .mockResolvedValueOnce({ items: [] })
      .mockResolvedValueOnce({ site: { id: 7, domain: 'example.com' }, permissions: { read: true, create: true, update: true, delete: true } })
    vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as never)
  })

  it('clears incompatible settings only after confirmed template and type changes', async () => {
    const wrapper = shallowMount(ResourceEditView, {
      props: { accessToken: 'token', permissions: new Set<string>() },
      global: { renderStubDefaultSlot: true },
    })
    await flushPromises()
		expect(wrapper.findAllComponents({ name: 'ElTabPane' }).some(
			(tab) => String(tab.props('name')).startsWith('extension:'),
		)).toBe(false)
		expect(wrapper.findAllComponents({ name: 'ElTabPane' }).some(
			(tab) => tab.props('name') === 'widgets',
		)).toBe(true)
		expect(wrapper.findComponent({ name: 'TabbedDynamicFieldsForm' }).exists()).toBe(true)

    const selects = wrapper.findAllComponents({ name: 'ElSelect' })
    expect(selects).toHaveLength(3)
    selects[0]?.vm.$emit('change', 'landing')
    await flushPromises()

    const model = wrapper
      .findComponent({ name: 'ElForm' })
      .props('model') as Record<string, unknown>
    expect(model.template_code).toBe('landing')
    expect(model.fields).toEqual({ hero_title: '', columns: null })
		expect(wrapper.findAllComponents({ name: 'ElTabPane' }).some(
			(tab) => tab.props('name') === 'widgets',
		)).toBe(false)

    selects[2]?.vm.$emit('change', 'link')
    await flushPromises()
    expect(model.type).toBe('link')
    expect(model.template_code).toBeNull()
    expect(model.fields).toEqual({})
    expect(ElMessageBox.confirm).toHaveBeenCalledTimes(2)
  })

  it('round-trips unknown type settings for an extensible type', async () => {
    requestMock.mockResolvedValueOnce({
      resource: { version: 2, slug: '', sort: 0, fields: { page_title: 'Old title' } },
    })
    const wrapper = shallowMount(ResourceEditView, {
      props: { accessToken: 'token', permissions: new Set<string>() },
      global: { renderStubDefaultSlot: true },
    })
    await flushPromises()
    const model = wrapper.findComponent({ name: 'ElForm' }).props('model') as Record<string, any>
    expect(model.type_settings).toEqual({ custom_option: { nested: true } })
    const save = wrapper.findAllComponents({ name: 'ElButton' }).find((button) => button.text() === 'Сохранить')
    save?.vm.$emit('click')
    await flushPromises()
    const updateCall = requestMock.mock.calls.find((call) => call[0] === '/api/sites/7/resources/9' && call[2]?.method === 'PATCH')
    const payload = JSON.parse(String((updateCall?.[2] as RequestInit).body))
    expect(payload.type_settings).toEqual({ custom_option: { nested: true } })
  })

	it('shows profile extensions only while they apply to the resource type', async () => {
		requestMock.mockReset()
		requestMock
			.mockResolvedValueOnce({
				resource: {
					id: 9, site_id: 7, parent_id: null, type: 'page', template_code: 'page',
					title: 'Page', menu_title: '', slug: '', path: '/', annotation: '',
					content_type: 'html', content: '', external_url: null, is_public: true,
					is_searchable: true, in_menu: true, in_sitemap: true, sort: 0,
					published_at: null, unpublished_at: null, deleted: false, deleted_at: null,
					fields: {}, type_settings: {},
					widgets: [],
				},
				permissions: { update: true, delete: true, restore: true },
			})
			.mockResolvedValueOnce({
				types: [
					{ code: 'page', label: 'Страница', capabilities: { supports_template: true, supports_content: true, supports_fields: true, mutable_type: true } },
					{ code: 'link', label: 'Ссылка', capabilities: { mutable_type: true } },
				],
				templates: [{ code: 'page', label: 'Страница', icon: 'document', fields: [], supports_resource_widgets: false, widget_areas: [] }],
				widgets: [],
				extensions: [{
					code: 'seo', title: 'SEO', applies_to: ['page'], fields: [], variables: [],
				}],
			})
			.mockResolvedValueOnce({ items: [] })
			.mockResolvedValueOnce({ site: { id: 7, domain: 'example.com' }, permissions: { read: true, create: true, update: true, delete: true } })
		const wrapper = shallowMount(ResourceEditView, {
			props: { accessToken: 'token', permissions: new Set<string>() },
			global: { renderStubDefaultSlot: true },
		})
		await flushPromises()
		expect(wrapper.findAllComponents({ name: 'ElTabPane' }).some(
			(tab) => tab.props('name') === 'extension:seo',
		)).toBe(true)

		wrapper.findAllComponents({ name: 'ElSelect' })[2]?.vm.$emit('change', 'link')
		await flushPromises()
		expect(wrapper.findAllComponents({ name: 'ElTabPane' }).some(
			(tab) => tab.props('name') === 'extension:seo',
		)).toBe(false)
	})
})
