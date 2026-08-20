// @vitest-environment jsdom

import { shallowMount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import type { ResourceWidget, WidgetDefinition } from '../../types/admin'
import WidgetCard from './WidgetCard.vue'

const definition: WidgetDefinition = {
  code: 'core_content',
  module_code: 'core',
  module_label: 'Core',
  module_description: 'Core widgets',
  label: 'Content',
  description: 'Resource content',
  fields: [],
  editor_tabs: [],
  summary_fields: [],
  views: [{ code: 'article', label: 'Статья' }],
}

const widget: ResourceWidget = {
  id: 1,
  code: 'core_content',
  area: 'body',
  position: 0,
  view: 'article',
  columns: 12,
  margin_top: 0,
  margin_bottom: 0,
  enabled: true,
  params: {},
}

describe('WidgetCard', () => {
  it('shows the declared custom view label instead of its storage code', () => {
    const wrapper = shallowMount(WidgetCard, {
      props: { widget, definition },
      global: { renderStubDefaultSlot: true },
    })

    expect(wrapper.text()).toContain('12/12 · Статья ·')
    expect(wrapper.text()).not.toContain('· article ·')
  })
})
