// @vitest-environment jsdom

import { shallowMount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import type { FieldDefinition, FieldEditorTab } from '../../types/admin'
import DynamicFieldsForm from './DynamicFieldsForm.vue'
import TabbedDynamicFieldsForm from './TabbedDynamicFieldsForm.vue'

const fields: FieldDefinition[] = [
  { key: 'title', type: 'string', label: 'Title', required: true, rules: [] },
  { key: 'color', type: 'string', label: 'Color', required: false, rules: [] },
]
const tabs: FieldEditorTab[] = [
  { code: 'content', label: 'Content', fields: ['title'] },
  { code: 'appearance', label: 'Appearance', fields: ['color'] },
]

describe('TabbedDynamicFieldsForm', () => {
  it('renders the flat form when tabs are absent', () => {
    const wrapper = shallowMount(TabbedDynamicFieldsForm, {
      props: { fields, modelValue: { title: '', color: '' } },
    })

    expect(wrapper.findComponent({ name: 'ElTabs' }).exists()).toBe(false)
    expect(wrapper.getComponent(DynamicFieldsForm).props('fields')).toEqual(fields)
  })

  it('renders vertical tabs, selects the first and partitions fields', async () => {
    const wrapper = shallowMount(TabbedDynamicFieldsForm, {
      props: {
        fields,
        editorTabs: tabs,
        modelValue: { title: '', color: '' },
      },
      global: { renderStubDefaultSlot: true },
    })
    const elementTabs = wrapper.getComponent({ name: 'ElTabs' })

    expect(elementTabs.props('tabPosition')).toBe('left')
    expect(elementTabs.props('modelValue')).toBe('content')
    expect(wrapper.findAllComponents(DynamicFieldsForm).map((form) =>
      form.props('fields').map((field: FieldDefinition) => field.key),
    )).toEqual([['title'], ['color']])

    elementTabs.vm.$emit('update:modelValue', 'appearance')
    await wrapper.vm.$nextTick()
    expect(elementTabs.props('modelValue')).toBe('appearance')
  })

  it('opens the first erroneous tab and resets for a new declaration', async () => {
    const wrapper = shallowMount(TabbedDynamicFieldsForm, {
      props: {
        fields,
        editorTabs: tabs,
        modelValue: { title: '', color: '' },
        errors: {},
      },
      global: { renderStubDefaultSlot: true },
    })
    const elementTabs = () => wrapper.getComponent({ name: 'ElTabs' })

    elementTabs().vm.$emit('update:modelValue', 'appearance')
    await wrapper.vm.$nextTick()
    await wrapper.setProps({ errors: { title: 'Required' } })
    expect(elementTabs().props('modelValue')).toBe('content')

    elementTabs().vm.$emit('update:modelValue', 'appearance')
    await wrapper.vm.$nextTick()
    await wrapper.setProps({ editorTabs: tabs.map((tab) => ({ ...tab })) })
    expect(elementTabs().props('modelValue')).toBe('content')
  })
})
