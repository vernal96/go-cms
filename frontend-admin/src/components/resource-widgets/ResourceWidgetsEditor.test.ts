// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { adminRequest } from '../../api/admin-api'
import type { ResourceTemplate, ResourceWidget, WidgetDefinition } from '../../types/admin'
import ResourceWidgetsEditor from './ResourceWidgetsEditor.vue'

vi.mock('../../api/admin-api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/admin-api')>()),
  adminRequest: vi.fn(),
  adminRequestVoid: vi.fn(),
}))

const requestMock = vi.mocked(adminRequest)
const template: ResourceTemplate = {
  code: 'page',
  label: 'Page',
  icon: 'document',
	fields: [],
	editor_tabs: [],
  supports_resource_widgets: true,
  widget_areas: ['body', 'sidebar'],
}
const definition: WidgetDefinition = {
  code: 'core_content',
  module_code: 'core',
  module_label: 'Core',
  module_description: '',
  label: 'Content',
  description: '',
  fields: [],
  editor_tabs: [],
  summary_fields: [],
  views: [],
}
const widget = (id: number, area: 'body' | 'sidebar', position: number): ResourceWidget => ({
  id,
  code: 'core_content',
  area,
  position,
  view: 'default',
  columns: 12,
  margin_top: 0,
  margin_bottom: 0,
  enabled: true,
  params: {},
})

function dragTransfer(): DataTransfer {
  const data = new Map<string, string>()
  const types: string[] = []
  return {
    dropEffect: 'none',
    effectAllowed: 'uninitialized',
    files: [],
    items: [],
    types,
    getData: (type: string) => data.get(type) ?? '',
    setData: (type: string, value: string) => {
      data.set(type, value)
      if (!types.includes(type)) types.push(type)
    },
  } as unknown as DataTransfer
}

function mountEditor(items: ResourceWidget[], canUpdate = true) {
  return mount(ResourceWidgetsEditor, {
    props: {
      accessToken: 'token',
      siteId: 7,
      resourceId: 9,
      template,
      definitions: [definition],
      modelValue: items,
      canUpdate,
    },
    global: {
      stubs: {
        WidgetPickerDialog: true,
        WidgetSettingsDialog: true,
      },
    },
  })
}

describe('ResourceWidgetsEditor drag and drop', () => {
  beforeEach(() => requestMock.mockReset())

  it('starts only from the handle and highlights the exact empty-area target', async () => {
    const wrapper = mountEditor([widget(41, 'body', 0)])
    const transfer = dragTransfer()
    const card = wrapper.find('.widget-card')
    const handle = wrapper.find('.widget-drag-handle')

    expect(card.attributes('draggable')).toBeUndefined()
    expect(handle.attributes('draggable')).toBe('true')
    await handle.trigger('dragstart', { dataTransfer: transfer })

    expect(transfer.types).toContain('application/x-go-cms-widget')
    expect(card.classes()).toContain('is-dragging')
    expect(wrapper.findAll('.widget-area.is-drag-available')).toHaveLength(1)

    const target = wrapper.find('.widget-empty-drop-target')
    await target.trigger('dragenter', { dataTransfer: transfer })
    expect(target.classes()).toContain('is-active')
    expect(target.element.closest('.widget-area')?.classList.contains('is-drop-area')).toBe(true)

    await handle.trigger('dragend', { dataTransfer: transfer })
    expect(card.classes()).not.toContain('is-dragging')
    expect(target.classes()).not.toContain('is-active')
  })

  it('locks duplicate drops while a cross-area reorder is being saved', async () => {
    let resolveRequest: ((value: { items: ResourceWidget[] }) => void) | undefined
    requestMock.mockImplementationOnce(() => new Promise((resolve) => { resolveRequest = resolve }))
    const source = [widget(41, 'body', 0), widget(42, 'body', 1)]
    const result = [widget(42, 'body', 0), widget(41, 'sidebar', 0)]
    const wrapper = mountEditor(source)
    const transfer = dragTransfer()

    await wrapper.findAll('.widget-drag-handle')[0]!.trigger('dragstart', { dataTransfer: transfer })
    const target = wrapper.find('.widget-empty-drop-target')
    await target.trigger('dragenter', { dataTransfer: transfer })
    await target.trigger('drop', { dataTransfer: transfer })
    await target.trigger('drop', { dataTransfer: transfer })

    expect(requestMock).toHaveBeenCalledTimes(1)
    expect(requestMock).toHaveBeenCalledWith(
      '/api/sites/7/resources/9/widgets/order',
      'token',
      {
        method: 'PUT',
        body: JSON.stringify({ expected_version: 1, items: [
          { id: 42, area: 'body', position: 0 },
          { id: 41, area: 'sidebar', position: 0 },
        ] }),
      },
    )
    expect(wrapper.findAll('.widget-area.is-reordering')).toHaveLength(2)
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([result])

    resolveRequest?.({ items: result })
    await flushPromises()
    expect(wrapper.find('.widget-area.is-reordering').exists()).toBe(false)
  })

  it('rolls back the optimistic order when persistence fails', async () => {
    requestMock.mockRejectedValueOnce(new Error('network down'))
    const source = [widget(41, 'body', 0), widget(77, 'sidebar', 0)]
    const wrapper = mountEditor(source)
    const transfer = dragTransfer()

    await wrapper.findAll('.widget-drag-handle')[0]!.trigger('dragstart', { dataTransfer: transfer })
    const sidebarTargets = wrapper.findAll('.widget-drop-target')
      .filter((target) => target.element.closest('.widget-area')?.querySelector('h3')?.textContent === 'Sidebar')
    await sidebarTargets[0]!.trigger('dragenter', { dataTransfer: transfer })
    await sidebarTargets[0]!.trigger('drop', { dataTransfer: transfer })
    await flushPromises()

    const updates = wrapper.emitted('update:modelValue') ?? []
    expect(updates).toHaveLength(2)
    expect(updates[1]).toEqual([source])
  })

  it('disables handle dragging without update permission', () => {
    const wrapper = mountEditor([widget(41, 'body', 0)], false)
    expect(wrapper.find('.widget-drag-handle').attributes('draggable')).toBe('false')
    expect(wrapper.find('.widget-area.is-drag-available').exists()).toBe(false)
  })
})
