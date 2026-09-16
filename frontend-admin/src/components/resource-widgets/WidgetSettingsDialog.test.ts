// @vitest-environment jsdom

import { shallowMount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import type { WidgetDefinition } from '../../types/admin'
import WidgetSettingsDialog from './WidgetSettingsDialog.vue'

function definition(tabs: WidgetDefinition['editor_tabs']): WidgetDefinition {
  return {
    code: 'settings_widget',
    module_code: 'test',
    module_label: 'Test',
    module_description: '',
    label: 'Settings',
    description: '',
    fields: [
      { key: 'title', type: 'string', label: 'Title', required: false, rules: [] },
      { key: 'color', type: 'string', label: 'Color', required: false, rules: [] },
    ],
    editor_tabs: tabs,
    summary_fields: [], param_types: {},
    views: [],
  }
}

describe('WidgetSettingsDialog tabs', () => {
  it('selects the first available tab and recovers when the active tab disappears', async () => {
    const wrapper = shallowMount(WidgetSettingsDialog, {
      props: {
        modelValue: true,
        definition: definition([
          { code: 'content', label: 'Content', fields: ['title'] },
          { code: 'appearance', label: 'Appearance', fields: ['color'] },
        ]),
        widget: null, sources: [],
        siteId: 7,
        accessToken: 'token',
      },
      global: { renderStubDefaultSlot: true, stubs: { ElDialog: { template: '<div><slot /><slot name="footer" /></div>' } } },
    })
    const tabs = () => wrapper.getComponent({ name: 'ElTabs' })
    expect(tabs().props('modelValue')).toBe('content')

    tabs().vm.$emit('update:modelValue', 'appearance')
    await wrapper.vm.$nextTick()
    expect(tabs().props('modelValue')).toBe('appearance')

    await wrapper.setProps({
      definition: definition([{ code: 'content', label: 'Content', fields: ['title'] }]),
    })
    expect(tabs().props('modelValue')).toBe('content')
  })
})

it('reopens a bound required parameter and saves the reference without a literal', async () => {
  const def = definition([])
  def.fields[0]!.required = true
  def.param_types = { title: { type: 'string', multiple: false } }
  const wrapper = shallowMount(WidgetSettingsDialog, {
    props: {
      modelValue: true, definition: def, siteId: 7, accessToken: 'token',
      sources: [{ kind: 'resource_property', key: 'title', label: 'Название', type: 'string', multiple: false }],
      widget: { id: 1, code: def.code, area: 'body', position: 0, view: 'default', columns: 12,
        margin_top: 0, margin_bottom: 0, enabled: true, params: {},
        param_bindings: { title: { kind: 'resource_property', key: 'title' } } },
    }, global: { renderStubDefaultSlot: true, stubs: { ElDialog: { template: '<div><slot /><slot name="footer" /></div>' } } },
  })
  const fields = wrapper.getComponent({ name: 'WidgetParamFields' })
  expect(fields.props('bindings')).toEqual({ title: { kind: 'resource_property', key: 'title' } })
  expect(fields.props('modelValue')).not.toHaveProperty('title')
  wrapper.findAllComponents({ name: 'ElButton' }).at(-1)!.vm.$emit('click')
  await wrapper.vm.$nextTick()
  const saved = wrapper.emitted('save')?.[0]?.[0] as Record<string, any>
  expect(saved.param_bindings).toEqual({ title: { kind: 'resource_property', key: 'title' } })
  expect(saved.params).not.toHaveProperty('title')
})

it('rejects a source that is no longer compatible with the widget field', async () => {
  const def = definition([])
  def.param_types = { title: { type: 'string', multiple: false } }
  const wrapper = shallowMount(WidgetSettingsDialog, {
    props: { modelValue: true, definition: def, widget: null, siteId: 7, accessToken: 'token', sources: [] },
    global: { renderStubDefaultSlot: true, stubs: { ElDialog: { template: '<div><slot /><slot name="footer" /></div>' } } },
  })
  wrapper.getComponent({ name: 'WidgetParamFields' }).vm.$emit('update:bindings', { title: { kind: 'resource_field', key: 'removed' } })
  await wrapper.vm.$nextTick()
  wrapper.findAllComponents({ name: 'ElButton' }).at(-1)!.vm.$emit('click')
  await wrapper.vm.$nextTick()
  expect(wrapper.emitted('save')).toBeUndefined()
  expect(wrapper.getComponent({ name: 'WidgetParamFields' }).props('errors').title).toContain('совместимое')
})
