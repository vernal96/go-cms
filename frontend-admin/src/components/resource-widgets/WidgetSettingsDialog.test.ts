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
    summary_fields: [],
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
        widget: null,
        siteId: 7,
        accessToken: 'token',
      },
      global: { renderStubDefaultSlot: true },
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
