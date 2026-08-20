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
                label: 'Заголовок',
                required: true,
                rules: [],
              },
            ],
          },
        ],
			extensions: [],
      })
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
      settings: {},
    })

    const buttons = wrapper.findAllComponents({ name: 'ElButton' })
    buttons[buttons.length - 1]?.vm.$emit('click')
    await flushPromises()

    expect(requestMock).toHaveBeenNthCalledWith(
      2,
      '/api/admin/sites/7/resources',
      'token',
      expect.objectContaining({ method: 'POST' }),
    )
    const init = requestMock.mock.calls[1]?.[2] as RequestInit
    expect(JSON.parse(String(init.body))).toEqual({
      parent_id: null,
      type: 'page',
      template_code: null,
      title: 'Home',
      menu_title: '',
      slug: '',
      settings: {},
    })
    expect(wrapper.emitted('created')?.[0]).toEqual([created, null])
  })
})
