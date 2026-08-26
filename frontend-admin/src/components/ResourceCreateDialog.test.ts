// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { adminRequest } from '../api/admin-api'
import type { ResourceTreeItem } from '../types/admin'
import ResourceCreateDialog from './ResourceCreateDialog.vue'

vi.mock('../api/admin-api', () => ({ adminRequest: vi.fn() }))

const requestMock = vi.mocked(adminRequest)

describe('ResourceCreateDialog', () => {
  beforeEach(() => {
    requestMock.mockReset()
  })

  it('creates a root Page with the minimal supported payload', async () => {
	const created: ResourceTreeItem = {
		id: 10,
		version: 1,
      parent_id: null,
      template_code: null,
      icon: 'document',
      title: 'Home',
      menu_title: '',
      display_title: 'Home',
		sort: 0,
		deleted: false,
		published: true,
		deleted_at: null,
      has_children: false,
      can_create_child: true,
    }
    requestMock
      .mockResolvedValueOnce({
        types: [
			{ code: 'page', label: 'Страница', capabilities: { supports_template: true, supports_content: true, supports_fields: true, mutable_type: true }, settings_fields: [], settings_defaults: {}, content_types: [{ code: 'html', label: 'HTML', editor: 'html' }] },
			{ code: 'link', label: 'Ссылка', capabilities: { mutable_type: true }, settings_fields: [], settings_defaults: {}, content_types: [] },
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
                label: 'Заголовок',
                required: true,
                rules: [],
              },
            ],
          },
        ],
			extensions: [],
      })
      .mockResolvedValueOnce({ items: [] })
      .mockResolvedValueOnce(created)
    const wrapper = shallowMount(ResourceCreateDialog, {
      props: { accessToken: 'token', siteId: 7 },
      global: {
        renderStubDefaultSlot: true,
        stubs: {
          ElDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        },
      },
    })

    await (
      wrapper.vm as unknown as {
        open(parent: ResourceTreeItem | null): Promise<void>
      }
    ).open(null)
    await flushPromises()
    const model = wrapper
      .findComponent({ name: 'ElForm' })
      .props('model') as Record<string, unknown>
    Object.assign(model, {
      title: ' Home ',
      menu_title: '',
      slug: '',
		fields: {},
    })

    const buttons = wrapper.findAllComponents({ name: 'ElButton' })
    buttons[buttons.length - 1]?.vm.$emit('click')
    await flushPromises()

    expect(requestMock).toHaveBeenNthCalledWith(
      3,
      '/api/sites/7/resources',
      'token',
      expect.objectContaining({ method: 'POST' }),
    )
    const init = requestMock.mock.calls[2]?.[2] as RequestInit
    expect(JSON.parse(String(init.body))).toEqual({
      parent_id: null,
      type: 'page',
      template_code: null,
		content_type: 'html',
      content: '',
      target_resource_id: null,
      title: 'Home',
      menu_title: '',
      slug: '',
		fields: {},
		type_settings: {},
    })
    expect(wrapper.emitted('created')?.[0]).toEqual([created, null])
  })

  it('uses capabilities for a custom target resource type', async () => {
    requestMock
      .mockResolvedValueOnce({
        types: [{
			code: 'custom_target', label: 'Товар каталога',
			capabilities: { supports_target_resource: true, supports_content: true, mutable_type: true },
			settings_fields: [{ key: 'catalog_mode', type: 'string', label: 'Режим каталога', required: true, rules: [] }],
			settings_defaults: { catalog_mode: 'standard' },
			content_types: [{ code: 'markdown', label: 'Markdown', editor: 'textarea' }],
		}],
        templates: [], widgets: [], extensions: [],
      })
      .mockResolvedValueOnce({ items: [{ id: 12, parent_id: null, type: 'page', display_title: 'Target', path: '/target' }] })
      .mockResolvedValueOnce({ id: 13, title: 'Reference' })
    const wrapper = shallowMount(ResourceCreateDialog, {
      props: { accessToken: 'token', siteId: 7 },
      global: { renderStubDefaultSlot: true, stubs: { ElDialog: { template: '<div><slot /><slot name="footer" /></div>' } } },
    })
    await (wrapper.vm as unknown as { open(parent: ResourceTreeItem | null): Promise<void> }).open(null)
    await flushPromises()
    const model = wrapper.findComponent({ name: 'ElForm' }).props('model') as Record<string, unknown>
		expect(model.type_settings).toEqual({ catalog_mode: 'standard' })
		expect(model.content_type).toBe('markdown')
    Object.assign(model, { title: 'Reference', target_resource_id: 12 })
    const buttons = wrapper.findAllComponents({ name: 'ElButton' })
    buttons[buttons.length - 1]?.vm.$emit('click')
    await flushPromises()

    const init = requestMock.mock.calls[2]?.[2] as RequestInit
    expect(JSON.parse(String(init.body))).toEqual(expect.objectContaining({
		type: 'custom_target', target_resource_id: 12, template_code: null,
		content_type: 'markdown', type_settings: { catalog_mode: 'standard' },
    }))
    expect(wrapper.findAllComponents({ name: 'ElSelect' })).toHaveLength(2)
  })
})
