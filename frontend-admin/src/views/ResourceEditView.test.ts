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
          settings: { page_title: 'Old title' },
        },
        permissions: { update: true, delete: true, restore: true },
      })
      .mockResolvedValueOnce({
        types: [
          { code: 'page', label: 'Страница' },
          { code: 'link', label: 'Ссылка' },
        ],
        templates: [
          {
            code: 'page',
            label: 'Страница',
            icon: 'document',
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
			extensions: [],
      })
      .mockResolvedValueOnce({ items: [] })
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

    const selects = wrapper.findAllComponents({ name: 'ElSelect' })
    expect(selects).toHaveLength(3)
    selects[0]?.vm.$emit('change', 'landing')
    await flushPromises()

    const model = wrapper
      .findComponent({ name: 'ElForm' })
      .props('model') as Record<string, unknown>
    expect(model.template_code).toBe('landing')
    expect(model.settings).toEqual({ hero_title: '', columns: null })

    selects[2]?.vm.$emit('change', 'link')
    await flushPromises()
    expect(model.type).toBe('link')
    expect(model.template_code).toBe('')
    expect(model.settings).toEqual({})
    expect(ElMessageBox.confirm).toHaveBeenCalledTimes(2)
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
					settings: {},
				},
				permissions: { update: true, delete: true, restore: true },
			})
			.mockResolvedValueOnce({
				types: [
					{ code: 'page', label: 'Страница' },
					{ code: 'link', label: 'Ссылка' },
				],
				templates: [{ code: 'page', label: 'Страница', icon: 'document', fields: [] }],
				extensions: [{
					code: 'seo', title: 'SEO', applies_to: ['page'], fields: [], variables: [],
				}],
			})
			.mockResolvedValueOnce({ items: [] })
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
